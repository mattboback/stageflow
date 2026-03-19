package jobs

import (
	"testing"

	"github.com/mattboback/stageflow/libs/go/models"
)

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
