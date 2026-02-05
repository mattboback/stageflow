package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/mattboback/stageflow/packages/shared-go/events"
	"github.com/mattboback/stageflow/packages/shared-go/models"
)

func TestHandleScanCompleted(t *testing.T) {
	orch, database, publisher, mem := setupTestOrchestrator(t)

	// Create a job first
	job := &models.Job{
		ID:        "job-123",
		State:     models.JobStateScanning,
		InputType: "zip",
		InputPath: "test.zip",
		PodID:     "pod-123",
		Config:    models.JobConfig{Modules: []string{"axe"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	insertJob(t, database, job)

	// Set expected scanners (this is normally done in startScanning)
	if err := database.SetExpectedScanners(context.Background(), "job-123", []string{"axe"}); err != nil {
		t.Fatalf("Failed to set expected scanners: %v", err)
	}

	payload := &events.ScanCompletedPayload{
		JobID:             "job-123",
		ScannerType:       "axe",
		ResultsPath:       "job-123/axe/results.json",
		ReportPath:        "job-123/axe/report.html",
		TotalPagesScanned: 3,
		Summary: events.ScanSummary{
			TotalViolations: 5,
			BySeverity:      map[string]int{"critical": 5},
		},
	}

	seedScanResults(t, mem, "job-123", payload.ResultsPath)

	err := orch.HandleScanCompleted(context.Background(), payload)
	if err != nil {
		t.Fatalf("HandleScanCompleted failed: %v", err)
	}

	// Verify job was marked as done
	completedJob, err := database.GetJob(context.Background(), "job-123")
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	if completedJob.State != models.JobStateDone {
		t.Errorf("Expected state DONE, got %s", completedJob.State)
	}

	if completedJob.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set")
	}

	// Verify job.completed event was published
	if publisher.completedCount() != 1 {
		t.Errorf("Expected 1 job.completed event, got %d", publisher.completedCount())
	}
}

func TestHandleScanFailed(t *testing.T) {
	orch, database, publisher, _ := setupTestOrchestrator(t)

	// Create a job first
	job := &models.Job{
		ID:        "job-123",
		State:     models.JobStateScanning,
		InputType: "zip",
		InputPath: "test.zip",
		PodID:     "pod-123",
		Config:    models.JobConfig{Modules: []string{"axe"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	insertJob(t, database, job)

	// Set expected scanners (this is normally done in startScanning)
	if err := database.SetExpectedScanners(context.Background(), "job-123", []string{"axe"}); err != nil {
		t.Fatalf("Failed to set expected scanners: %v", err)
	}

	payload := &events.ScanFailedPayload{
		JobID:        "job-123",
		ScannerType:  "axe",
		Error:        "Browser crashed",
		ErrorDetails: "Chromium segfault",
	}

	err := orch.HandleScanFailed(context.Background(), payload)
	if err != nil {
		t.Fatalf("HandleScanFailed failed: %v", err)
	}

	// Verify job was marked as failed (all scanners failed)
	failedJob, err := database.GetJob(context.Background(), "job-123")
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	if failedJob.State != models.JobStateFailed {
		t.Errorf("Expected state FAILED, got %s", failedJob.State)
	}

	// Verify job.failed event was published
	if publisher.failedCount() != 1 {
		t.Errorf("Expected 1 job.failed event, got %d", publisher.failedCount())
	}
}

func TestHandleMultipleScanners(t *testing.T) {
	orch, database, publisher, mem := setupTestOrchestrator(t)

	// Create a job with multiple scanner types
	job := &models.Job{
		ID:        "job-multi",
		State:     models.JobStateScanning,
		InputType: "zip",
		InputPath: "test.zip",
		PodID:     "pod-multi",
		Config:    models.JobConfig{Modules: []string{"axe", "lighthouse"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	insertJob(t, database, job)

	// Set expected scanners (this is normally done in startScanning)
	if err := database.SetExpectedScanners(
		context.Background(),
		"job-multi",
		[]string{"axe", "lighthouse"},
	); err != nil {
		t.Fatalf("Failed to set expected scanners: %v", err)
	}

	// First scanner completes (axe)
	axePayload := &events.ScanCompletedPayload{
		JobID:             "job-multi",
		ScannerType:       "axe",
		ResultsPath:       "job-multi/axe/results.json",
		ReportPath:        "job-multi/axe/report.html",
		TotalPagesScanned: 3,
	}

	seedScanResults(t, mem, "job-multi", axePayload.ResultsPath)

	err := orch.HandleScanCompleted(context.Background(), axePayload)
	if err != nil {
		t.Fatalf("HandleScanCompleted (axe) failed: %v", err)
	}

	// Job should NOT be done yet - waiting for lighthouse
	partialJob, err := database.GetJob(context.Background(), "job-multi")
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	if partialJob.State == models.JobStateDone {
		t.Errorf("Expected job to NOT be DONE yet (waiting for lighthouse), got %s", partialJob.State)
	}

	if publisher.completedCount() != 0 {
		t.Errorf("Expected 0 job.completed events (waiting for lighthouse), got %d", publisher.completedCount())
	}

	// Verify axe was recorded as completed
	if len(partialJob.CompletedScanners) != 1 {
		t.Errorf("Expected 1 completed scanner, got %d", len(partialJob.CompletedScanners))
	}

	// Second scanner completes (lighthouse)
	lighthousePayload := &events.ScanCompletedPayload{
		JobID:             "job-multi",
		ScannerType:       "lighthouse",
		ResultsPath:       "job-multi/lighthouse/results.json",
		ReportPath:        "job-multi/lighthouse/report.html",
		TotalPagesScanned: 3,
	}

	seedScanResults(t, mem, "job-multi", lighthousePayload.ResultsPath)

	err = orch.HandleScanCompleted(context.Background(), lighthousePayload)
	if err != nil {
		t.Fatalf("HandleScanCompleted (lighthouse) failed: %v", err)
	}

	// NOW job should be done
	completedJob, err := database.GetJob(context.Background(), "job-multi")
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	if completedJob.State != models.JobStateDone {
		t.Errorf("Expected state DONE after all scanners complete, got %s", completedJob.State)
	}

	// Verify job.completed event was published
	if publisher.completedCount() != 1 {
		t.Errorf("Expected 1 job.completed event, got %d", publisher.completedCount())
	}

	// Verify both scanner results are recorded
	if len(completedJob.CompletedScanners) != 2 {
		t.Errorf("Expected 2 completed scanners, got %d", len(completedJob.CompletedScanners))
	}

	if len(completedJob.ScannerResults) != 2 {
		t.Errorf("Expected 2 scanner results, got %d", len(completedJob.ScannerResults))
	}

	// Verify the job.completed event includes scanner artifacts
	if publisher.completedCount() == 1 {
		completedPayload := publisher.firstCompleted()
		if len(completedPayload.ScannerArtifacts) != 2 {
			t.Errorf("Expected 2 scanner artifacts in event, got %d", len(completedPayload.ScannerArtifacts))
		}
	}
}

func TestHandlePartialScannerSuccess(t *testing.T) {
	orch, database, publisher, mem := setupTestOrchestrator(t)

	// Create a job with multiple scanner types
	job := &models.Job{
		ID:        "job-partial",
		State:     models.JobStateScanning,
		InputType: "zip",
		InputPath: "test.zip",
		PodID:     "pod-partial",
		Config:    models.JobConfig{Modules: []string{"axe", "lighthouse"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	insertJob(t, database, job)

	// Set expected scanners
	if err := database.SetExpectedScanners(
		context.Background(),
		"job-partial",
		[]string{"axe", "lighthouse"},
	); err != nil {
		t.Fatalf("Failed to set expected scanners: %v", err)
	}

	// First scanner succeeds (axe)
	axePayload := &events.ScanCompletedPayload{
		JobID:             "job-partial",
		ScannerType:       "axe",
		ResultsPath:       "job-partial/axe/results.json",
		ReportPath:        "job-partial/axe/report.html",
		TotalPagesScanned: 3,
	}

	seedScanResults(t, mem, "job-partial", axePayload.ResultsPath)

	err := orch.HandleScanCompleted(context.Background(), axePayload)
	if err != nil {
		t.Fatalf("HandleScanCompleted (axe) failed: %v", err)
	}

	// Second scanner fails (lighthouse)
	lighthouseFailPayload := &events.ScanFailedPayload{
		JobID:        "job-partial",
		ScannerType:  "lighthouse",
		Error:        "Lighthouse timed out",
		ErrorDetails: "Page load exceeded 30s",
	}

	err = orch.HandleScanFailed(context.Background(), lighthouseFailPayload)
	if err != nil {
		t.Fatalf("HandleScanFailed (lighthouse) failed: %v", err)
	}

	// Job should complete successfully (partial success - axe succeeded)
	completedJob, err := database.GetJob(context.Background(), "job-partial")
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	if completedJob.State != models.JobStateDone {
		t.Errorf("Expected state DONE (partial success), got %s", completedJob.State)
	}

	// Verify job.completed event was published (not job.failed)
	if publisher.completedCount() != 1 {
		t.Errorf("Expected 1 job.completed event, got %d", publisher.completedCount())
	}

	if publisher.failedCount() != 0 {
		t.Errorf("Expected 0 job.failed events, got %d", publisher.failedCount())
	}

	// Verify scanner results include both success and failure
	if len(completedJob.ScannerResults) != 2 {
		t.Errorf("Expected 2 scanner results, got %d", len(completedJob.ScannerResults))
	}

	if axeResult, ok := completedJob.ScannerResults["axe"]; ok {
		if !axeResult.Success {
			t.Error("Expected axe result to be successful")
		}
	} else {
		t.Error("Expected axe result to exist")
	}

	if lhResult, ok := completedJob.ScannerResults["lighthouse"]; ok {
		if lhResult.Success {
			t.Error("Expected lighthouse result to be failed")
		}
	} else {
		t.Error("Expected lighthouse result to exist")
	}
}
