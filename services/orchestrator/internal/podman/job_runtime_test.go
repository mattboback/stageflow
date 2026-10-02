package podman

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/mattboback/stageflow/libs/go/models"
)

type fakeJobRuntimeClient struct {
	createPodFunc        func(context.Context, *PodCreateRequest) (*PodCreateResponse, error)
	inspectPodFunc       func(context.Context, string) (*PodInfo, error)
	createVolumeFunc     func(context.Context, string) error
	inspectVolumeFunc    func(context.Context, string) (*VolumeInfo, error)
	createContainerFunc  func(context.Context, *ContainerCreateRequest) (*ContainerCreateResponse, error)
	inspectContainerFunc func(context.Context, string) (*ContainerInfo, error)
	startContainerFunc   func(context.Context, string) error
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

func (f *fakeJobRuntimeClient) InspectPod(ctx context.Context, podID string) (*PodInfo, error) {
	if f.inspectPodFunc != nil {
		return f.inspectPodFunc(ctx, podID)
	}

	return nil, &APIError{StatusCode: 404, Body: "pod not found"}
}

func (f *fakeJobRuntimeClient) InspectContainer(ctx context.Context, containerID string) (*ContainerInfo, error) {
	if f.inspectContainerFunc != nil {
		return f.inspectContainerFunc(ctx, containerID)
	}

	return nil, &APIError{StatusCode: 404, Body: "container not found"}
}

func TestJobRuntimeCreateJobPodBuildsPodmanRequest(t *testing.T) {
	var gotReq *PodCreateRequest

	runtime := NewJobRuntime(JobRuntimeConfig{DataDir: t.TempDir(), DirectoryOwner: func(string) error { return nil },
		Client: &fakeJobRuntimeClient{
			createPodFunc: func(_ context.Context, req *PodCreateRequest) (*PodCreateResponse, error) {
				gotReq = req

				return &PodCreateResponse{ID: "pod-abc"}, nil
			},
		},
		PodNetnsMode:    PodNetnsModeBridge,
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

	if gotReq.Netns.Nsmode != PodNetnsModeBridge {
		t.Fatalf("netns mode = %q, want %q", gotReq.Netns.Nsmode, PodNetnsModeBridge)
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

	runtime := NewJobRuntime(JobRuntimeConfig{DataDir: t.TempDir(), DirectoryOwner: func(string) error { return nil },
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
		PodNetnsMode:    PodNetnsModeHost,
		ExtractionImage: "extractor:latest",
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

	if !reflect.DeepEqual(createVolumeCalls, []string{"stageflow_data"}) {
		t.Fatalf("createVolume calls = %#v", createVolumeCalls)
	}

	if inspectCalls["stageflow_data"] != 2 {
		t.Fatalf("inspectVolume call count = %d, want 2", inspectCalls["stageflow_data"])
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

	assertExtractorMountContract(t, gotReq)

	if len(gotReq.Mounts) != 3 {
		t.Fatalf("mount count = %d, want 1", len(gotReq.Mounts))
	}

	mount := gotReq.Mounts[0]
	if mount.Source != "/volumes/stageflow_data/jobs/job-123/input.zip" || mount.Destination != "/job/input.zip" ||
		mount.Type != "bind" {
		t.Fatalf("unexpected workspace mount: %#v", mount)
	}

	if mount.ChownToContainerUser || !mount.ReadOnly {
		t.Fatalf("input mount must remain scoped and read-only: %+v", mount)
	}
}

func TestJobRuntimeStartExtractionWorkerUsesExistingWorkspaceVolumeWhenInspectSucceeds(t *testing.T) {
	var (
		createVolumeCalls  []string
		inspectVolumeCalls []string
		gotReq             *ContainerCreateRequest
	)

	runtime := NewJobRuntime(JobRuntimeConfig{DataDir: t.TempDir(), DirectoryOwner: func(string) error { return nil },
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

	if !reflect.DeepEqual(inspectVolumeCalls, []string{"stageflow_data"}) {
		t.Fatalf("inspectVolume calls = %#v", inspectVolumeCalls)
	}

	if len(createVolumeCalls) != 0 {
		t.Fatalf("createVolume calls = %#v, want none", createVolumeCalls)
	}

	if gotReq == nil {
		t.Fatal("expected container create request")
	}

	if len(gotReq.Mounts) != 3 {
		t.Fatalf("mount count = %d, want 1", len(gotReq.Mounts))
	}

	if mount := gotReq.Mounts[0]; mount.Source != "/existing/stageflow_data/jobs/job-123/input.zip" ||
		mount.Destination != "/job/input.zip" {
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

	runtime := NewJobRuntime(JobRuntimeConfig{DataDir: t.TempDir(), DirectoryOwner: func(string) error { return nil },
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

	plan := &ScannerLaunchPlan{
		Name:   "scanner-axe-job-123",
		Image:  "scanner:latest",
		User:   "0",
		Env:    map[string]string{"SCAN_URLS": `["https://example.com"]`},
		Labels: map[string]string{"scanner_type": "axe", "component": "scanner"},
		Volumes: []VolumeRequirement{
			{Name: "stageflow_data", Destination: "/workspace", ReadOnly: true},
			{Name: "results-job-123", Destination: "/results"},
		},
		ResourceLimits: ScannerResourceLimits{MemoryLimitMB: 512, MemorySwapMB: 512},
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
	if workspaceMount.Source != "/volumes/stageflow_data" ||
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

func TestJobRuntimeStartScannerAdoptsDeterministicExistingContainer(t *testing.T) {
	var (
		createCalls int
		startedID   string
	)

	runtime := NewJobRuntime(JobRuntimeConfig{DataDir: t.TempDir(), DirectoryOwner: func(string) error { return nil },
		Client: &fakeJobRuntimeClient{
			inspectContainerFunc: func(_ context.Context, name string) (*ContainerInfo, error) {
				if name != "scanner-axe-job-restart" {
					t.Fatalf("InspectContainer() name = %q", name)
				}

				return &ContainerInfo{
					ID:    "existing-axe",
					Name:  name,
					State: "running",
					Labels: map[string]string{
						"managed_by":   "orchestrator",
						"job_id":       "job-restart",
						"component":    "scanner",
						"scanner_type": "axe",
					},
				}, nil
			},
			createContainerFunc: func(_ context.Context, _ *ContainerCreateRequest) (*ContainerCreateResponse, error) {
				createCalls++

				return nil, errors.New("must not create duplicate container")
			},
			startContainerFunc: func(_ context.Context, id string) error {
				startedID = id

				return nil
			},
		},
	})

	job := &models.Job{ID: "job-restart", PodID: "pod-restart"}
	plan := &ScannerLaunchPlan{
		Name:  "scanner-axe-job-restart",
		Image: "scanner:latest",
		Labels: map[string]string{
			"managed_by":   "orchestrator",
			"job_id":       "job-restart",
			"component":    "scanner",
			"scanner_type": "axe",
		},
	}

	result, err := runtime.StartScanner(t.Context(), job, plan)
	if err != nil {
		t.Fatalf("StartScanner() error = %v", err)
	}

	if createCalls != 0 {
		t.Fatalf("CreateContainer() calls = %d, want 0", createCalls)
	}

	if startedID != "existing-axe" {
		t.Fatalf("StartContainer() ID = %q, want existing-axe", startedID)
	}

	if result == nil || result.ContainerID != "existing-axe" || !result.Existing || !result.Started {
		t.Fatalf("StartScanner() result = %#v", result)
	}
}

func TestJobRuntimeStartScannerRejectsContainerWithWrongOwnershipLabels(t *testing.T) {
	runtime := NewJobRuntime(JobRuntimeConfig{DataDir: t.TempDir(), DirectoryOwner: func(string) error { return nil },
		Client: &fakeJobRuntimeClient{
			inspectContainerFunc: func(_ context.Context, name string) (*ContainerInfo, error) {
				return &ContainerInfo{
					ID:    "unrelated-container",
					Name:  name,
					State: "running",
					Labels: map[string]string{
						"managed_by":   "someone-else",
						"job_id":       "job-restart",
						"component":    "scanner",
						"scanner_type": "axe",
					},
				}, nil
			},
		},
	})

	job := &models.Job{ID: "job-restart", PodID: "pod-restart"}
	plan := &ScannerLaunchPlan{
		Name:  "scanner-axe-job-restart",
		Image: "scanner:latest",
		Labels: map[string]string{
			"managed_by":   "orchestrator",
			"job_id":       "job-restart",
			"component":    "scanner",
			"scanner_type": "axe",
		},
	}

	_, err := runtime.StartScanner(t.Context(), job, plan)
	if err == nil {
		t.Fatal("StartScanner() adopted a container with mismatched ownership labels")
	}

	if !strings.Contains(err.Error(), "refusing to adopt") {
		t.Fatalf("StartScanner() error = %v", err)
	}
}

func TestJobRuntimeStartScannerDoesNotRestartExitedContainer(t *testing.T) {
	startCalls := 0
	runtime := NewJobRuntime(JobRuntimeConfig{DataDir: t.TempDir(), DirectoryOwner: func(string) error { return nil },
		Client: &fakeJobRuntimeClient{
			inspectContainerFunc: func(_ context.Context, name string) (*ContainerInfo, error) {
				return &ContainerInfo{
					ID:    "completed-axe",
					Name:  name,
					State: "exited",
					Labels: map[string]string{
						"managed_by":   "orchestrator",
						"job_id":       "job-restart",
						"component":    "scanner",
						"scanner_type": "axe",
					},
				}, nil
			},
			startContainerFunc: func(_ context.Context, _ string) error {
				startCalls++

				return nil
			},
		},
	})

	job := &models.Job{ID: "job-restart", PodID: "pod-restart"}
	plan := &ScannerLaunchPlan{
		Name:  "scanner-axe-job-restart",
		Image: "scanner:latest",
		Labels: map[string]string{
			"managed_by":   "orchestrator",
			"job_id":       "job-restart",
			"component":    "scanner",
			"scanner_type": "axe",
		},
	}

	result, err := runtime.StartScanner(t.Context(), job, plan)
	if err != nil {
		t.Fatalf("StartScanner() error = %v", err)
	}

	if startCalls != 0 {
		t.Fatalf("StartContainer() calls = %d, want 0", startCalls)
	}

	if result == nil || result.ContainerID != "completed-axe" || !result.Existing || result.Started {
		t.Fatalf("StartScanner() result = %#v", result)
	}
}

// The production incident these cover: `pods/create` exceeded the client's
// response-header timeout, Podman created the pod anyway, and every redelivery
// then collided with the name until the job was abandoned in PENDING.

func TestCreateJobPodAdoptsPodPodmanCreatedAfterALostResponse(t *testing.T) {
	inspected := 0
	runtime := NewJobRuntime(JobRuntimeConfig{DataDir: t.TempDir(), DirectoryOwner: func(string) error { return nil },
		Client: &fakeJobRuntimeClient{
			// A transport timeout, not an API status: this is why recovery cannot
			// be gated on a 409.
			createPodFunc: func(_ context.Context, _ *PodCreateRequest) (*PodCreateResponse, error) {
				return nil, errors.New("net/http: timeout awaiting response headers")
			},
			inspectPodFunc: func(_ context.Context, name string) (*PodInfo, error) {
				inspected++

				return &PodInfo{
					ID:   "pod-recovered",
					Name: name,
					Labels: map[string]string{
						"managed_by": "orchestrator",
						"job_id":     "job-123",
					},
				}, nil
			},
		},
	})

	podID, err := runtime.CreateJobPod(context.Background(), &models.Job{ID: "job-123"})
	if err != nil {
		t.Fatalf("CreateJobPod error: %v", err)
	}

	if podID != "pod-recovered" {
		t.Fatalf("podID = %q, want %q", podID, "pod-recovered")
	}

	if inspected != 1 {
		t.Fatalf("InspectPod calls = %d, want 1", inspected)
	}
}

func TestCreateJobPodAdoptsPodOnConflict(t *testing.T) {
	runtime := NewJobRuntime(JobRuntimeConfig{DataDir: t.TempDir(), DirectoryOwner: func(string) error { return nil },
		Client: &fakeJobRuntimeClient{
			createPodFunc: func(_ context.Context, _ *PodCreateRequest) (*PodCreateResponse, error) {
				return nil, &APIError{StatusCode: 409, Body: `{"cause":"pod already exists"}`}
			},
			inspectPodFunc: func(_ context.Context, name string) (*PodInfo, error) {
				return &PodInfo{
					ID:   "pod-existing",
					Name: name,
					Labels: map[string]string{
						"managed_by": "orchestrator",
						"job_id":     "job-123",
					},
				}, nil
			},
		},
	})

	podID, err := runtime.CreateJobPod(context.Background(), &models.Job{ID: "job-123"})
	if err != nil {
		t.Fatalf("CreateJobPod error: %v", err)
	}

	if podID != "pod-existing" {
		t.Fatalf("podID = %q, want %q", podID, "pod-existing")
	}
}

func TestCreateJobPodRefusesToAdoptAPodItDoesNotOwn(t *testing.T) {
	runtime := NewJobRuntime(JobRuntimeConfig{DataDir: t.TempDir(), DirectoryOwner: func(string) error { return nil },
		Client: &fakeJobRuntimeClient{
			createPodFunc: func(_ context.Context, _ *PodCreateRequest) (*PodCreateResponse, error) {
				return nil, &APIError{StatusCode: 409, Body: "pod already exists"}
			},
			inspectPodFunc: func(_ context.Context, name string) (*PodInfo, error) {
				return &PodInfo{
					ID:     "pod-someone-else",
					Name:   name,
					Labels: map[string]string{"managed_by": "orchestrator", "job_id": "a-different-job"},
				}, nil
			},
		},
	})

	_, err := runtime.CreateJobPod(context.Background(), &models.Job{ID: "job-123"})
	if err == nil {
		t.Fatal("expected an error rather than adopting a pod belonging to another job")
	}

	if !strings.Contains(err.Error(), "refusing to adopt pod") {
		t.Fatalf("error = %v, want it to explain the refusal", err)
	}
}

func TestCreateJobPodReportsTheCreateFailureWhenNoPodExists(t *testing.T) {
	createErr := errors.New("podman socket unavailable")
	runtime := NewJobRuntime(JobRuntimeConfig{DataDir: t.TempDir(), DirectoryOwner: func(string) error { return nil },
		Client: &fakeJobRuntimeClient{
			createPodFunc: func(_ context.Context, _ *PodCreateRequest) (*PodCreateResponse, error) {
				return nil, createErr
			},
			// Default inspect returns 404, so there is nothing to adopt.
		},
	})

	_, err := runtime.CreateJobPod(context.Background(), &models.Job{ID: "job-123"})
	if err == nil {
		t.Fatal("expected an error")
	}

	// The create failure is the useful diagnosis; the 404 from the recovery probe
	// would only describe the symptom.
	if !errors.Is(err, createErr) {
		t.Fatalf("error = %v, want it to wrap the original create error", err)
	}
}

func TestCreateJobPodDoesNotInspectOnTheHappyPath(t *testing.T) {
	inspected := 0
	runtime := NewJobRuntime(JobRuntimeConfig{DataDir: t.TempDir(), DirectoryOwner: func(string) error { return nil },
		Client: &fakeJobRuntimeClient{
			inspectPodFunc: func(_ context.Context, _ string) (*PodInfo, error) {
				inspected++

				return nil, &APIError{StatusCode: 404}
			},
		},
	})

	if _, err := runtime.CreateJobPod(context.Background(), &models.Job{ID: "job-123"}); err != nil {
		t.Fatalf("CreateJobPod error: %v", err)
	}

	if inspected != 0 {
		t.Fatalf("InspectPod calls = %d, want 0: a successful create needs no recovery probe", inspected)
	}
}

func TestJobPodNameIsDeterministic(t *testing.T) {
	// Cleanup relies on reconstructing this name when no pod ID was recorded.
	if got := JobPodName("abc"); got != "job-abc" {
		t.Fatalf("JobPodName = %q, want %q", got, "job-abc")
	}
}

func assertExtractorMountContract(t *testing.T, gotReq *ContainerCreateRequest) {
	t.Helper()

	if gotReq.Env["INPUT_PATH"] != "/job/input.zip" || gotReq.Env["SITE_DIR"] != "/site/content" {
		t.Fatalf("extractor input/output contract: %#v", gotReq.Env)
	}

	for _, obsolete := range []string{"ZIP_PATH", "EXTRACT_DIR"} {
		if _, ok := gotReq.Env[obsolete]; ok {
			t.Fatalf("obsolete extractor environment %s", obsolete)
		}
	}
	// Atomic extraction stages and renames siblings. The destination therefore
	// must be a child of its writable mount, never the mountpoint itself.
	writableParent := ""

	for _, mount := range gotReq.Mounts {
		if !mount.ReadOnly && filepath.Dir(gotReq.Env["SITE_DIR"]) == mount.Destination {
			writableParent = mount.Destination
		}
	}

	if writableParent == "" {
		t.Fatal("atomic extraction has no writable parent mount")
	}
}
