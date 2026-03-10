package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/mattboback/stageflow/packages/shared-go/models"
	podman "github.com/mattboback/stageflow/platform/orchestrator/internal/adapters/runtime"
)

func TestHandleURLJob_Success(t *testing.T) {
	orch, database, _, _ := setupTestOrchestrator(t)
	defer orch.WaitForMonitors()

	mockClient := &mockPodmanClient{
		createPodFunc: func(_ context.Context, _ *podman.PodCreateRequest) (*podman.PodCreateResponse, error) {
			return &podman.PodCreateResponse{ID: "pod-123"}, nil
		},
		inspectVolumeFunc: func(_ context.Context, name string) (*podman.VolumeInfo, error) {
			return &podman.VolumeInfo{Name: name, Mountpoint: "/volumes/" + name}, nil
		},
		createContainerFunc: func(_ context.Context, _ *podman.ContainerCreateRequest) (*podman.ContainerCreateResponse, error) {
			return &podman.ContainerCreateResponse{ID: "container-123"}, nil
		},
		startContainerFunc: func(_ context.Context, _ string) error {
			return nil
		},
		waitContainerFunc: func(_ context.Context, _ string) (*podman.ContainerWaitResponse, error) {
			return &podman.ContainerWaitResponse{StatusCode: 0}, nil
		},
	}

	orch.podmanClient = mockClient

	job := &models.Job{
		ID:        "job-url-1",
		State:     models.JobStatePending,
		InputType: inputTypeURLs,
		URLs:      []string{"https://example.com", "https://example.com/about"},
		Config:    models.JobConfig{Modules: []string{"axe"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := database.CreateJob(context.Background(), job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	if err := orch.handleURLJob(context.Background(), job); err != nil {
		t.Fatalf("handleURLJob failed: %v", err)
	}

	// Verify job state updated to SCANNING (check in-memory object)
	if job.State != models.JobStateScanning {
		t.Errorf("expected job state SCANNING, got %s", job.State)
	}

	if job.PodID == "" {
		t.Error("expected pod ID to be set")
	}
}

func TestHandleURLJob_LoopbackTargetRequiresHostNetns(t *testing.T) {
	orch, database, publisher, _ := setupTestOrchestrator(t)
	defer orch.WaitForMonitors()

	loopbackURL := "http://127.0.0.1:3000"
	msg := "loopback targets require POD_NETNS_MODE=host for job pods (local dev only)"

	mockClient := &mockPodmanClient{
		createPodFunc: func(_ context.Context, _ *podman.PodCreateRequest) (*podman.PodCreateResponse, error) {
			t.Fatalf("unexpected CreatePod call; loopback guardrail should fail before pod creation")
			return nil, errors.New("unreachable")
		},
	}

	orch.podmanClient = mockClient

	job := &models.Job{
		ID:        "job-url-loopback",
		State:     models.JobStatePending,
		InputType: inputTypeURLs,
		URLs:      []string{loopbackURL},
		Config:    models.JobConfig{Modules: []string{"axe"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := database.CreateJob(context.Background(), job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	err := orch.handleURLJob(context.Background(), job)
	if err == nil {
		t.Fatalf("handleURLJob err = nil, want non-nil")
	}

	if !strings.Contains(err.Error(), "POD_NETNS_MODE=host") {
		t.Fatalf("handleURLJob err = %q, want to contain %q", err.Error(), "POD_NETNS_MODE=host")
	}

	updated, err := database.GetJob(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("failed to fetch job: %v", err)
	}

	if updated.State != models.JobStateFailed {
		t.Fatalf("job.State = %s, want %s", updated.State, models.JobStateFailed)
	}

	if updated.Error != msg {
		t.Fatalf("job.Error = %q, want %q", updated.Error, msg)
	}

	if updated.ErrorDetails != "pod_netns_mode=bridge" {
		t.Fatalf("job.ErrorDetails = %q, want %q", updated.ErrorDetails, "pod_netns_mode=bridge")
	}

	if updated.LastStage != "scanning" {
		t.Fatalf("job.LastStage = %q, want %q", updated.LastStage, "scanning")
	}

	if publisher.failedCount() != 1 {
		t.Fatalf("publisher.failedCount() = %d, want 1", publisher.failedCount())
	}
}

func TestHandleURLJob_CreatesNewPod(t *testing.T) {
	orch, database, _, _ := setupTestOrchestrator(t)
	defer orch.WaitForMonitors()

	var capturedPodReq *podman.PodCreateRequest

	mockClient := &mockPodmanClient{
		createPodFunc: func(_ context.Context, req *podman.PodCreateRequest) (*podman.PodCreateResponse, error) {
			capturedPodReq = req
			return &podman.PodCreateResponse{ID: "pod-new-123"}, nil
		},
		inspectVolumeFunc: func(_ context.Context, name string) (*podman.VolumeInfo, error) {
			return &podman.VolumeInfo{Name: name, Mountpoint: "/volumes/" + name}, nil
		},
		createContainerFunc: func(_ context.Context, _ *podman.ContainerCreateRequest) (*podman.ContainerCreateResponse, error) {
			return &podman.ContainerCreateResponse{ID: "container-123"}, nil
		},
		startContainerFunc: func(_ context.Context, _ string) error {
			return nil
		},
		waitContainerFunc: func(_ context.Context, _ string) (*podman.ContainerWaitResponse, error) {
			return &podman.ContainerWaitResponse{StatusCode: 0}, nil
		},
	}

	orch.podmanClient = mockClient

	job := &models.Job{
		ID:        "job-url-2",
		State:     models.JobStatePending,
		InputType: inputTypeURLs,
		URLs:      []string{"https://test.com"},
		Config:    models.JobConfig{Modules: []string{"lighthouse"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := database.CreateJob(context.Background(), job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	if err := orch.handleURLJob(context.Background(), job); err != nil {
		t.Fatalf("handleURLJob failed: %v", err)
	}

	// Verify pod was created with correct name and labels
	if capturedPodReq == nil {
		t.Fatal("expected pod create request")
	}

	expectedName := fmt.Sprintf("job-%s", job.ID)
	if capturedPodReq.Name != expectedName {
		t.Errorf("expected pod name %s, got %s", expectedName, capturedPodReq.Name)
	}

	if capturedPodReq.Labels["job_id"] != job.ID {
		t.Errorf("expected job_id label %s, got %s", job.ID, capturedPodReq.Labels["job_id"])
	}

	if capturedPodReq.Labels["managed_by"] != "orchestrator" {
		t.Error("expected managed_by label to be 'orchestrator'")
	}
}

func TestHandleURLJob_ReusesExistingPod(t *testing.T) {
	orch, database, _, _ := setupTestOrchestrator(t)
	defer orch.WaitForMonitors()

	podCreateCalls := 0

	mockClient := &mockPodmanClient{
		createPodFunc: func(_ context.Context, _ *podman.PodCreateRequest) (*podman.PodCreateResponse, error) {
			podCreateCalls++
			return &podman.PodCreateResponse{ID: "pod-existing"}, nil
		},
		inspectVolumeFunc: func(_ context.Context, name string) (*podman.VolumeInfo, error) {
			return &podman.VolumeInfo{Name: name, Mountpoint: "/volumes/" + name}, nil
		},
		createContainerFunc: func(_ context.Context, _ *podman.ContainerCreateRequest) (*podman.ContainerCreateResponse, error) {
			return &podman.ContainerCreateResponse{ID: "container-123"}, nil
		},
		startContainerFunc: func(_ context.Context, _ string) error {
			return nil
		},
		waitContainerFunc: func(_ context.Context, _ string) (*podman.ContainerWaitResponse, error) {
			return &podman.ContainerWaitResponse{StatusCode: 0}, nil
		},
	}

	orch.podmanClient = mockClient

	job := &models.Job{
		ID:        "job-url-3",
		State:     models.JobStatePending,
		InputType: inputTypeURLs,
		URLs:      []string{"https://existing.com"},
		PodID:     "pod-already-exists",
		Config:    models.JobConfig{Modules: []string{"axe"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := database.CreateJob(context.Background(), job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	if err := orch.handleURLJob(context.Background(), job); err != nil {
		t.Fatalf("handleURLJob failed: %v", err)
	}

	// Verify pod was NOT created (should reuse existing)
	if podCreateCalls != 0 {
		t.Errorf("expected no pod creation calls, got %d", podCreateCalls)
	}

	// Verify job still has original pod ID (check in-memory object)
	if job.PodID != "pod-already-exists" {
		t.Errorf("expected pod ID to remain 'pod-already-exists', got %s", job.PodID)
	}
}

func TestHandleURLJob_IgnoresTerminalStates(t *testing.T) {
	orch, database, _, _ := setupTestOrchestrator(t)
	defer orch.WaitForMonitors()

	terminalStates := []models.JobState{
		models.JobStateDone,
		models.JobStateFailed,
	}

	for _, state := range terminalStates {
		t.Run(string(state), func(t *testing.T) {
			job := &models.Job{
				ID:        fmt.Sprintf("job-terminal-%s", state),
				State:     state,
				InputType: inputTypeURLs,
				URLs:      []string{"https://example.com"},
				Config:    models.JobConfig{Modules: []string{"axe"}},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			if err := database.CreateJob(context.Background(), job); err != nil {
				t.Fatalf("failed to create job: %v", err)
			}

			if err := orch.handleURLJob(context.Background(), job); err != nil {
				t.Fatalf("handleURLJob should not fail for terminal state: %v", err)
			}

			// Verify state didn't change (check in-memory object)
			if job.State != state {
				t.Errorf("expected state to remain %s, got %s", state, job.State)
			}
		})
	}
}

func TestHandleURLJob_SkipsInProgressStates(t *testing.T) {
	orch, database, _, _ := setupTestOrchestrator(t)
	defer orch.WaitForMonitors()

	inProgressStates := []models.JobState{
		models.JobStateScanning,
		models.JobStateCompleting,
	}

	for _, state := range inProgressStates {
		t.Run(string(state), func(t *testing.T) {
			job := &models.Job{
				ID:        fmt.Sprintf("job-inprogress-%s", state),
				State:     state,
				InputType: inputTypeURLs,
				URLs:      []string{"https://example.com"},
				Config:    models.JobConfig{Modules: []string{"axe"}},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			if err := database.CreateJob(context.Background(), job); err != nil {
				t.Fatalf("failed to create job: %v", err)
			}

			if err := orch.handleURLJob(context.Background(), job); err != nil {
				t.Fatalf("handleURLJob should not fail for in-progress state: %v", err)
			}

			// Verify state didn't change (check in-memory object)
			if job.State != state {
				t.Errorf("expected state to remain %s, got %s", state, job.State)
			}
		})
	}
}

func TestHandleURLJob_WithPodNetwork(t *testing.T) {
	orch, database, _, _ := setupTestOrchestrator(t)
	defer orch.WaitForMonitors()

	orch.podNetwork = "stageflow-network"

	var capturedPodReq *podman.PodCreateRequest

	mockClient := &mockPodmanClient{
		createPodFunc: func(_ context.Context, req *podman.PodCreateRequest) (*podman.PodCreateResponse, error) {
			capturedPodReq = req
			return &podman.PodCreateResponse{ID: "pod-network-123"}, nil
		},
		inspectVolumeFunc: func(_ context.Context, name string) (*podman.VolumeInfo, error) {
			return &podman.VolumeInfo{Name: name, Mountpoint: "/volumes/" + name}, nil
		},
		createContainerFunc: func(_ context.Context, _ *podman.ContainerCreateRequest) (*podman.ContainerCreateResponse, error) {
			return &podman.ContainerCreateResponse{ID: "container-123"}, nil
		},
		startContainerFunc: func(_ context.Context, _ string) error {
			return nil
		},
		waitContainerFunc: func(_ context.Context, _ string) (*podman.ContainerWaitResponse, error) {
			return &podman.ContainerWaitResponse{StatusCode: 0}, nil
		},
	}

	orch.podmanClient = mockClient

	job := &models.Job{
		ID:        "job-url-network",
		State:     models.JobStatePending,
		InputType: inputTypeURLs,
		URLs:      []string{"https://example.com"},
		Config:    models.JobConfig{Modules: []string{"axe"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := database.CreateJob(context.Background(), job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	if err := orch.handleURLJob(context.Background(), job); err != nil {
		t.Fatalf("handleURLJob failed: %v", err)
	}

	if capturedPodReq == nil {
		t.Fatal("expected pod create request")
	}

	// Verify network configuration
	if _, exists := capturedPodReq.Networks["stageflow-network"]; !exists {
		t.Error("expected pod to use stageflow-network")
	}

	if capturedPodReq.Netns.Nsmode != "bridge" {
		t.Errorf("expected Netns.Nsmode to be 'bridge', got %s", capturedPodReq.Netns.Nsmode)
	}
}

func TestHandleURLJob_PodCreationFailure(t *testing.T) {
	orch, database, _, _ := setupTestOrchestrator(t)
	defer orch.WaitForMonitors()

	mockClient := &mockPodmanClient{
		createPodFunc: func(_ context.Context, _ *podman.PodCreateRequest) (*podman.PodCreateResponse, error) {
			return nil, errors.New("pod creation failed")
		},
	}

	orch.podmanClient = mockClient

	job := &models.Job{
		ID:        "job-url-fail",
		State:     models.JobStatePending,
		InputType: inputTypeURLs,
		URLs:      []string{"https://example.com"},
		Config:    models.JobConfig{Modules: []string{"axe"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := database.CreateJob(context.Background(), job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	err := orch.handleURLJob(context.Background(), job)
	if err == nil {
		t.Fatal("expected error when pod creation fails")
	}

	if err.Error() != "failed to create pod: pod creation failed" {
		t.Errorf("unexpected error message: %v", err)
	}
}
