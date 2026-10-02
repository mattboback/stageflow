package jobstore

import (
	"testing"
	"time"

	"github.com/mattboback/stageflow/libs/go/models"
)

func TestClampPercentage(t *testing.T) {
	cases := []struct {
		current  int
		total    int
		expected int
	}{
		{-1, 10, 0},
		{0, 0, 0},
		{5, 10, 50},
		{15, 10, 100},
	}

	for _, tc := range cases {
		if got := clampPercentage(tc.current, tc.total); got != tc.expected {
			t.Fatalf("clampPercentage(%d, %d) = %d, want %d", tc.current, tc.total, got, tc.expected)
		}
	}
}

func TestJobRecordToModel(t *testing.T) {
	now := time.Now().UTC()
	rec := &JobRecord{
		JobID:             "abc",
		State:             models.JobStateScanning,
		Error:             "oops",
		TotalPages:        5,
		CurrentPage:       2,
		ExpectedScanners:  []string{"axe", "lighthouse"},
		CompletedScanners: []string{"axe"},
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	model := rec.ToModel()
	if model.ID != rec.JobID || model.State != rec.State {
		t.Fatalf("basic fields not copied: %#v", model)
	}

	if model.Progress == nil || model.Progress.Percentage != 40 {
		t.Fatalf("expected progress at 40%%, got %#v", model.Progress)
	}

	if len(model.RemainingScanners) != 1 || model.RemainingScanners[0] != "lighthouse" {
		t.Fatalf("expected remaining scanner lighthouse, got %#v", model.RemainingScanners)
	}

	rec.TotalPages = 0

	model = rec.ToModel()
	if model.Progress != nil {
		t.Fatalf("expected nil progress when total pages is zero, got %#v", model.Progress)
	}
}
