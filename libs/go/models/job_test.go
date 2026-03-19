package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestJobState_ValuesAndHelpers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		state      JobState
		want       string
		isValid    bool
		isTerminal bool
	}{
		{"Pending", JobStatePending, "PENDING", true, false},
		{"Extracting", JobStateExtracting, "EXTRACTING", true, false},
		{"Ready", JobStateReady, "READY_TO_SCAN", true, false},
		{"Scanning", JobStateScanning, "SCANNING", true, false},
		{"Completing", JobStateCompleting, "COMPLETING", true, false},
		{"Done", JobStateDone, "DONE", true, true},
		{"Failed", JobStateFailed, "FAILED", true, true},
		{"Unknown", JobState("???"), "???", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if string(tt.state) != tt.want {
				t.Fatalf("Expected %s, got %s", tt.want, string(tt.state))
			}

			if tt.state.IsValid() != tt.isValid {
				t.Fatalf("IsValid mismatch for %q", tt.state)
			}

			if tt.state.IsTerminal() != tt.isTerminal {
				t.Fatalf("IsTerminal mismatch for %q", tt.state)
			}
		})
	}
}

func TestJob_JSONTagsAndOmitEmpty(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 12, 25, 10, 0, 0, 0, time.UTC)
	job := &Job{
		ID:        "job-123",
		State:     JobStatePending,
		InputType: JobInputTypeZip,
		InputPath: "staging/job-123/test.zip",
		Config: JobConfig{
			Modules:    []string{"axe"},
			Screenshot: true,
		},
		CreatedAt: now,
		UpdatedAt: now,
		// URLs nil => should be omitted
		// CompletedAt nil => should be omitted
	}

	b, err := json.Marshal(job)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]any
	if unmarshalErr := json.Unmarshal(b, &m); unmarshalErr != nil {
		t.Fatalf("unmarshal: %v", unmarshalErr)
	}

	// Required / expected keys
	for _, k := range []string{"id", "state", "input_type", "input_path", "config", "created_at", "updated_at"} {
		if _, ok := m[k]; !ok {
			t.Fatalf("expected key %q in JSON: %s", k, string(b))
		}
	}

	// Omitempty checks
	if _, ok := m["urls"]; ok {
		t.Fatalf("expected urls to be omitted when nil: %s", string(b))
	}

	if _, ok := m["completed_at"]; ok {
		t.Fatalf("expected completed_at to be omitted when nil: %s", string(b))
	}
}
