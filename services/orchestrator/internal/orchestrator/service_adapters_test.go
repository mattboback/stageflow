package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/models"
	scanners "github.com/mattboback/stageflow/libs/go/scannerregistry"
	db "github.com/mattboback/stageflow/platform/orchestrator/internal/adapters/repository"
	podman "github.com/mattboback/stageflow/platform/orchestrator/internal/adapters/runtime"
)

func TestOrchestratorRuntimeCreateJobPodDelegatesWithoutRecordingInternalEvents(t *testing.T) {
	orch, database, _, _ := setupTestOrchestratorWithConfig(t, func(config *Config) {
		config.PodmanClient = &mockPodmanClient{
			createPodFunc: func(_ context.Context, _ *podman.PodCreateRequest) (*podman.PodCreateResponse, error) {
				return &podman.PodCreateResponse{ID: "pod-123"}, nil
			},
		}
	})
	defer orch.WaitForMonitors()

	job := &models.Job{
		ID:        "job-runtime-pod",
		State:     models.JobStatePending,
		InputType: models.JobInputTypeZip,
		InputPath: "staging/job-runtime-pod/site.zip",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	insertJob(t, database, job)

	podID, err := (orchestratorRuntime{orchestrator: orch}).CreateJobPod(context.Background(), job)
	if err != nil {
		t.Fatalf("CreateJobPod() error = %v", err)
	}

	if podID != "pod-123" {
		t.Fatalf("CreateJobPod() podID = %q, want %q", podID, "pod-123")
	}

	events, err := database.ListJobEvents(context.Background(), job.ID, db.ListJobEventsOptions{Limit: 10})
	if err != nil {
		t.Fatalf("ListJobEvents() error = %v", err)
	}

	if len(events) != 0 {
		t.Fatalf("ListJobEvents() returned %d events, want 0", len(events))
	}
}

func TestHandleExtractionReadyUsesConfiguredScannerLaunchPlanner(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "test-openrouter-key")
	t.Setenv("OPENROUTER_APP_TITLE", "StageFlow Test")
	t.Setenv("OPENROUTER_APP_REFERER", "https://example.com")

	registry := scanners.NewRegistry("localhost/stageflow/scanner-runner:latest")
	if err := registry.Register(&scanners.Definition{
		ID:      "ai-navigator",
		Name:    "AI Navigator",
		Enabled: true,
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	var gotReq *podman.ContainerCreateRequest

	orch, database, _, _ := setupTestOrchestratorWithConfig(t, func(config *Config) {
		config.PodNetnsMode = podNetnsModeHost
		config.ScannerRegistry = registry
		config.PodmanClient = &mockPodmanClient{
			inspectVolumeFunc: func(_ context.Context, name string) (*podman.VolumeInfo, error) {
				return &podman.VolumeInfo{Name: name, Mountpoint: "/volumes/" + name}, nil
			},
			createContainerFunc: func(_ context.Context, req *podman.ContainerCreateRequest) (*podman.ContainerCreateResponse, error) {
				gotReq = req
				return &podman.ContainerCreateResponse{ID: "scanner-ctr"}, nil
			},
			startContainerFunc: func(_ context.Context, _ string) error { return nil },
			waitContainerFunc: func(_ context.Context, _ string) (*podman.ContainerWaitResponse, error) {
				return &podman.ContainerWaitResponse{StatusCode: 0}, nil
			},
		}
	})
	defer orch.WaitForMonitors()

	job := &models.Job{
		ID:        "job-configured-planner",
		State:     models.JobStateExtracting,
		InputType: models.JobInputTypeZip,
		InputPath: "staging/job-configured-planner/site.zip",
		PodID:     "pod-123",
		Config: models.JobConfig{
			Modules: []string{"ai-navigator"},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	insertJob(t, database, job)

	err := orch.HandleExtractionReady(context.Background(), &events.ExtractionReadyPayload{
		JobID:                  job.ID,
		TotalPages:             1,
		ProvenancePath:         "/workspace/provenance.json",
		ProvenanceArtifactPath: job.ID + "/provenance.json",
	})
	if err != nil {
		t.Fatalf("HandleExtractionReady() error = %v", err)
	}

	if gotReq == nil {
		t.Fatal("expected scanner container create request")
	}

	if got, want := gotReq.Env["NATS_URL"], hostNetnsNATSURL; got != want {
		t.Fatalf("NATS_URL = %q, want %q", got, want)
	}

	if got, want := gotReq.Env["MINIO_ENDPOINT"], hostNetnsMinioEndpoint; got != want {
		t.Fatalf("MINIO_ENDPOINT = %q, want %q", got, want)
	}

	if got, want := gotReq.Env["OPENROUTER_API_KEY"], "test-openrouter-key"; got != want {
		t.Fatalf("OPENROUTER_API_KEY = %q, want %q", got, want)
	}
}

func TestHandleExtractionReadyUsesRegistryDefaultScannerImageWhenNoOverride(t *testing.T) {
	registry := scanners.NewRegistry("registry/default:latest")
	if err := registry.Register(&scanners.Definition{
		ID:      "axe",
		Name:    "axe",
		Enabled: true,
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	var gotReq *podman.ContainerCreateRequest

	orch, database, _, _ := setupTestOrchestratorWithConfig(t, func(config *Config) {
		config.ScannerRegistry = registry
		config.ScannerImage = "localhost/stageflow/scanner-runner:latest"
		config.PodmanClient = &mockPodmanClient{
			inspectVolumeFunc: func(_ context.Context, name string) (*podman.VolumeInfo, error) {
				return &podman.VolumeInfo{Name: name, Mountpoint: "/volumes/" + name}, nil
			},
			createContainerFunc: func(_ context.Context, req *podman.ContainerCreateRequest) (*podman.ContainerCreateResponse, error) {
				gotReq = req
				return &podman.ContainerCreateResponse{ID: "scanner-ctr"}, nil
			},
			startContainerFunc: func(_ context.Context, _ string) error { return nil },
			waitContainerFunc: func(_ context.Context, _ string) (*podman.ContainerWaitResponse, error) {
				return &podman.ContainerWaitResponse{StatusCode: 0}, nil
			},
		}
	})
	defer orch.WaitForMonitors()

	job := &models.Job{
		ID:        "job-registry-default-image",
		State:     models.JobStateExtracting,
		InputType: models.JobInputTypeZip,
		InputPath: "staging/job-registry-default-image/site.zip",
		PodID:     "pod-123",
		Config: models.JobConfig{
			Modules: []string{"axe"},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	insertJob(t, database, job)

	err := orch.HandleExtractionReady(context.Background(), &events.ExtractionReadyPayload{
		JobID:                  job.ID,
		TotalPages:             1,
		ProvenancePath:         "/workspace/provenance.json",
		ProvenanceArtifactPath: job.ID + "/provenance.json",
	})
	if err != nil {
		t.Fatalf("HandleExtractionReady() error = %v", err)
	}

	if gotReq == nil {
		t.Fatal("expected scanner container create request")
	}

	if got, want := gotReq.Image, "registry/default:latest"; got != want {
		t.Fatalf("scanner image = %q, want %q", got, want)
	}
}

func TestHandleExtractionReadyUsesExplicitScannerImageOverride(t *testing.T) {
	registry := scanners.NewRegistry("registry/default:latest")
	if err := registry.Register(&scanners.Definition{
		ID:      "axe",
		Name:    "axe",
		Enabled: true,
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	var gotReq *podman.ContainerCreateRequest

	orch, database, _, _ := setupTestOrchestratorWithConfig(t, func(config *Config) {
		config.ScannerRegistry = registry
		config.ScannerImage = "explicit/override:latest"
		config.ScannerImageOverride = "explicit/override:latest"
		config.PodmanClient = &mockPodmanClient{
			inspectVolumeFunc: func(_ context.Context, name string) (*podman.VolumeInfo, error) {
				return &podman.VolumeInfo{Name: name, Mountpoint: "/volumes/" + name}, nil
			},
			createContainerFunc: func(_ context.Context, req *podman.ContainerCreateRequest) (*podman.ContainerCreateResponse, error) {
				gotReq = req
				return &podman.ContainerCreateResponse{ID: "scanner-ctr"}, nil
			},
			startContainerFunc: func(_ context.Context, _ string) error { return nil },
			waitContainerFunc: func(_ context.Context, _ string) (*podman.ContainerWaitResponse, error) {
				return &podman.ContainerWaitResponse{StatusCode: 0}, nil
			},
		}
	})
	defer orch.WaitForMonitors()

	job := &models.Job{
		ID:        "job-scanner-image-override",
		State:     models.JobStateExtracting,
		InputType: models.JobInputTypeZip,
		InputPath: "staging/job-scanner-image-override/site.zip",
		PodID:     "pod-123",
		Config: models.JobConfig{
			Modules: []string{"axe"},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	insertJob(t, database, job)

	err := orch.HandleExtractionReady(context.Background(), &events.ExtractionReadyPayload{
		JobID:                  job.ID,
		TotalPages:             1,
		ProvenancePath:         "/workspace/provenance.json",
		ProvenanceArtifactPath: job.ID + "/provenance.json",
	})
	if err != nil {
		t.Fatalf("HandleExtractionReady() error = %v", err)
	}

	if gotReq == nil {
		t.Fatal("expected scanner container create request")
	}

	if got, want := gotReq.Image, "explicit/override:latest"; got != want {
		t.Fatalf("scanner image = %q, want %q", got, want)
	}
}
