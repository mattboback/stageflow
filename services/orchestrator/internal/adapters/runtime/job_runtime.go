package podman

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/mattboback/stageflow/libs/go/logging"
	"github.com/mattboback/stageflow/libs/go/models"
	scanners "github.com/mattboback/stageflow/libs/go/scannerregistry"
	"github.com/mattboback/stageflow/libs/go/storage"
	appjobs "github.com/mattboback/stageflow/services/orchestrator/internal/application/jobs"
)

type jobRuntimeClient interface {
	CreatePod(ctx context.Context, req *PodCreateRequest) (*PodCreateResponse, error)
	CreateVolume(ctx context.Context, name string) error
	InspectVolume(ctx context.Context, name string) (*VolumeInfo, error)
	CreateContainer(ctx context.Context, req *ContainerCreateRequest) (*ContainerCreateResponse, error)
	InspectContainer(ctx context.Context, containerID string) (*ContainerInfo, error)
	StartContainer(ctx context.Context, containerID string) error
}

type JobRuntimeConfig struct {
	Client          jobRuntimeClient
	ScannerRegistry *scanners.Registry
	ExtractionImage string
	PodNetwork      string
	PodNetnsMode    string
	PodHostMappings []string
	NatsHost        string
	MinioHost       string
	MinioAccessKey  string
	MinioSecretKey  string
	MinioUseSSL     bool
}

type JobRuntime struct {
	client          jobRuntimeClient
	scannerRegistry *scanners.Registry
	extractionImage string
	podNetwork      string
	podNetnsMode    string
	podHostMappings []string
	natsHost        string
	minioHost       string
	minioAccessKey  string
	minioSecretKey  string
	minioUseSSL     bool
}

type ContainerLaunchResult struct {
	ContainerID string
	Started     bool
	Existing    bool
}

func NewJobRuntime(config JobRuntimeConfig) *JobRuntime {
	podNetnsMode := config.PodNetnsMode
	if podNetnsMode == "" {
		podNetnsMode = appjobs.PodNetnsModeBridge
	}

	natsHost := config.NatsHost
	if natsHost == "" {
		natsHost = "nats"
	}

	minioHost := config.MinioHost
	if minioHost == "" {
		minioHost = "minio"
	}

	return &JobRuntime{
		client:          config.Client,
		scannerRegistry: config.ScannerRegistry,
		extractionImage: config.ExtractionImage,
		podNetwork:      config.PodNetwork,
		podNetnsMode:    podNetnsMode,
		podHostMappings: append([]string(nil), config.PodHostMappings...),
		natsHost:        natsHost,
		minioHost:       minioHost,
		minioAccessKey:  config.MinioAccessKey,
		minioSecretKey:  config.MinioSecretKey,
		minioUseSSL:     config.MinioUseSSL,
	}
}

func (r *JobRuntime) PodNetnsMode() string {
	if r == nil {
		return appjobs.PodNetnsModeBridge
	}

	return r.podNetnsMode
}

func (r *JobRuntime) AllowsLoopbackTargets() bool {
	return r.PodNetnsMode() == appjobs.PodNetnsModeHost
}

func (r *JobRuntime) ResolveScannerTypes(modules []string) []string {
	if r == nil || r.scannerRegistry == nil {
		return append([]string(nil), modules...)
	}

	return r.scannerRegistry.ResolveModules(modules)
}

func (r *JobRuntime) CreateJobPod(ctx context.Context, job *models.Job) (string, error) {
	if job == nil {
		return "", errors.New("job is nil")
	}

	podReq := &PodCreateRequest{
		Name: "job-" + job.ID,
		Labels: map[string]string{
			"managed_by": "orchestrator",
			"job_id":     job.ID,
		},
		Netns:   PodNetns{Nsmode: r.PodNetnsMode()},
		HostAdd: append([]string(nil), r.podHostMappings...),
	}

	if r.PodNetnsMode() == appjobs.PodNetnsModeBridge && r.podNetwork != "" {
		podReq.Networks = map[string]PerNetworkOptions{
			r.podNetwork: {},
		}
	}

	podResp, err := r.client.CreatePod(ctx, podReq)
	if err != nil {
		return "", fmt.Errorf("failed to create pod: %w", err)
	}

	return podResp.ID, nil
}

func (r *JobRuntime) StartExtractionWorker(
	ctx context.Context,
	job *models.Job,
	podID string,
) (*ContainerLaunchResult, error) {
	if job == nil {
		return nil, errors.New("job is nil")
	}

	workspaceVolume, err := r.ensureVolume(ctx, "workspace-"+job.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare workspace volume: %w", err)
	}

	req := &ContainerCreateRequest{
		Name:  "extraction-worker-" + job.ID,
		Image: r.extractionImage,
		Pod:   podID,
		Env:   r.extractionEnv(ctx, job),
		Mounts: []VolumeMount{
			{
				Type:        "bind",
				Source:      workspaceVolume.Mountpoint,
				Destination: "/workspace",
				// The extractor image runs as a non-root user; the fresh
				// volume mountpoint is owned by container-root, so have
				// Podman chown it to the extractor at start.
				ChownToContainerUser: true,
			},
		},
		Labels: map[string]string{
			"managed_by": "orchestrator",
			"job_id":     job.ID,
			"component":  "extraction-worker",
		},
	}

	return r.createAndStartContainer(
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

	containerResp, err := r.client.CreateContainer(ctx, req)
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
		// A scanner may have finished and published its durable NATS event just
		// before the orchestrator exited. Reattach the monitor, but never rerun
		// the completed container and emit duplicate scan events.
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
	plan *appjobs.ScannerLaunchPlan,
) (*ContainerLaunchResult, error) {
	if job == nil {
		return nil, errors.New("job is nil")
	}

	if plan == nil {
		return nil, errors.New("scanner launch plan is required")
	}

	scannerType := plan.Labels["scanner_type"]

	mounts, err := r.resolveVolumeMounts(ctx, plan.Volumes)
	if err != nil {
		return nil, err
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
	natsURL, minioEndpoint := r.serviceEndpoints()
	env := map[string]string{
		"JOB_ID":                job.ID,
		"INPUT_PATH":            job.InputPath,
		"NATS_URL":              natsURL,
		"MINIO_ENDPOINT":        minioEndpoint,
		"MINIO_ACCESS_KEY":      r.minioAccessKey,
		"MINIO_SECRET_KEY":      r.minioSecretKey,
		"MINIO_USE_SSL":         strconv.FormatBool(r.minioUseSSL),
		"WORKSPACE":             "/workspace",
		"PORT":                  "8080",
		"MINIO_ARTIFACT_BUCKET": storage.BucketArtifacts,
	}

	if requestID := logging.RequestID(ctx); requestID != "" {
		env["REQUEST_ID"] = requestID
	}

	if runID := logging.RunID(ctx); runID != "" {
		env["RUN_ID"] = runID
	}

	return env
}

func (r *JobRuntime) serviceEndpoints() (string, string) {
	if r.PodNetnsMode() == appjobs.PodNetnsModeHost {
		return appjobs.HostNetnsNATSURL, appjobs.HostNetnsMinioEndpoint
	}

	return "nats://" + r.natsHost + ":4222", r.minioHost + ":9000"
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
	volumes []appjobs.VolumeRequirement,
) ([]VolumeMount, error) {
	mounts := make([]VolumeMount, 0, len(volumes))

	for _, volume := range volumes {
		volumeInfo, err := r.ensureVolume(ctx, volume.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare %s volume: %w", volume.Name, err)
		}

		mounts = append(mounts, VolumeMount{
			Type:        "bind",
			Source:      volumeInfo.Mountpoint,
			Destination: volume.Destination,
			ReadOnly:    volume.ReadOnly,
		})
	}

	return mounts, nil
}

func (r *JobRuntime) createAndStartContainer(
	ctx context.Context,
	req *ContainerCreateRequest,
	createErrMessage, startErrMessage string,
) (*ContainerLaunchResult, error) {
	containerResp, err := r.client.CreateContainer(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", createErrMessage, err)
	}

	result := &ContainerLaunchResult{ContainerID: containerResp.ID}
	if startErr := r.client.StartContainer(ctx, containerResp.ID); startErr != nil {
		return result, fmt.Errorf("%s: %w", startErrMessage, startErr)
	}

	result.Started = true

	return result, nil
}
