package test

import (
	"context"
	"testing"

	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/models"
)

// TestE2E_ScanFailureFlow tests failure during scanning phase.
func TestE2E_ScanFailureFlow(t *testing.T) {
	orch, database, podmanClient, publisher, _ := setupE2ETest(t)

	ctx := context.Background()
	jobID := "test-job-789"

	jobCreated := &events.JobCreatedPayload{
		JobID:     jobID,
		InputType: events.InputTypeZip,
		InputPath: "staging/test-job-789/site.zip",
		Config:    models.JobConfig{Modules: []string{"axe"}},
	}
	if err := orch.HandleJobCreated(ctx, jobCreated); err != nil {
		t.Fatalf("HandleJobCreated failed: %v", err)
	}

	extractionReady := &events.ExtractionReadyPayload{
		JobID:          jobID,
		ProvenancePath: "/workspace/provenance.json",
		BaseURL:        "http://127.0.0.1:8080",
		TotalPages:     3,
	}
	if err := orch.HandleExtractionReady(ctx, extractionReady); err != nil {
		t.Fatalf("HandleExtractionReady failed: %v", err)
	}

	job := mustGetJob(t, database, jobID)
	if job.State != models.JobStateScanning {
		t.Fatalf("Expected state SCANNING, got %s", job.State)
	}

	t.Log("Step 3: Handling scan.failed event")

	scanFailed := &events.ScanFailedPayload{
		JobID:        jobID,
		ScannerType:  "axe", // Must match the expected scanner type.
		Error:        "Browser crashed",
		ErrorDetails: "Chromium segmentation fault",
	}

	err := orch.HandleScanFailed(ctx, scanFailed)
	if err != nil {
		t.Fatalf("HandleScanFailed failed: %v", err)
	}

	job = mustGetJob(t, database, jobID)
	if job.State != models.JobStateFailed {
		t.Errorf("Expected state FAILED, got %s", job.State)
	}

	if job.Error == "" {
		t.Errorf("Expected error to be set")
	}

	if len(podmanClient.pods) != 0 {
		t.Errorf("Expected pod to be cleaned up, found %d pods", len(podmanClient.pods))
	}

	if len(publisher.failedEvents) != 1 {
		t.Errorf("Expected 1 job.failed event, got %d", len(publisher.failedEvents))
	} else if publisher.failedEvents[0].Stage != "scanning" {
		t.Errorf("Expected stage scanning, got %s", publisher.failedEvents[0].Stage)
	}
}
