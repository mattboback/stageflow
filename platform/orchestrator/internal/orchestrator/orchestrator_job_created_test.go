package orchestrator

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/mattboback/stageflow/packages/shared-go/events"
	"github.com/mattboback/stageflow/packages/shared-go/models"
	"github.com/mattboback/stageflow/platform/orchestrator/internal/podman"
)

func TestHandleJobCreated(t *testing.T) {
	orch, database, _, _ := setupTestOrchestrator(t)
	defer orch.WaitForMonitors()

	payload := &events.JobCreatedPayload{
		JobID:     "job-123",
		InputType: "zip",
		InputPath: "staging/job-123/test.zip",
		Config: models.JobConfig{
			Modules:    []string{"axe", "keyboard"},
			Screenshot: true,
		},
	}

	err := orch.HandleJobCreated(context.Background(), payload)
	if err != nil {
		t.Fatalf("HandleJobCreated failed: %v", err)
	}

	// Verify job was created in database
	job, err := database.GetJob(context.Background(), "job-123")
	if err != nil {
		t.Fatalf("Failed to get job from database: %v", err)
	}

	if job.State != models.JobStateExtracting {
		t.Errorf("Expected state EXTRACTING, got %s", job.State)
	}

	if job.PodID == "" {
		t.Error("Expected pod ID to be set")
	}
}

func TestHandleURLJobTransitionsToScanning(t *testing.T) {
	orch, database, _, _ := setupTestOrchestrator(t)
	defer orch.WaitForMonitors()

	payload := &events.JobCreatedPayload{
		JobID:     "url-job",
		InputType: "urls",
		URLs:      []string{"https://example.com"},
		Config: models.JobConfig{
			Modules: []string{"axe"},
		},
	}

	if err := orch.HandleJobCreated(context.Background(), payload); err != nil {
		t.Fatalf("HandleJobCreated for URL job failed: %v", err)
	}

	job, err := database.GetJob(context.Background(), "url-job")
	if err != nil {
		t.Fatalf("failed to fetch job: %v", err)
	}

	if job.State != models.JobStateScanning {
		t.Fatalf("expected state SCANNING, got %s", job.State)
	}

	if job.PodID == "" {
		t.Fatalf("expected pod to be created for URL job")
	}
}

func TestHandleJobCreated_DuplicatePendingRetriesThenIgnoresExtracting(t *testing.T) {
	database := newInMemoryDB(t)
	publisher := &mockPublisher{
		publishedJobCompleted: make([]*events.JobCompletedPayload, 0),
		publishedJobFailed:    make([]*events.JobFailedPayload, 0),
	}
	mem := newMemoryStorage()

	createPodCalls := 0
	podmanClient := &mockPodmanClient{
		createPodFunc: func(_ context.Context, _ *podman.PodCreateRequest) (*podman.PodCreateResponse, error) {
			createPodCalls++

			return &podman.PodCreateResponse{ID: fmt.Sprintf("pod-%d", createPodCalls)}, nil
		},
	}

	orch := NewOrchestrator(&Config{
		PodmanClient:   podmanClient,
		Database:       database,
		Publisher:      publisher,
		Storage:        mem,
		StagingStorage: mem,
	})
	defer orch.WaitForMonitors()

	insertJob(t, database, &models.Job{
		ID:        "job-dup",
		State:     models.JobStatePending,
		InputType: "zip",
		InputPath: "staging/job-dup/test.zip",
		Config: models.JobConfig{
			Modules: []string{"axe"},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	payload := &events.JobCreatedPayload{
		JobID:     "job-dup",
		InputType: "zip",
		InputPath: "staging/job-dup/test.zip",
		Config: models.JobConfig{
			Modules: []string{"axe"},
		},
	}

	if err := orch.HandleJobCreated(t.Context(), payload); err != nil {
		t.Fatalf("HandleJobCreated() first duplicate call error = %v, want nil", err)
	}

	job, err := database.GetJob(t.Context(), "job-dup")
	if err != nil {
		t.Fatalf("GetJob() after first duplicate call error = %v", err)
	}

	if job.State != models.JobStateExtracting {
		t.Errorf("job.State after first duplicate call = %s, want %s", job.State, models.JobStateExtracting)
	}

	if job.PodID != "pod-1" {
		t.Errorf("job.PodID after first duplicate call = %q, want %q", job.PodID, "pod-1")
	}

	err = orch.HandleJobCreated(t.Context(), payload)
	if err != nil {
		t.Fatalf("HandleJobCreated() second duplicate call error = %v, want nil", err)
	}

	job, err = database.GetJob(t.Context(), "job-dup")
	if err != nil {
		t.Fatalf("GetJob() after second duplicate call error = %v", err)
	}

	if job.PodID != "pod-1" {
		t.Errorf("job.PodID after second duplicate call = %q, want %q", job.PodID, "pod-1")
	}

	if createPodCalls != 1 {
		t.Errorf("createPodCalls = %d, want 1", createPodCalls)
	}
}

func TestResolveDuplicateJobCreated_RecreatesMissingJob(t *testing.T) {
	database := newInMemoryDB(t)

	orch := NewOrchestrator(&Config{
		PodmanClient:   &mockPodmanClient{},
		Database:       database,
		Publisher:      &mockPublisher{},
		Storage:        newMemoryStorage(),
		StagingStorage: newMemoryStorage(),
	})
	defer orch.WaitForMonitors()

	fallbackJob := &models.Job{
		ID:        "job-missing",
		State:     models.JobStatePending,
		InputType: "urls",
		URLs:      []string{"https://example.com"},
		Config: models.JobConfig{
			Modules: []string{"axe"},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	job, handled, err := orch.resolveDuplicateJobCreated(t.Context(), fallbackJob)
	if err != nil {
		t.Fatalf("resolveDuplicateJobCreated() error = %v", err)
	}

	if handled {
		t.Fatalf("resolveDuplicateJobCreated() handled = true, want false")
	}

	if job == nil {
		t.Fatal("resolveDuplicateJobCreated() returned nil job")
	}

	if job.ID != fallbackJob.ID {
		t.Fatalf("resolveDuplicateJobCreated() job.ID = %q, want %q", job.ID, fallbackJob.ID)
	}

	storedJob, err := database.GetJob(t.Context(), fallbackJob.ID)
	if err != nil {
		t.Fatalf("database.GetJob() error = %v", err)
	}

	if storedJob.ID != fallbackJob.ID {
		t.Fatalf("stored job ID = %q, want %q", storedJob.ID, fallbackJob.ID)
	}
}
