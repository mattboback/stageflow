package jobs

import (
	"testing"

	"github.com/mattboback/stageflow/libs/go/models"
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
