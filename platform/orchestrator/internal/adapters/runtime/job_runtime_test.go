package podman

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/mattboback/stageflow/packages/shared-go/models"
	appjobs "github.com/mattboback/stageflow/platform/orchestrator/internal/application/jobs"
)

type fakeJobRuntimeClient struct {
	createPodFunc       func(context.Context, *PodCreateRequest) (*PodCreateResponse, error)
	createVolumeFunc    func(context.Context, string) error
	inspectVolumeFunc   func(context.Context, string) (*VolumeInfo, error)
	createContainerFunc func(context.Context, *ContainerCreateRequest) (*ContainerCreateResponse, error)
	startContainerFunc  func(context.Context, string) error
}

func (f *fakeJobRuntimeClient) CreatePod(ctx context.Context, req *PodCreateRequest) (*PodCreateResponse, error) {
	if f.createPodFunc != nil {
		return f.createPodFunc(ctx, req)
	}

	return &PodCreateResponse{ID: "pod-123"}, nil
}

func (f *fakeJobRuntimeClient) CreateVolume(ctx context.Context, name string) error {
	if f.createVolumeFunc != nil {
		return f.createVolumeFunc(ctx, name)
	}

	return nil
}

func (f *fakeJobRuntimeClient) InspectVolume(ctx context.Context, name string) (*VolumeInfo, error) {
	if f.inspectVolumeFunc != nil {
		return f.inspectVolumeFunc(ctx, name)
	}

	return &VolumeInfo{Name: name, Mountpoint: "/volumes/" + name}, nil
}

func (f *fakeJobRuntimeClient) CreateContainer(
	ctx context.Context,
	req *ContainerCreateRequest,
) (*ContainerCreateResponse, error) {
	if f.createContainerFunc != nil {
		return f.createContainerFunc(ctx, req)
	}

	return &ContainerCreateResponse{ID: "container-123"}, nil
}

func (f *fakeJobRuntimeClient) StartContainer(ctx context.Context, containerID string) error {
	if f.startContainerFunc != nil {
		return f.startContainerFunc(ctx, containerID)
	}

	return nil
}

func TestJobRuntimeCreateJobPodBuildsPodmanRequest(t *testing.T) {
	var gotReq *PodCreateRequest

	runtime := NewJobRuntime(JobRuntimeConfig{
		Client: &fakeJobRuntimeClient{
			createPodFunc: func(_ context.Context, req *PodCreateRequest) (*PodCreateResponse, error) {
				gotReq = req

				return &PodCreateResponse{ID: "pod-abc"}, nil
			},
		},
		PodNetnsMode:    appjobs.PodNetnsModeBridge,
		PodNetwork:      "job-network",
		PodHostMappings: []string{"app.local:169.254.1.2"},
	})

	podID, err := runtime.CreateJobPod(context.Background(), &models.Job{ID: "job-123"})
	if err != nil {
		t.Fatalf("CreateJobPod error: %v", err)
	}

	if podID != "pod-abc" {
		t.Fatalf("CreateJobPod podID = %q, want %q", podID, "pod-abc")
	}

	if gotReq == nil {
		t.Fatal("expected pod create request")
	}

	if gotReq.Name != "job-job-123" {
		t.Fatalf("pod name = %q, want %q", gotReq.Name, "job-job-123")
	}

	if gotReq.Labels["job_id"] != "job-123" || gotReq.Labels["managed_by"] != "orchestrator" {
		t.Fatalf("unexpected labels: %#v", gotReq.Labels)
	}

	if gotReq.Netns.Nsmode != appjobs.PodNetnsModeBridge {
		t.Fatalf("netns mode = %q, want %q", gotReq.Netns.Nsmode, appjobs.PodNetnsModeBridge)
	}

	if !reflect.DeepEqual(gotReq.Networks, map[string]PerNetworkOptions{"job-network": {}}) {
		t.Fatalf("networks = %#v, want job-network bridge config", gotReq.Networks)
	}

	if !reflect.DeepEqual(gotReq.HostAdd, []string{"app.local:169.254.1.2"}) {
		t.Fatalf("host mappings = %#v", gotReq.HostAdd)
	}
}

//nolint:gocyclo
func TestJobRuntimeStartExtractionWorkerCreatesMissingWorkspaceVolumeAfterInspectMiss(t *testing.T) {
	var (
		missingVolumeErr  = errors.New("missing volume")
		createVolumeCalls []string
		inspectCalls      = map[string]int{}
		gotReq            *ContainerCreateRequest
		startedID         string
	)

	runtime := NewJobRuntime(JobRuntimeConfig{
		Client: &fakeJobRuntimeClient{
			createVolumeFunc: func(_ context.Context, name string) error {
				createVolumeCalls = append(createVolumeCalls, name)
				return nil
			},
			inspectVolumeFunc: func(_ context.Context, name string) (*VolumeInfo, error) {
				inspectCalls[name]++
				if inspectCalls[name] == 1 {
					return nil, missingVolumeErr
				}

				return &VolumeInfo{Name: name, Mountpoint: "/volumes/" + name}, nil
			},
			createContainerFunc: func(_ context.Context, req *ContainerCreateRequest) (*ContainerCreateResponse, error) {
				gotReq = req
				return &ContainerCreateResponse{ID: "ctr-extract"}, nil
			},
			startContainerFunc: func(_ context.Context, containerID string) error {
				startedID = containerID
				return nil
			},
		},
		PodNetnsMode:    appjobs.PodNetnsModeHost,
		ExtractionImage: "extractor:latest",
		NatsHost:        "nats.internal",
		MinioHost:       "minio.internal",
		MinioAccessKey:  "minio-access",
		MinioSecretKey:  "minio-secret",
		MinioUseSSL:     true,
	})

	result, err := runtime.StartExtractionWorker(context.Background(), &models.Job{
		ID:        "job-123",
		InputPath: "uploads/input.zip",
	}, "pod-1")
	if err != nil {
		t.Fatalf("StartExtractionWorker error: %v", err)
	}

	if result == nil || result.ContainerID != "ctr-extract" || !result.Started {
		t.Fatalf("unexpected result: %#v", result)
	}

	if !reflect.DeepEqual(createVolumeCalls, []string{"workspace-job-123"}) {
		t.Fatalf("createVolume calls = %#v", createVolumeCalls)
	}

	if inspectCalls["workspace-job-123"] != 2 {
		t.Fatalf("inspectVolume call count = %d, want 2", inspectCalls["workspace-job-123"])
	}

	if startedID != "ctr-extract" {
		t.Fatalf("start container id = %q, want %q", startedID, "ctr-extract")
	}

	if gotReq == nil {
		t.Fatal("expected container create request")
	}

	if gotReq.Name != "extraction-worker-job-123" {
		t.Fatalf("container name = %q", gotReq.Name)
	}

	if gotReq.Image != "extractor:latest" {
		t.Fatalf("container image = %q", gotReq.Image)
	}

	if gotReq.Pod != "pod-1" {
		t.Fatalf("container pod = %q", gotReq.Pod)
	}

	if gotReq.Env["NATS_URL"] != appjobs.HostNetnsNATSURL {
		t.Fatalf("NATS_URL = %q, want %q", gotReq.Env["NATS_URL"], appjobs.HostNetnsNATSURL)
	}

	if gotReq.Env["MINIO_ENDPOINT"] != appjobs.HostNetnsMinioEndpoint {
		t.Fatalf("MINIO_ENDPOINT = %q, want %q", gotReq.Env["MINIO_ENDPOINT"], appjobs.HostNetnsMinioEndpoint)
	}

	if gotReq.Env["MINIO_USE_SSL"] != "true" {
		t.Fatalf("MINIO_USE_SSL = %q, want true", gotReq.Env["MINIO_USE_SSL"])
	}

	if len(gotReq.Mounts) != 1 {
		t.Fatalf("mount count = %d, want 1", len(gotReq.Mounts))
	}

	mount := gotReq.Mounts[0]
	if mount.Source != "/volumes/workspace-job-123" || mount.Destination != "/workspace" || mount.Type != "bind" {
		t.Fatalf("unexpected workspace mount: %#v", mount)
	}
}

func TestJobRuntimeStartExtractionWorkerUsesExistingWorkspaceVolumeWhenInspectSucceeds(t *testing.T) {
	var (
		createVolumeCalls  []string
		inspectVolumeCalls []string
		gotReq             *ContainerCreateRequest
	)

	runtime := NewJobRuntime(JobRuntimeConfig{
		Client: &fakeJobRuntimeClient{
			createVolumeFunc: func(_ context.Context, name string) error {
				createVolumeCalls = append(createVolumeCalls, name)
				t.Fatalf("CreateVolume should not be called when InspectVolume succeeds for %q", name)

				return nil
			},
			inspectVolumeFunc: func(_ context.Context, name string) (*VolumeInfo, error) {
				inspectVolumeCalls = append(inspectVolumeCalls, name)

				return &VolumeInfo{Name: name, Mountpoint: "/existing/" + name}, nil
			},
			createContainerFunc: func(_ context.Context, req *ContainerCreateRequest) (*ContainerCreateResponse, error) {
				gotReq = req
				return &ContainerCreateResponse{ID: "container-123"}, nil
			},
		},
		ExtractionImage: "extractor:latest",
		NatsHost:        "nats.internal",
		MinioHost:       "minio.internal",
	})

	result, err := runtime.StartExtractionWorker(context.Background(), &models.Job{
		ID:        "job-123",
		InputPath: "uploads/input.zip",
	}, "pod-1")
	if err != nil {
		t.Fatalf("StartExtractionWorker error: %v", err)
	}

	if result == nil || result.ContainerID != "container-123" || !result.Started {
		t.Fatalf("unexpected result: %#v", result)
	}

	if !reflect.DeepEqual(inspectVolumeCalls, []string{"workspace-job-123"}) {
		t.Fatalf("inspectVolume calls = %#v", inspectVolumeCalls)
	}

	if len(createVolumeCalls) != 0 {
		t.Fatalf("createVolume calls = %#v, want none", createVolumeCalls)
	}

	if gotReq == nil {
		t.Fatal("expected container create request")
	}

	if len(gotReq.Mounts) != 1 {
		t.Fatalf("mount count = %d, want 1", len(gotReq.Mounts))
	}

	if mount := gotReq.Mounts[0]; mount.Source != "/existing/workspace-job-123" || mount.Destination != "/workspace" {
		t.Fatalf("unexpected workspace mount: %#v", mount)
	}
}

//nolint:gocyclo
func TestJobRuntimeStartScannerPreparesVolumeMounts(t *testing.T) {
	missingVolumeErr := errors.New("missing volume")

	var (
		inspectCalls      = map[string]int{}
		createVolumeCalls []string
		gotReq            *ContainerCreateRequest
		startedID         string
	)

	runtime := NewJobRuntime(JobRuntimeConfig{
		Client: &fakeJobRuntimeClient{
			inspectVolumeFunc: func(_ context.Context, name string) (*VolumeInfo, error) {
				inspectCalls[name]++
				if name == "results-job-123" && inspectCalls[name] == 1 {
					return nil, missingVolumeErr
				}

				return &VolumeInfo{Name: name, Mountpoint: "/volumes/" + name}, nil
			},
			createVolumeFunc: func(_ context.Context, name string) error {
				createVolumeCalls = append(createVolumeCalls, name)
				return nil
			},
			createContainerFunc: func(_ context.Context, req *ContainerCreateRequest) (*ContainerCreateResponse, error) {
				gotReq = req
				return &ContainerCreateResponse{ID: "ctr-scan"}, nil
			},
			startContainerFunc: func(_ context.Context, containerID string) error {
				startedID = containerID
				return nil
			},
		},
	})

	plan := &appjobs.ScannerLaunchPlan{
		Name:   "scanner-axe-job-123",
		Image:  "scanner:latest",
		User:   "0",
		Env:    map[string]string{"SCAN_URLS": `["https://example.com"]`},
		Labels: map[string]string{"scanner_type": "axe", "component": "scanner"},
		Volumes: []appjobs.VolumeRequirement{
			{Name: "workspace-job-123", Destination: "/workspace", ReadOnly: true},
			{Name: "results-job-123", Destination: "/results"},
		},
		ResourceLimits: appjobs.ResourceLimits{MemoryLimitMB: 512, MemorySwapMB: 512},
	}

	result, err := runtime.StartScanner(context.Background(), &models.Job{ID: "job-123", PodID: "pod-1"}, plan)
	if err != nil {
		t.Fatalf("StartScanner error: %v", err)
	}

	if result == nil || result.ContainerID != "ctr-scan" || !result.Started {
		t.Fatalf("unexpected result: %#v", result)
	}

	if !reflect.DeepEqual(createVolumeCalls, []string{"results-job-123"}) {
		t.Fatalf("createVolume calls = %#v", createVolumeCalls)
	}

	if startedID != "ctr-scan" {
		t.Fatalf("start container id = %q, want %q", startedID, "ctr-scan")
	}

	if gotReq == nil {
		t.Fatal("expected container create request")
	}

	if gotReq.Name != plan.Name || gotReq.Image != plan.Image || gotReq.Pod != "pod-1" || gotReq.User != plan.User {
		t.Fatalf("unexpected container identity: %#v", gotReq)
	}

	if len(gotReq.Mounts) != 2 {
		t.Fatalf("mount count = %d, want 2", len(gotReq.Mounts))
	}

	workspaceMount := gotReq.Mounts[0]
	if workspaceMount.Source != "/volumes/workspace-job-123" ||
		workspaceMount.Destination != "/workspace" ||
		!workspaceMount.ReadOnly {
		t.Fatalf("unexpected workspace mount: %#v", workspaceMount)
	}

	resultsMount := gotReq.Mounts[1]
	if resultsMount.Source != "/volumes/results-job-123" ||
		resultsMount.Destination != "/results" ||
		resultsMount.ReadOnly {
		t.Fatalf("unexpected results mount: %#v", resultsMount)
	}

	if gotReq.ResourceLimits == nil ||
		gotReq.ResourceLimits.MemoryLimitMB != 512 ||
		gotReq.ResourceLimits.MemorySwapMB != 512 {
		t.Fatalf("unexpected resource limits: %#v", gotReq.ResourceLimits)
	}
}
