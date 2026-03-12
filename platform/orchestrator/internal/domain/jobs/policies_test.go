package jobs

import (
	"testing"

	"github.com/mattboback/stageflow/packages/shared-go/models"
)

func TestDecideDuplicateJobCreated(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		state   models.JobState
		want    DuplicateJobCreatedAction
		wantErr bool
	}{
		{
			name:  "pending retries orchestration",
			state: models.JobStatePending,
			want:  DuplicateJobCreatedRetryOrchestration,
		},
		{
			name:  "extracting ignores duplicate",
			state: models.JobStateExtracting,
			want:  DuplicateJobCreatedIgnore,
		},
		{
			name:  "terminal job ignores duplicate",
			state: models.JobStateDone,
			want:  DuplicateJobCreatedIgnore,
		},
		{
			name:  "failed job ignores duplicate",
			state: models.JobStateFailed,
			want:  DuplicateJobCreatedIgnore,
		},
		{
			name:    "unexpected state errors",
			state:   models.JobState("UNKNOWN"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := DecideDuplicateJobCreated(tt.state)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Fatalf("DecideDuplicateJobCreated() error = %v", err)
			}

			if got != tt.want {
				t.Fatalf("DecideDuplicateJobCreated() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestDecideExtractionReady(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		state   models.JobState
		want    ExtractionReadyAction
		wantErr bool
	}{
		{
			name:  "extracting advances to ready",
			state: models.JobStateExtracting,
			want:  ExtractionReadyAdvance,
		},
		{
			name:  "pending advances to ready",
			state: models.JobStatePending,
			want:  ExtractionReadyAdvance,
		},
		{
			name:  "ready is idempotent",
			state: models.JobStateReady,
			want:  ExtractionReadyAlreadyReady,
		},
		{
			name:  "scanning ignores duplicate event",
			state: models.JobStateScanning,
			want:  ExtractionReadyIgnore,
		},
		{
			name:  "failed ignores duplicate event",
			state: models.JobStateFailed,
			want:  ExtractionReadyIgnore,
		},
		{
			name:    "unexpected state errors",
			state:   models.JobState("UNKNOWN"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := DecideExtractionReady(tt.state)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Fatalf("DecideExtractionReady() error = %v", err)
			}

			if got != tt.want {
				t.Fatalf("DecideExtractionReady() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestDecideURLJobPreparation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		state   models.JobState
		want    URLJobPreparationAction
		wantErr bool
	}{
		{
			name:  "pending advances to ready",
			state: models.JobStatePending,
			want:  URLJobPreparationAdvanceToReady,
		},
		{
			name:  "extracting advances to ready",
			state: models.JobStateExtracting,
			want:  URLJobPreparationAdvanceToReady,
		},
		{
			name:  "ready is idempotent",
			state: models.JobStateReady,
			want:  URLJobPreparationAlreadyReady,
		},
		{
			name:  "scanning ignores duplicate setup",
			state: models.JobStateScanning,
			want:  URLJobPreparationIgnore,
		},
		{
			name:    "unknown state errors",
			state:   models.JobState("UNKNOWN"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := DecideURLJobPreparation(tt.state)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Fatalf("DecideURLJobPreparation() error = %v", err)
			}

			if got != tt.want {
				t.Fatalf("DecideURLJobPreparation() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestDecideExtractionStart(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		state   models.JobState
		want    ExtractionStartAction
		wantErr bool
	}{
		{
			name:  "pending advances to extracting",
			state: models.JobStatePending,
			want:  ExtractionStartAdvance,
		},
		{
			name:  "extracting is idempotent",
			state: models.JobStateExtracting,
			want:  ExtractionStartAlreadyExtracting,
		},
		{
			name:    "ready is illegal",
			state:   models.JobStateReady,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := DecideExtractionStart(tt.state)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Fatalf("DecideExtractionStart() error = %v", err)
			}

			if got != tt.want {
				t.Fatalf("DecideExtractionStart() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestValidateURLTargets(t *testing.T) {
	t.Parallel()

	t.Run("allows normal remote targets in bridge mode", func(t *testing.T) {
		t.Parallel()

		if err := ValidateURLTargets([]string{"https://example.com"}, false); err != nil {
			t.Fatalf("ValidateURLTargets() error = %v", err)
		}
	})

	t.Run("rejects loopback targets without host networking", func(t *testing.T) {
		t.Parallel()

		if err := ValidateURLTargets([]string{"http://127.0.0.1:3000"}, false); err == nil {
			t.Fatal("ValidateURLTargets() error = nil, want non-nil")
		}
	})
}

func TestDecideScanFailureCompletion(t *testing.T) {
	t.Parallel()

	t.Run("waits when more scanners are still running", func(t *testing.T) {
		t.Parallel()

		job := &models.Job{
			ScannerResults: map[string]*models.ScannerResult{
				"axe": {Success: false},
			},
		}

		got := DecideScanFailureCompletion(job, false)
		if got != ScanFailureWait {
			t.Fatalf("DecideScanFailureCompletion() = %s, want %s", got, ScanFailureWait)
		}
	})

	t.Run("completes when at least one scanner succeeded", func(t *testing.T) {
		t.Parallel()

		job := &models.Job{
			ScannerResults: map[string]*models.ScannerResult{
				"axe":        {Success: true},
				"lighthouse": {Success: false},
			},
		}

		got := DecideScanFailureCompletion(job, true)
		if got != ScanFailureCompleteWithPartialResults {
			t.Fatalf("DecideScanFailureCompletion() = %s, want %s", got, ScanFailureCompleteWithPartialResults)
		}
	})

	t.Run("fails when all scanners failed", func(t *testing.T) {
		t.Parallel()

		job := &models.Job{
			ScannerResults: map[string]*models.ScannerResult{
				"axe":        {Success: false},
				"lighthouse": {Success: false},
			},
		}

		got := DecideScanFailureCompletion(job, true)
		if got != ScanFailureFailJob {
			t.Fatalf("DecideScanFailureCompletion() = %s, want %s", got, ScanFailureFailJob)
		}
	})
}

func TestSelectPrimaryScanner(t *testing.T) {
	t.Parallel()

	t.Run("prefers first successful expected scanner", func(t *testing.T) {
		t.Parallel()

		job := &models.Job{ExpectedScanners: []string{"lighthouse", "axe"}}

		got, ok := SelectPrimaryScanner(job, []string{"axe", "lighthouse"})
		if !ok {
			t.Fatal("expected primary scanner")
		}

		if got != "lighthouse" {
			t.Fatalf("SelectPrimaryScanner() = %q, want %q", got, "lighthouse")
		}
	})

	t.Run("falls back to alphabetical order", func(t *testing.T) {
		t.Parallel()

		job := &models.Job{ExpectedScanners: []string{"seo"}}

		got, ok := SelectPrimaryScanner(job, []string{"pa11y", "axe"})
		if !ok {
			t.Fatal("expected primary scanner")
		}

		if got != "axe" {
			t.Fatalf("SelectPrimaryScanner() = %q, want %q", got, "axe")
		}
	})
}
