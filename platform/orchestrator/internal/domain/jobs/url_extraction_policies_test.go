package jobs

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/mattboback/stageflow/packages/shared-go/models"
)

func TestURLExtractionPoliciesUseDomainTransitionHelper(t *testing.T) {
	t.Parallel()

	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("expected caller information")
	}

	sourcePath := filepath.Join(filepath.Dir(testFile), "url_extraction_policies.go")
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if strings.Contains(string(source), "sharedjob.CanTransition(") {
		t.Fatal("expected URL extraction policies to use CanTransitionTo instead of sharedjob.CanTransition")
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
