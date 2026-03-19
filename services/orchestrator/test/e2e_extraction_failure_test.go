package test

import (
	"context"
	"testing"

	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/models"
)

// TestE2E_ExtractionFailureFlow tests failure during extraction phase.
func TestE2E_ExtractionFailureFlow(t *testing.T) {
	orch, database, podmanClient, publisher, _ := setupE2ETest(t)

	ctx := context.Background()
	jobID := "test-job-456"

	t.Log("Step 1: Handling job.created event")

	jobCreated := &events.JobCreatedPayload{
		JobID:     jobID,
		InputType: "zip",
		InputPath: "staging/test-job-456/corrupt.zip",
		Config:    models.JobConfig{Modules: []string{"axe"}},
	}

	err := orch.HandleJobCreated(ctx, jobCreated)
	if err != nil {
		t.Fatalf("HandleJobCreated failed: %v", err)
	}

	if len(podmanClient.pods) != 1 {
		t.Errorf("Expected 1 pod, got %d", len(podmanClient.pods))
	}

	t.Log("Step 2: Handling extraction.failed event")

	extractionFailed := &events.ExtractionFailedPayload{
		JobID:        jobID,
		Error:        "Corrupt ZIP file",
		ErrorDetails: "Invalid ZIP header",
	}

	err = orch.HandleExtractionFailed(ctx, extractionFailed)
	if err != nil {
		t.Fatalf("HandleExtractionFailed failed: %v", err)
	}

	job := mustGetJob(t, database, jobID)
	if job.State != models.JobStateFailed {
		t.Errorf("Expected state FAILED, got %s", job.State)
	}

	if job.Error != "Corrupt ZIP file" {
		t.Errorf("Expected error message 'Corrupt ZIP file', got '%s'", job.Error)
	}

	if job.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set on failure")
	}

	if len(podmanClient.pods) != 0 {
		t.Errorf("Expected pod to be cleaned up, found %d pods", len(podmanClient.pods))
	}

	if len(publisher.failedEvents) != 1 {
		t.Errorf("Expected 1 job.failed event, got %d", len(publisher.failedEvents))
	} else {
		event := publisher.failedEvents[0]
		if event.JobID != jobID {
			t.Errorf("Expected job ID %s, got %s", jobID, event.JobID)
		}

		if event.Stage != "extraction" {
			t.Errorf("Expected stage extraction, got %s", event.Stage)
		}
	}
}
