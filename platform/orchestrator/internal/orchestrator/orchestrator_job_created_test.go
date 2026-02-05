package orchestrator

import (
	"context"
	"testing"

	"github.com/mattboback/stageflow/packages/shared-go/events"
	"github.com/mattboback/stageflow/packages/shared-go/models"
)

func TestHandleJobCreated(t *testing.T) {
	orch, database, _, _ := setupTestOrchestrator(t)

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
