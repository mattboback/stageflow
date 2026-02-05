package orchestrator

import (
	"context"
	"errors"
	"testing"

	"github.com/mattboback/stageflow/packages/shared-go/events"
	"github.com/mattboback/stageflow/packages/shared-go/models"
)

func TestCompletionError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      *completionError
		expected string
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: "",
		},
		{
			name:     "error with nil Err field",
			err:      &completionError{Stage: "test", Err: nil},
			expected: "",
		},
		{
			name:     "error with message",
			err:      &completionError{Stage: "testing", Err: errors.New("test error")},
			expected: "test error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.err.Error()
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestCompletionError_Unwrap(t *testing.T) {
	t.Parallel()

	t.Run("nil error returns nil", func(t *testing.T) {
		t.Parallel()

		var cerr *completionError
		if cerr.Unwrap() != nil {
			t.Error("expected nil from Unwrap()")
		}
	})

	t.Run("returns wrapped error", func(t *testing.T) {
		t.Parallel()

		wrapped := errors.New("wrapped error")
		cerr := &completionError{Stage: "test", Err: wrapped}

		if !errors.Is(cerr.Unwrap(), wrapped) {
			t.Error("expected wrapped error from Unwrap()")
		}
	})
}

func TestCompletionStage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		err           error
		expectedStage string
		expectedErr   error
	}{
		{
			name:          "nil error",
			err:           nil,
			expectedStage: "",
			expectedErr:   nil,
		},
		{
			name:          "regular error",
			err:           errors.New("regular error"),
			expectedStage: events.JobFailStageReporting,
			expectedErr:   errors.New("regular error"),
		},
		{
			name:          "completion error with stage",
			err:           &completionError{Stage: "reporting", Err: errors.New("report failed")},
			expectedStage: "reporting",
			expectedErr:   errors.New("report failed"),
		},
		{
			name:          "completion error without stage",
			err:           &completionError{Stage: "", Err: errors.New("no stage")},
			expectedStage: events.JobFailStageReporting,
			expectedErr:   errors.New("no stage"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stage, err := completionStage(tt.err)

			if stage != tt.expectedStage {
				t.Errorf("expected stage %q, got %q", tt.expectedStage, stage)
			}

			switch {
			case tt.expectedErr == nil && err != nil:
				t.Errorf("expected nil error, got %v", err)
			case tt.expectedErr != nil && err == nil:
				t.Error("expected error, got nil")
			case tt.expectedErr != nil && err != nil && err.Error() != tt.expectedErr.Error():
				t.Errorf("expected error %q, got %q", tt.expectedErr.Error(), err.Error())
			}
		})
	}
}

func TestSelectPrimaryScanner_NoSuccessfulScanners(t *testing.T) {
	t.Parallel()

	orch, _, _, _ := setupTestOrchestrator(t)

	job := &models.Job{
		ID:               "job-no-success",
		ExpectedScanners: []string{"axe", "lighthouse"},
	}

	scanner, ok := orch.selectPrimaryScanner(job, []string{})
	if ok {
		t.Error("expected ok to be false when no successful scanners")
	}

	if scanner != "" {
		t.Errorf("expected empty scanner, got %s", scanner)
	}
}

func TestSelectPrimaryScanner_FirstExpectedScanner(t *testing.T) {
	t.Parallel()

	orch, _, _, _ := setupTestOrchestrator(t)

	job := &models.Job{
		ID:               "job-primary",
		ExpectedScanners: []string{"axe", "lighthouse", "pa11y"},
	}

	successfulScanners := []string{"pa11y", "lighthouse", "axe"}

	scanner, ok := orch.selectPrimaryScanner(job, successfulScanners)
	if !ok {
		t.Fatal("expected ok to be true")
	}

	// Should return first expected scanner that succeeded (axe)
	if scanner != "axe" {
		t.Errorf("expected primary scanner to be 'axe', got %s", scanner)
	}
}

func TestSelectPrimaryScanner_FallbackToAlphabetical(t *testing.T) {
	t.Parallel()

	orch, _, _, _ := setupTestOrchestrator(t)

	job := &models.Job{
		ID:               "job-fallback",
		ExpectedScanners: []string{"axe", "lighthouse"},
	}

	// Only pa11y and htmlcs succeeded (not in expected list)
	successfulScanners := []string{"pa11y", "htmlcs"}

	scanner, ok := orch.selectPrimaryScanner(job, successfulScanners)
	if !ok {
		t.Fatal("expected ok to be true")
	}

	// Should return first alphabetically (htmlcs)
	if scanner != "htmlcs" {
		t.Errorf("expected primary scanner to be 'htmlcs', got %s", scanner)
	}
}

func TestSelectPrimaryScanner_OnlyOneSuccessful(t *testing.T) {
	t.Parallel()

	orch, _, _, _ := setupTestOrchestrator(t)

	job := &models.Job{
		ID:               "job-single",
		ExpectedScanners: []string{"axe", "lighthouse", "pa11y"},
	}

	successfulScanners := []string{"lighthouse"}

	scanner, ok := orch.selectPrimaryScanner(job, successfulScanners)
	if !ok {
		t.Fatal("expected ok to be true")
	}

	if scanner != "lighthouse" {
		t.Errorf("expected primary scanner to be 'lighthouse', got %s", scanner)
	}
}

func TestSelectPrimaryScanner_PreservesOrder(t *testing.T) {
	t.Parallel()

	orch, _, _, _ := setupTestOrchestrator(t)

	job := &models.Job{
		ID:               "job-order",
		ExpectedScanners: []string{"lighthouse", "axe", "pa11y"},
	}

	// All succeeded but in different order
	successfulScanners := []string{"pa11y", "axe", "lighthouse"}

	scanner, ok := orch.selectPrimaryScanner(job, successfulScanners)
	if !ok {
		t.Fatal("expected ok to be true")
	}

	// Should return first from ExpectedScanners (lighthouse)
	if scanner != "lighthouse" {
		t.Errorf("expected primary scanner to be 'lighthouse', got %s", scanner)
	}
}

func TestCompleteJobWithAggregatedResults_AlreadyDone(t *testing.T) {
	orch, database, _, _ := setupTestOrchestrator(t)

	job := &models.Job{
		ID:    "job-already-done",
		State: models.JobStateDone,
	}

	if err := database.CreateJob(context.Background(), job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	// Should return nil without error
	if err := orch.completeJobWithAggregatedResults(context.Background(), job); err != nil {
		t.Errorf("unexpected error for already completed job: %v", err)
	}

	// State should remain DONE
	updatedJob, err := database.GetJob(context.Background(), job.ID)
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

	if err := database.CreateJob(context.Background(), job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	// Should return nil without error
	if err := orch.completeJobWithAggregatedResults(context.Background(), job); err != nil {
		t.Errorf("unexpected error for already failed job: %v", err)
	}

	// State should remain FAILED
	updatedJob, err := database.GetJob(context.Background(), job.ID)
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

	if err := database.CreateJob(context.Background(), job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	err := orch.completeJobWithAggregatedResults(context.Background(), job)
	if err == nil {
		t.Fatal("expected error when no successful scanners")
	}

	if err.Error() != "no successful scanner results found" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestSelectPrimaryScanner_EmptyExpectedScanners(t *testing.T) {
	t.Parallel()

	orch, _, _, _ := setupTestOrchestrator(t)

	job := &models.Job{
		ID:               "job-no-expected",
		ExpectedScanners: []string{},
	}

	successfulScanners := []string{"axe", "lighthouse"}

	scanner, ok := orch.selectPrimaryScanner(job, successfulScanners)
	if !ok {
		t.Fatal("expected ok to be true")
	}

	// Should fallback to alphabetical (axe)
	if scanner != "axe" {
		t.Errorf("expected primary scanner to be 'axe', got %s", scanner)
	}
}

func TestCompletionError_AsError(t *testing.T) {
	t.Parallel()

	baseErr := errors.New("base error")
	cerr := &completionError{Stage: "testing", Err: baseErr}

	var target *completionError
	if !errors.As(cerr, &target) {
		t.Error("errors.As should work with completionError")
	}

	if target.Stage != "testing" {
		t.Errorf("expected stage 'testing', got %s", target.Stage)
	}

	if !errors.Is(cerr, baseErr) {
		t.Error("errors.Is should unwrap to base error")
	}
}

func TestSelectPrimaryScanner_NilExpectedScanners(t *testing.T) {
	t.Parallel()

	orch, _, _, _ := setupTestOrchestrator(t)

	job := &models.Job{
		ID:               "job-nil-expected",
		ExpectedScanners: nil,
	}

	successfulScanners := []string{"pa11y", "axe"}

	scanner, ok := orch.selectPrimaryScanner(job, successfulScanners)
	if !ok {
		t.Fatal("expected ok to be true")
	}

	// Should fallback to alphabetical (axe)
	if scanner != "axe" {
		t.Errorf("expected primary scanner to be 'axe', got %s", scanner)
	}
}
