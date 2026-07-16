package orchestrator

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mattboback/stageflow/libs/go/models"
	db "github.com/mattboback/stageflow/services/orchestrator/internal/adapters/repository"
	podman "github.com/mattboback/stageflow/services/orchestrator/internal/adapters/runtime"
)

func TestStartReconcilesContainerCreatedBeforeOrchestratorCrash(t *testing.T) {
	var (
		createCalls atomic.Int32
		startCalls  atomic.Int32
	)

	orch, database, _, _ := setupTestOrchestratorWithConfig(t, func(config *Config) {
		config.PodmanClient = &mockPodmanClient{
			inspectVolumeFunc: func(_ context.Context, name string) (*podman.VolumeInfo, error) {
				return &podman.VolumeInfo{Name: name, Mountpoint: "/volumes/" + name}, nil
			},
			inspectContainerFunc: func(_ context.Context, name string) (*podman.ContainerInfo, error) {
				if name != "scanner-axe-job-crash" {
					return nil, &podman.APIError{StatusCode: 404, Body: "container not found"}
				}

				return &podman.ContainerInfo{
					ID:    "container-created-before-crash",
					Name:  name,
					State: "running",
					Labels: map[string]string{
						"managed_by":   "orchestrator",
						"job_id":       "job-crash",
						"component":    "scanner",
						"scanner_type": "axe",
					},
				}, nil
			},
			createContainerFunc: func(_ context.Context, _ *podman.ContainerCreateRequest) (*podman.ContainerCreateResponse, error) {
				createCalls.Add(1)

				return &podman.ContainerCreateResponse{ID: "duplicate"}, nil
			},
			startContainerFunc: func(_ context.Context, containerID string) error {
				if containerID != "container-created-before-crash" {
					t.Fatalf("StartContainer() ID = %q", containerID)
				}

				startCalls.Add(1)

				return nil
			},
			waitContainerFunc: func(_ context.Context, _ string) (*podman.ContainerWaitResponse, error) {
				return &podman.ContainerWaitResponse{StatusCode: 0}, nil
			},
		}
	})

	insertJob(t, database, &models.Job{
		ID:        "job-crash",
		State:     models.JobStateScanning,
		InputType: models.JobInputTypeURLs,
		URLs:      []string{"https://example.com"},
		PodID:     "pod-crash",
		Config:    models.JobConfig{Modules: []string{"axe"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	if err := database.UpdateJobPodID(t.Context(), "job-crash", "pod-crash"); err != nil {
		t.Fatalf("UpdateJobPodID() error = %v", err)
	}

	if err := database.PrepareScannerLaunches(t.Context(), "job-crash", []string{"axe"}); err != nil {
		t.Fatalf("PrepareScannerLaunches() error = %v", err)
	}

	claimed, err := database.ClaimScannerLaunch(t.Context(), "job-crash", "axe")
	if err != nil || !claimed {
		t.Fatalf("ClaimScannerLaunch() = (%v, %v), want (true, nil)", claimed, err)
	}
	// Simulate the process dying after Podman created/started the deterministic
	// container but before MarkScannerLaunched persisted its ID.

	ctx, cancel := context.WithCancel(t.Context())
	orch.Start(ctx)
	cancel()
	orch.WaitForMonitors()

	if createCalls.Load() != 0 {
		t.Fatalf("CreateContainer() calls = %d, want 0", createCalls.Load())
	}

	if startCalls.Load() != 1 {
		t.Fatalf("StartContainer() calls = %d, want 1", startCalls.Load())
	}

	launch, err := database.GetScannerLaunch(t.Context(), "job-crash", "axe")
	if err != nil {
		t.Fatalf("GetScannerLaunch() error = %v", err)
	}

	if launch.State != db.ScannerLaunchLaunched || launch.ContainerID != "container-created-before-crash" {
		t.Fatalf("recovered launch = %#v", launch)
	}

	if launch.AttemptCount != 2 {
		t.Fatalf("attempt count = %d, want 2", launch.AttemptCount)
	}
}
