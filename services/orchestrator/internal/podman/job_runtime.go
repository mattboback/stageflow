package podman

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mattboback/stageflow/libs/go/logging"
	"github.com/mattboback/stageflow/libs/go/models"
	"github.com/mattboback/stageflow/libs/go/scannercatalog"
)

type jobRuntimeClient interface {
	CreatePod(ctx context.Context, req *PodCreateRequest) (*PodCreateResponse, error)
	InspectPod(ctx context.Context, podID string) (*PodInfo, error)
	CreateVolume(ctx context.Context, name string) error
	InspectVolume(ctx context.Context, name string) (*VolumeInfo, error)
	CreateContainer(ctx context.Context, req *ContainerCreateRequest) (*ContainerCreateResponse, error)
	InspectContainer(ctx context.Context, containerID string) (*ContainerInfo, error)
	StartContainer(ctx context.Context, containerID string) error
}

type JobRuntimeConfig struct {
	DataVolume      string
	DirectoryOwner  func(string) error
	DataDir         string
	Client          jobRuntimeClient
	ExtractionImage string
	PodNetwork      string
	PodNetnsMode    string
	PodHostMappings []string
}

type JobRuntime struct {
	dataVolume      string
	directoryOwner  func(string) error
	dataDir         string
	client          jobRuntimeClient
	extractionImage string
	podNetwork      string
	podNetnsMode    string
	podHostMappings []string
}

type ContainerLaunchResult struct {
	ContainerID string
	Started     bool
	Existing    bool
}

func NewJobRuntime(config JobRuntimeConfig) *JobRuntime {
	podNetnsMode := config.PodNetnsMode
	if podNetnsMode == "" {
		podNetnsMode = PodNetnsModeBridge
	}

	if config.DataDir == "" {
		config.DataDir = "/data"
	}

	if config.DirectoryOwner == nil {
		config.DirectoryOwner = func(name string) error { return os.Chown(name, 65532, 65532) }
	}

	if config.DataVolume == "" {
		config.DataVolume = "stageflow_data"
	}

	return &JobRuntime{
		client:          config.Client,
		directoryOwner:  config.DirectoryOwner,
		dataDir:         config.DataDir,
		dataVolume:      config.DataVolume,
		extractionImage: config.ExtractionImage,
		podNetwork:      config.PodNetwork,
		podNetnsMode:    podNetnsMode,
		podHostMappings: config.PodHostMappings,
	}
}
func (r *JobRuntime) PodNetnsMode() string {
	if r == nil {
		return PodNetnsModeBridge
	}

	return r.podNetnsMode
}

func (r *JobRuntime) AllowsLoopbackTargets() bool {
	return r.PodNetnsMode() == PodNetnsModeHost
}

// JobPodName returns the deterministic pod name for a job. It is deterministic so
// that a pod can be found again by name when its create response was lost, and so
// that cleanup can remove it without knowing its ID.
func JobPodName(jobID string) string {
	return "job-" + jobID
}

func (r *JobRuntime) CreateJobPod(ctx context.Context, job *models.Job) (string, error) {
	if job == nil {
		return "", errors.New("job is nil")
	}

	podReq := &PodCreateRequest{
		Name: JobPodName(job.ID),
		Labels: map[string]string{
			"managed_by": "orchestrator",
			"job_id":     job.ID,
		},
		Netns:   PodNetns{Nsmode: r.PodNetnsMode()},
		HostAdd: append([]string(nil), r.podHostMappings...),
	}

	if r.PodNetnsMode() == PodNetnsModeBridge && r.podNetwork != "" {
		podReq.Networks = map[string]PerNetworkOptions{
			r.podNetwork: {},
		}
	}

	return r.ensurePod(ctx, podReq)
}

// ensurePod creates a pod, recovering the one Podman may already have made.
//
// This mirrors ensureContainerStarted below, for the same reason: the pod name is
// deterministic, so a create whose response never arrived leaves a pod that every
// later attempt then collides with. In production a `pods/create` call exceeded
// the client's response-header timeout, Podman created the pod anyway, and every
// retry that followed got 409 "pod already exists", so the job never left
// PENDING.
//
// Two deliberate differences from the container path:
//
//   - No pre-inspect. A container is inspected first because the caller must
//     choose between adopting, starting, and skipping it. A pod yields only an ID,
//     so the extra round-trip would buy nothing on the happy path.
//   - Recovery runs on *any* create error, not just a 409. The failure that
//     started this was a transport timeout, not an API status, so gating on 409
//     would still lose the first attempt.
func (r *JobRuntime) ensurePod(ctx context.Context, req *PodCreateRequest) (string, error) {
	if req == nil || req.Name == "" {
		return "", errors.New("pod name is required")
	}

	resp, err := retryPodCreate(ctx, r.client, req)
	if err == nil {
		if resp == nil || resp.ID == "" {
			return "", errors.New("created pod has no ID")
		}

		return resp.ID, nil
	}

	// Podman may have created the deterministic pod even when the response was
	// lost, or another claimant may have won the create race. Inspecting by name
	// makes both cases idempotent.
	if existing, inspectErr := r.client.InspectPod(ctx, req.Name); inspectErr == nil {
		return adoptPod(existing, req.Labels)
	}

	return "", fmt.Errorf("failed to create pod: %w", err)
}

// adoptPod accepts an existing pod only when it carries the labels we would have
// set ourselves, so a name collision with something else is refused rather than
// hijacked. A pod needs no state check: unlike a container it runs nothing on its
// own, and Podman accepts new containers into a created or stopped pod.
func adoptPod(pod *PodInfo, expectedLabels map[string]string) (string, error) {
	if pod == nil || pod.ID == "" {
		return "", errors.New("inspected pod has no ID")
	}

	for _, key := range []string{"managed_by", "job_id"} {
		expected := expectedLabels[key]
		if expected == "" {
			continue
		}

		if pod.Labels[key] != expected {
			return "", fmt.Errorf(
				"refusing to adopt pod %s: label %s=%q, want %q",
				pod.ID,
				key,
				pod.Labels[key],
				expected,
			)
		}
	}

	return pod.ID, nil
}

// The extractor's limits. It streams the archive to disk and serves static
// files, so it needs little; see TestJobMemoryFitsBudget.
const (
	extractorMemoryMB = 256
	extractorCPUs     = 1
)

func (r *JobRuntime) StartExtractionWorker(
	ctx context.Context,
	job *models.Job,
	podID string,
) (*ContainerLaunchResult, error) {
	if job == nil {
		return nil, errors.New("job is nil")
	}

	jobSource, err := r.jobSource(ctx, job.ID)
	if err != nil {
		return nil, err
	}

	req := &ContainerCreateRequest{
		Name:  "extraction-worker-" + job.ID,
		Image: r.extractionImage,
		Pod:   podID,
		Env:   r.extractionEnv(ctx, job),
		Mounts: []VolumeMount{
			{
				Type:        "bind",
				Source:      filepath.Join(jobSource, "input.zip"),
				Destination: "/job/input.zip",
				ReadOnly:    true,
			},
			{Type: "bind", Source: filepath.Join(jobSource, "site"), Destination: "/site"},
			{Type: "bind", Source: filepath.Join(jobSource, ".provenance"), Destination: "/provenance"},
		},
		User: "65532",
		// The extractor unpacks the ZIP, then serves the site to every scanner
		// until the job ends, so its limits count alongside theirs.
		ResourceLimits: &ResourceLimits{
			MemoryLimitMB: extractorMemoryMB,
			MemorySwapMB:  extractorMemoryMB,
			CPUQuota:      cpuQuota(extractorCPUs),
			CPUPeriod:     cpuPeriod,
		},

		Labels: map[string]string{
			"managed_by": "orchestrator",
			"job_id":     job.ID,
			"component":  "extraction-worker",
		},
	}

	return r.ensureContainerStarted(
		ctx,
		req,
		"failed to create extraction worker container",
		"failed to start extraction worker container",
	)
}

func (r *JobRuntime) ensureContainerStarted(
	ctx context.Context,
	req *ContainerCreateRequest,
	createErrMessage, startErrMessage string,
) (*ContainerLaunchResult, error) {
	if req == nil || req.Name == "" {
		return nil, errors.New("container name is required")
	}

	if existing, inspectErr := r.client.InspectContainer(ctx, req.Name); inspectErr == nil {
		return r.startExistingContainer(ctx, existing, req.Labels, startErrMessage)
	} else if !isAPIStatus(inspectErr, 404) {
		return nil, fmt.Errorf("failed to inspect existing container %s: %w", req.Name, inspectErr)
	}

	containerResp, err := retryContainerCreate(ctx, r.client, req)
	if err == nil {
		if containerResp == nil || containerResp.ID == "" {
			return nil, errors.New("created container has no ID")
		}

		return r.startCreatedContainer(ctx, containerResp.ID, startErrMessage)
	}

	// Podman may have created the deterministic container even when the
	// response was lost, or another claimant may have won the create race.
	// Inspecting by name makes both cases idempotent.
	existing, inspectErr := r.client.InspectContainer(ctx, req.Name)
	if inspectErr == nil {
		return r.startExistingContainer(ctx, existing, req.Labels, startErrMessage)
	}

	return nil, fmt.Errorf("%s: %w", createErrMessage, err)
}

func (r *JobRuntime) startExistingContainer(
	ctx context.Context,
	container *ContainerInfo,
	expectedLabels map[string]string,
	startErrMessage string,
) (*ContainerLaunchResult, error) {
	if container == nil || container.ID == "" {
		return nil, errors.New("inspected container has no ID")
	}

	actualLabels := container.labels()

	for _, key := range []string{"managed_by", "job_id", "component", "scanner_type"} {
		expected := expectedLabels[key]
		if expected == "" {
			continue
		}

		if actualLabels[key] != expected {
			return nil, fmt.Errorf(
				"refusing to adopt container %s: label %s=%q, want %q",
				container.ID,
				key,
				actualLabels[key],
				expected,
			)
		}
	}

	result := &ContainerLaunchResult{ContainerID: container.ID, Existing: true}
	if isTerminalContainerState(container) {
		// A scanner may have finished and written its durable outcome just
		// before the orchestrator exited. Adopt the container, but never rerun
		// it: the reconciler settles the scanner from the outcome on disk.
		return result, nil
	}

	if err := r.client.StartContainer(ctx, container.ID); err != nil {
		refreshed, inspectErr := r.client.InspectContainer(ctx, container.ID)
		if inspectErr != nil || refreshed.State != ContainerState("running") {
			return result, fmt.Errorf("%s: %w", startErrMessage, err)
		}
	}

	result.Started = true

	return result, nil
}

func isTerminalContainerState(container *ContainerInfo) bool {
	state := strings.ToLower(strings.TrimSpace(string(container.State)))
	if state == "exited" || state == "stopped" || state == "dead" {
		return true
	}

	status := strings.ToLower(strings.TrimSpace(container.Status))

	return strings.HasPrefix(status, "exited") || strings.HasPrefix(status, "stopped")
}

func (r *JobRuntime) startCreatedContainer(
	ctx context.Context,
	containerID, startErrMessage string,
) (*ContainerLaunchResult, error) {
	result := &ContainerLaunchResult{ContainerID: containerID}
	if err := r.client.StartContainer(ctx, containerID); err != nil {
		container, inspectErr := r.client.InspectContainer(ctx, containerID)
		if inspectErr != nil || container.State != ContainerState("running") {
			return result, fmt.Errorf("%s: %w", startErrMessage, err)
		}
	}

	result.Started = true

	return result, nil
}

func (r *JobRuntime) StartScanner(
	ctx context.Context,
	job *models.Job,
	plan *ScannerLaunchPlan,
) (*ContainerLaunchResult, error) {
	if job == nil {
		return nil, errors.New("job is nil")
	}

	if plan == nil {
		return nil, errors.New("scanner launch plan is required")
	}

	scannerType := plan.Labels["scanner_type"]

	jobSource, err := r.jobSource(ctx, job.ID)
	if err != nil {
		return nil, err
	}

	if publishURLProvenanceErr := r.publishURLProvenance(ctx, job); publishURLProvenanceErr != nil {
		return nil, publishURLProvenanceErr
	}

	mounts, err := r.resolveVolumeMounts(ctx, plan.Volumes)
	if err != nil {
		return nil, err
	}

	if _, statErr := os.Stat(filepath.Join(r.dataDir, "jobs", job.ID, "auth.json")); statErr == nil {
		mounts = append(
			mounts,
			VolumeMount{
				Type:        "bind",
				Source:      filepath.Join(jobSource, "auth.json"),
				Destination: "/run/stageflow/storage-state.json",
				ReadOnly:    true,
			},
		)
	}

	req := &ContainerCreateRequest{
		Name:   plan.Name,
		Image:  plan.Image,
		Pod:    job.PodID,
		User:   plan.User,
		Env:    plan.Env,
		Mounts: mounts,
		Labels: plan.Labels,
		ResourceLimits: &ResourceLimits{
			MemoryLimitMB: plan.ResourceLimits.MemoryLimitMB,
			MemorySwapMB:  plan.ResourceLimits.MemorySwapMB,
			CPUQuota:      plan.ResourceLimits.CPUQuota,
			CPUPeriod:     cpuPeriod,
		},
	}

	return r.ensureContainerStarted(
		ctx,
		req,
		fmt.Sprintf("failed to create %s scanner container", scannerType),
		fmt.Sprintf("failed to start %s scanner container", scannerType),
	)
}

func (r *JobRuntime) extractionEnv(ctx context.Context, job *models.Job) map[string]string {
	env := map[string]string{
		"JOB_ID":          job.ID,
		"INPUT_PATH":      "/job/input.zip",
		"SITE_DIR":        "/site/content",
		"PROVENANCE_PATH": "/provenance/provenance.json",
		"HOME":            "/tmp",

		"PORT": "8080",
	}

	if requestID := logging.RequestID(ctx); requestID != "" {
		env["REQUEST_ID"] = requestID
	}

	if runID := logging.RunID(ctx); runID != "" {
		env["RUN_ID"] = runID
	}

	return env
}

func (r *JobRuntime) jobSource(ctx context.Context, jobID string) (string, error) {
	volume, err := r.ensureVolume(ctx, r.dataVolume)
	if err != nil {
		return "", err
	}

	for _, name := range append([]string{"", "site", ".provenance"}, scannercatalog.IDs()...) {
		dir := filepath.Join(r.dataDir, "jobs", jobID, name)
		if mkdirAllErr := os.MkdirAll(dir, 0o700); mkdirAllErr != nil {
			return "", mkdirAllErr
		}

		if directoryOwnerErr := r.directoryOwner(dir); directoryOwnerErr != nil {
			return "", directoryOwnerErr
		}
	}

	return filepath.Join(volume.Mountpoint, "jobs", jobID), nil
}

func (r *JobRuntime) ensureVolume(ctx context.Context, name string) (*VolumeInfo, error) {
	if info, err := r.client.InspectVolume(ctx, name); err == nil {
		return info, nil
	}

	if err := r.client.CreateVolume(ctx, name); err != nil {
		return nil, err
	}

	return r.client.InspectVolume(ctx, name)
}

func (r *JobRuntime) resolveVolumeMounts(
	ctx context.Context,
	volumes []VolumeRequirement,
) ([]VolumeMount, error) {
	mounts := make([]VolumeMount, 0, len(volumes))

	for _, volume := range volumes {
		volumeInfo, err := r.ensureVolume(ctx, volume.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare %s volume: %w", volume.Name, err)
		}

		mounts = append(mounts, VolumeMount{
			Type:        "bind",
			Source:      filepath.Join(volumeInfo.Mountpoint, volume.Subpath),
			Destination: volume.Destination,
			ReadOnly:    volume.ReadOnly,
		})
	}

	return mounts, nil
}

// Create retries are bounded within one persisted launch attempt. Deterministic
// names and the existing inspect fallback recover a lost response safely.
func retryPodCreate(ctx context.Context, client jobRuntimeClient, req *PodCreateRequest) (*PodCreateResponse, error) {
	var (
		response *PodCreateResponse
		err      error
	)
	for attempt := range 3 {
		response, err = client.CreatePod(ctx, req)
		if err == nil || isAPIStatus(err, 409) {
			return response, err
		}

		if attempt < 2 {
			if pauseErr := retryPause(ctx, attempt); pauseErr != nil {
				return nil, pauseErr
			}
		}
	}

	return response, err
}

func retryContainerCreate(
	ctx context.Context,
	client jobRuntimeClient,
	req *ContainerCreateRequest,
) (*ContainerCreateResponse, error) {
	var (
		response *ContainerCreateResponse
		err      error
	)
	for attempt := range 3 {
		response, err = client.CreateContainer(ctx, req)
		if err == nil || isAPIStatus(err, 409) {
			return response, err
		}

		if attempt < 2 {
			if pauseErr := retryPause(ctx, attempt); pauseErr != nil {
				return nil, pauseErr
			}
		}
	}

	return response, err
}
func retryPause(ctx context.Context, attempt int) error {
	timer := time.NewTimer(time.Duration(attempt+1) * 100 * time.Millisecond)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
