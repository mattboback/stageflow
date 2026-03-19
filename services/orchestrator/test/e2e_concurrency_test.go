package test

import (
	"context"
	"testing"

	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/models"
)

// TestE2E_MultipleJobsConcurrent tests handling multiple jobs concurrently.
func TestE2E_MultipleJobsConcurrent(t *testing.T) {
	orch, database, podmanClient, publisher, mem := setupE2ETest(t)

	ctx := context.Background()
	numJobs := 5

	// Create multiple jobs.
	t.Logf("Creating %d concurrent jobs", numJobs)

	for i := range numJobs {
		jobID := "concurrent-job-" + string(rune('A'+i))

		jobCreated := &events.JobCreatedPayload{
			JobID:     jobID,
			InputType: "zip",
			InputPath: "staging/" + jobID + "/test.zip",
			Config:    models.JobConfig{Modules: []string{"axe"}},
		}

		if err := orch.HandleJobCreated(ctx, jobCreated); err != nil {
			t.Fatalf("HandleJobCreated failed for job %s: %v", jobID, err)
		}
	}

	if len(podmanClient.pods) != numJobs {
		t.Errorf("Expected %d pods, got %d", numJobs, len(podmanClient.pods))
	}

	for i := range numJobs {
		jobID := "concurrent-job-" + string(rune('A'+i))

		if err := orch.HandleExtractionReady(ctx, &events.ExtractionReadyPayload{
			JobID:      jobID,
			TotalPages: 1,
		}); err != nil {
			t.Fatalf("HandleExtractionReady failed for job %s: %v", jobID, err)
		}

		scanCompleted := &events.ScanCompletedPayload{
			JobID:             jobID,
			ScannerType:       "axe",
			ResultsPath:       jobID + "/axe/results.json",
			ReportPath:        jobID + "/axe/report.html",
			TotalPagesScanned: 1,
		}
		seedScanResults(t, mem, jobID, scanCompleted.ResultsPath)

		if err := orch.HandleScanCompleted(ctx, scanCompleted); err != nil {
			t.Fatalf("HandleScanCompleted failed for job %s: %v", jobID, err)
		}
	}

	if len(publisher.completedEvents) != numJobs {
		t.Errorf("Expected %d completed events, got %d", numJobs, len(publisher.completedEvents))
	}

	if len(podmanClient.pods) != 0 {
		t.Errorf("Expected all pods cleaned up, found %d", len(podmanClient.pods))
	}

	for i := range numJobs {
		jobID := "concurrent-job-" + string(rune('A'+i))

		job := mustGetJob(t, database, jobID)
		if job.State != models.JobStateDone {
			t.Errorf("Job %s: expected DONE, got %s", jobID, job.State)
		}
	}
}
