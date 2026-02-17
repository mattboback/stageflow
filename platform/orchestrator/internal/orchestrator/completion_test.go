package orchestrator

import (
	"testing"
	"time"

	"github.com/mattboback/stageflow/packages/shared-go/models"
)

func TestCompleteJobWithAggregatedResults_AlreadyDone(t *testing.T) {
	orch, database, _, _ := setupTestOrchestrator(t)

	job := &models.Job{
		ID:    "job-already-done",
		State: models.JobStateDone,
	}

	if err := database.CreateJob(t.Context(), job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	// Should return nil without error
	if err := orch.completeJobWithAggregatedResults(t.Context(), job); err != nil {
		t.Errorf("unexpected error for already completed job: %v", err)
	}

	// State should remain DONE
	updatedJob, err := database.GetJob(t.Context(), job.ID)
	if err != nil {
		t.Fatalf("failed to get job: %v", err)
	}

	if updatedJob.State != models.JobStateDone {
		t.Errorf("expected state DONE, got %s", updatedJob.State)
	}
}

func TestCompleteJobWithAggregatedResults_AlreadyFailed(t *testing.T) {
	orch, database, _, _ := setupTestOrchestrator(t)

	job := &models.Job{
		ID:    "job-already-failed",
		State: models.JobStateFailed,
	}

	if err := database.CreateJob(t.Context(), job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	// Should return nil without error
	if err := orch.completeJobWithAggregatedResults(t.Context(), job); err != nil {
		t.Errorf("unexpected error for already failed job: %v", err)
	}

	// State should remain FAILED
	updatedJob, err := database.GetJob(t.Context(), job.ID)
	if err != nil {
		t.Fatalf("failed to get job: %v", err)
	}

	if updatedJob.State != models.JobStateFailed {
		t.Errorf("expected state FAILED, got %s", updatedJob.State)
	}
}

func TestCompleteJobWithAggregatedResults_NoSuccessfulScanners(t *testing.T) {
	orch, database, _, _ := setupTestOrchestrator(t)

	job := &models.Job{
		ID:    "job-no-success",
		State: models.JobStateCompleting,
		ScannerResults: map[string]*models.ScannerResult{
			"axe": {
				ScannerType: "axe",
				Success:     false,
			},
			"lighthouse": {
				ScannerType: "lighthouse",
				Success:     false,
			},
		},
	}

	if err := database.CreateJob(t.Context(), job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	err := orch.completeJobWithAggregatedResults(t.Context(), job)
	if err == nil {
		t.Fatal("expected error when no successful scanners")
	}

	if err.Error() != "no successful scanner results found" {
		t.Errorf(
			"completeJobWithAggregatedResults error = %q, want %q",
			err.Error(),
			"no successful scanner results found",
		)
	}
}

func TestCompleteJobWithAggregatedResults_PrimaryArtifactsFollowExpectedScannerOrder(t *testing.T) {
	orch, database, publisher, mem := setupTestOrchestrator(t)

	job := &models.Job{
		ID:               "job-primary-expected-order",
		State:            models.JobStateScanning,
		InputType:        models.JobInputTypeZip,
		InputPath:        "staging/job-primary-expected-order/site.zip",
		Config:           models.JobConfig{Modules: []string{"axe", "lighthouse"}},
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
		ExpectedScanners: []string{"lighthouse", "axe"},
		ScannerResults: map[string]*models.ScannerResult{
			"axe": {
				ScannerType:  "axe",
				ResultsPath:  "job-primary-expected-order/axe/results.json",
				ReportPath:   "job-primary-expected-order/axe/report.html",
				StageLogPath: "job-primary-expected-order/axe/scan-stage-log.json",
				RecipePath:   "job-primary-expected-order/axe/scan-recipe.json",
				Success:      true,
			},
			"lighthouse": {
				ScannerType:  "lighthouse",
				ResultsPath:  "job-primary-expected-order/lighthouse/results.json",
				ReportPath:   "job-primary-expected-order/lighthouse/report.html",
				StageLogPath: "job-primary-expected-order/lighthouse/scan-stage-log.json",
				RecipePath:   "job-primary-expected-order/lighthouse/scan-recipe.json",
				Success:      true,
			},
		},
	}

	seedScanResults(t, mem, job.ID, job.ScannerResults["axe"].ResultsPath)
	seedScanResults(t, mem, job.ID, job.ScannerResults["lighthouse"].ResultsPath)

	if err := database.CreateJob(t.Context(), job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	if err := orch.completeJobWithAggregatedResults(t.Context(), job); err != nil {
		t.Fatalf("completeJobWithAggregatedResults: %v", err)
	}

	storedJob, err := database.GetJob(t.Context(), job.ID)
	if err != nil {
		t.Fatalf("failed to get job: %v", err)
	}

	wantPrimaryReport := job.ScannerResults["lighthouse"].ReportPath
	if storedJob.ReportKey != wantPrimaryReport {
		t.Fatalf("ReportKey = %q, want %q", storedJob.ReportKey, wantPrimaryReport)
	}

	completed := publisher.firstCompleted()
	if completed == nil {
		t.Fatal("expected job.completed payload to be published")
	}

	if completed.Artifacts.ReportHTML != wantPrimaryReport {
		t.Fatalf("Artifacts.ReportHTML = %q, want %q", completed.Artifacts.ReportHTML, wantPrimaryReport)
	}
}

func TestCompleteJobWithAggregatedResults_PrimaryArtifactsFallbackToAlphabetical(t *testing.T) {
	orch, database, publisher, mem := setupTestOrchestrator(t)

	job := &models.Job{
		ID:               "job-primary-fallback-alphabetical",
		State:            models.JobStateScanning,
		InputType:        models.JobInputTypeZip,
		InputPath:        "staging/job-primary-fallback-alphabetical/site.zip",
		Config:           models.JobConfig{Modules: []string{"axe", "pa11y"}},
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
		ExpectedScanners: []string{"lighthouse"},
		ScannerResults: map[string]*models.ScannerResult{
			"pa11y": {
				ScannerType:  "pa11y",
				ResultsPath:  "job-primary-fallback-alphabetical/pa11y/results.json",
				ReportPath:   "job-primary-fallback-alphabetical/pa11y/report.html",
				StageLogPath: "job-primary-fallback-alphabetical/pa11y/scan-stage-log.json",
				RecipePath:   "job-primary-fallback-alphabetical/pa11y/scan-recipe.json",
				Success:      true,
			},
			"axe": {
				ScannerType:  "axe",
				ResultsPath:  "job-primary-fallback-alphabetical/axe/results.json",
				ReportPath:   "job-primary-fallback-alphabetical/axe/report.html",
				StageLogPath: "job-primary-fallback-alphabetical/axe/scan-stage-log.json",
				RecipePath:   "job-primary-fallback-alphabetical/axe/scan-recipe.json",
				Success:      true,
			},
		},
	}

	seedScanResults(t, mem, job.ID, job.ScannerResults["axe"].ResultsPath)
	seedScanResults(t, mem, job.ID, job.ScannerResults["pa11y"].ResultsPath)

	if err := database.CreateJob(t.Context(), job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	if err := orch.completeJobWithAggregatedResults(t.Context(), job); err != nil {
		t.Fatalf("completeJobWithAggregatedResults: %v", err)
	}

	storedJob, err := database.GetJob(t.Context(), job.ID)
	if err != nil {
		t.Fatalf("failed to get job: %v", err)
	}

	wantPrimaryReport := job.ScannerResults["axe"].ReportPath
	if storedJob.ReportKey != wantPrimaryReport {
		t.Fatalf("ReportKey = %q, want %q", storedJob.ReportKey, wantPrimaryReport)
	}

	completed := publisher.firstCompleted()
	if completed == nil {
		t.Fatal("expected job.completed payload to be published")
	}

	if completed.Artifacts.ReportHTML != wantPrimaryReport {
		t.Fatalf("Artifacts.ReportHTML = %q, want %q", completed.Artifacts.ReportHTML, wantPrimaryReport)
	}
}
