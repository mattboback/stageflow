package jobs

import (
	"testing"

	"github.com/mattboback/stageflow/packages/shared-go/events"
)

func TestNormalizeFailureStage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		stage string
		want  string
	}{
		{name: "empty defaults to scanning", stage: "", want: events.JobFailStageScanning},
		{name: "completing maps to reporting", stage: "completing", want: events.JobFailStageReporting},
		{name: "setup maps to scanning", stage: "setup", want: events.JobFailStageScanning},
		{name: "scanner component maps to scanning", stage: "scanner-axe", want: events.JobFailStageScanning},
		{name: "extraction preserved", stage: events.JobFailStageExtraction, want: events.JobFailStageExtraction},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := NormalizeFailureStage(tt.stage); got != tt.want {
				t.Fatalf("NormalizeFailureStage(%q) = %q, want %q", tt.stage, got, tt.want)
			}
		})
	}
}
