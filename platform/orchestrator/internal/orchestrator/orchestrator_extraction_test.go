package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/mattboback/stageflow/packages/shared-go/events"
	"github.com/mattboback/stageflow/packages/shared-go/models"
)

func TestHandleExtractionReady(t *testing.T) {
	orch, database, _, _ := setupTestOrchestrator(t)

	// Create a job first
	job := &models.Job{
		ID:        "job-123",
		State:     models.JobStateExtracting,
		InputType: "zip",
		InputPath: "test.zip",
		PodID:     "pod-123",
		Config:    models.JobConfig{Modules: []string{"axe"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	insertJob(t, database, job)

	payload := &events.ExtractionReadyPayload{
		JobID:                  "job-123",
		ProvenancePath:         "/workspace/provenance.json",
		BaseURL:                "http://localhost:8080",
		TotalPages:             3,
		ProvenanceArtifactPath: "job-123/provenance.json",
	}

	err := orch.HandleExtractionReady(context.Background(), payload)
	if err != nil {
		t.Fatalf("HandleExtractionReady failed: %v", err)
	}

	// Verify job state was updated
	updatedJob, err := database.GetJob(context.Background(), "job-123")
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	if updatedJob.State != models.JobStateScanning {
		t.Errorf("Expected state SCANNING, got %s", updatedJob.State)
	}

	if updatedJob.ProvenancePath != "/workspace/provenance.json" {
		t.Errorf("Expected provenance path to be persisted, got %q", updatedJob.ProvenancePath)
	}

	if updatedJob.ProvenanceKey != "job-123/provenance.json" {
		t.Errorf("Expected provenance key to be persisted, got %q", updatedJob.ProvenanceKey)
	}
}

func TestHandleExtractionFailed(t *testing.T) {
	orch, database, publisher, _ := setupTestOrchestrator(t)

	// Create a job first
	job := &models.Job{
		ID:        "job-123",
		State:     models.JobStateExtracting,
		InputType: "zip",
		InputPath: "test.zip",
		PodID:     "pod-123",
		Config:    models.JobConfig{Modules: []string{"axe"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	insertJob(t, database, job)

	payload := &events.ExtractionFailedPayload{
		JobID:        "job-123",
		Error:        "Failed to extract ZIP",
		ErrorDetails: "Corrupt ZIP file",
	}

	err := orch.HandleExtractionFailed(context.Background(), payload)
	if err != nil {
		t.Fatalf("HandleExtractionFailed failed: %v", err)
	}

	// Verify job was marked as failed
	failedJob, err := database.GetJob(context.Background(), "job-123")
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	if failedJob.State != models.JobStateFailed {
		t.Errorf("Expected state FAILED, got %s", failedJob.State)
	}

	if failedJob.Error != "Failed to extract ZIP" {
		t.Errorf("Expected error message, got %s", failedJob.Error)
	}

	// Verify job.failed event was published
	if publisher.failedCount() != 1 {
		t.Errorf("Expected 1 job.failed event, got %d", publisher.failedCount())
	}
}
