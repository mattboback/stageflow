package test

import (
	"context"
	"testing"

	"github.com/mattboback/stageflow/packages/shared-go/events"
	"github.com/mattboback/stageflow/packages/shared-go/models"
)

// TestE2E_SuccessfulJobFlow tests the complete happy path from job creation to completion.
func TestE2E_SuccessfulJobFlow(t *testing.T) {
	orch, database, podmanClient, publisher, mem := setupE2ETest(t)

	ctx := context.Background()
	jobID := "test-job-123"

	t.Log("Step 1: Handling job.created event")
	jobCreated := &events.JobCreatedPayload{
		JobID:     jobID,
		InputType: "zip",
		InputPath: "staging/test-job-123/website.zip",
		Config: models.JobConfig{
			Modules:    []string{"axe", "keyboard"},
			Screenshot: true,
		},
	}

	err := orch.HandleJobCreated(ctx, jobCreated)
	if err != nil {
		t.Fatalf("HandleJobCreated failed: %v", err)
	}

	job := mustGetJob(t, database, jobID)
	if job.State != models.JobStateExtracting {
		t.Errorf("Expected state EXTRACTING, got %s", job.State)
	}
	if job.PodID == "" {
		t.Error("Expected pod ID to be set")
	}
	if len(podmanClient.pods) != 1 {
		t.Errorf("Expected 1 pod to be created, got %d", len(podmanClient.pods))
	}

	t.Log("Step 2: Handling extraction.ready event")
	extractionReady := &events.ExtractionReadyPayload{
		JobID:                  jobID,
		ProvenancePath:         "/workspace/provenance.json",
		BaseURL:                "http://localhost:8080",
		TotalPages:             5,
		ProvenanceArtifactPath: jobID + "/provenance.json",
	}

	err = orch.HandleExtractionReady(ctx, extractionReady)
	if err != nil {
		t.Fatalf("HandleExtractionReady failed: %v", err)
	}

	job = mustGetJob(t, database, jobID)
	if job.State != models.JobStateScanning {
		t.Errorf("Expected state SCANNING, got %s", job.State)
	}

	t.Log("Step 3: Handling scan.completed event")
	scanCompleted := &events.ScanCompletedPayload{
		JobID:             jobID,
		ScannerType:       "axe",
		ResultsPath:       jobID + "/axe/results.json",
		ReportPath:        jobID + "/axe/report.html",
		TotalPagesScanned: 5,
		Summary: events.ScanSummary{
			TotalViolations: 12,
			BySeverity: map[string]int{
				"critical": 10,
				"minor":    2,
			},
		},
	}

	seedScanResults(t, mem, jobID, scanCompleted.ResultsPath)

	err = orch.HandleScanCompleted(ctx, scanCompleted)
	if err != nil {
		t.Fatalf("HandleScanCompleted failed: %v", err)
	}

	job = mustGetJob(t, database, jobID)
	if job.State != models.JobStateDone {
		t.Errorf("Expected state DONE, got %s", job.State)
	}
	if job.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set")
	}

	if len(podmanClient.pods) != 0 {
		t.Errorf("Expected pod to be cleaned up, found %d pods", len(podmanClient.pods))
	}

	if len(publisher.completedEvents) != 1 {
		t.Errorf("Expected 1 job.completed event, got %d", len(publisher.completedEvents))
	} else {
		event := publisher.completedEvents[0]
		if event.JobID != jobID {
			t.Errorf("Expected job ID %s, got %s", jobID, event.JobID)
		}
		if event.Status != "success" {
			t.Errorf("Expected status success, got %s", event.Status)
		}
	}

	if len(publisher.failedEvents) != 0 {
		t.Errorf("Expected 0 job.failed events, got %d", len(publisher.failedEvents))
	}
}
