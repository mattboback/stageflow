package job

import (
	"testing"

	"github.com/mattboback/stageflow/packages/shared-go/models"
)

func TestOrder(t *testing.T) {
	tests := []struct {
		state    models.JobState
		expected int
	}{
		{models.JobStatePending, 0},
		{models.JobStateExtracting, 1},
		{models.JobStateReady, 2},
		{models.JobStateScanning, 3},
		{models.JobStateCompleting, 4},
		{models.JobStateDone, 5},
		{models.JobStateFailed, 5},
		{"UNKNOWN", -1},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			got := Order(tt.state)
			if got != tt.expected {
				t.Errorf("Order(%s) = %d, want %d", tt.state, got, tt.expected)
			}
		})
	}
}

func TestIsTerminal(t *testing.T) {
	tests := []struct {
		state    models.JobState
		expected bool
	}{
		{models.JobStatePending, false},
		{models.JobStateExtracting, false},
		{models.JobStateReady, false},
		{models.JobStateScanning, false},
		{models.JobStateCompleting, false},
		{models.JobStateDone, true},
		{models.JobStateFailed, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			got := IsTerminal(tt.state)
			if got != tt.expected {
				t.Errorf("IsTerminal(%s) = %v, want %v", tt.state, got, tt.expected)
			}
		})
	}
}

func TestCanTransition(t *testing.T) {
	validTransitions := []struct {
		from, to models.JobState
	}{
		{models.JobStatePending, models.JobStateExtracting},
		{models.JobStatePending, models.JobStateReady},
		{models.JobStatePending, models.JobStateFailed},
		{models.JobStateExtracting, models.JobStateReady},
		{models.JobStateExtracting, models.JobStateFailed},
		{models.JobStateReady, models.JobStateScanning},
		{models.JobStateReady, models.JobStateFailed},
		{models.JobStateScanning, models.JobStateCompleting},
		{models.JobStateScanning, models.JobStateFailed},
		{models.JobStateCompleting, models.JobStateDone},
		{models.JobStateCompleting, models.JobStateFailed},
	}

	for _, tt := range validTransitions {
		name := string(tt.from) + " -> " + string(tt.to)
		t.Run(name, func(t *testing.T) {
			if !CanTransition(tt.from, tt.to) {
				t.Errorf("CanTransition(%s, %s) = false, want true", tt.from, tt.to)
			}
		})
	}

	invalidTransitions := []struct {
		from, to models.JobState
	}{
		{models.JobStateDone, models.JobStatePending},
		{models.JobStateDone, models.JobStateFailed},
		{models.JobStateFailed, models.JobStatePending},
		{models.JobStateFailed, models.JobStateDone},
		{models.JobStatePending, models.JobStateScanning},
		{models.JobStatePending, models.JobStateCompleting},
		{models.JobStatePending, models.JobStateDone},
		{models.JobStateExtracting, models.JobStateScanning},
		{models.JobStateReady, models.JobStateCompleting},
		{models.JobStateScanning, models.JobStateDone},
	}

	for _, tt := range invalidTransitions {
		name := string(tt.from) + " -> " + string(tt.to)
		t.Run(name, func(t *testing.T) {
			if CanTransition(tt.from, tt.to) {
				t.Errorf("CanTransition(%s, %s) = true, want false", tt.from, tt.to)
			}
		})
	}
}

func TestAllowedTransitions(t *testing.T) {
	pending := AllowedTransitions(models.JobStatePending)
	if len(pending) != 3 {
		t.Errorf("AllowedTransitions(PENDING) = %d items, want 3", len(pending))
	}

	done := AllowedTransitions(models.JobStateDone)
	if len(done) != 0 {
		t.Errorf("AllowedTransitions(DONE) = %d items, want 0", len(done))
	}

	failed := AllowedTransitions(models.JobStateFailed)
	if len(failed) != 0 {
		t.Errorf("AllowedTransitions(FAILED) = %d items, want 0", len(failed))
	}

	unknown := AllowedTransitions("UNKNOWN")
	if len(unknown) != 0 {
		t.Errorf("AllowedTransitions(UNKNOWN) = %d items, want 0", len(unknown))
	}
}

func TestIsLaterThan(t *testing.T) {
	tests := []struct {
		a, b     models.JobState
		expected bool
	}{
		{models.JobStateExtracting, models.JobStatePending, true},
		{models.JobStateReady, models.JobStateExtracting, true},
		{models.JobStateDone, models.JobStateScanning, true},
		{models.JobStatePending, models.JobStateExtracting, false},
		{models.JobStatePending, models.JobStatePending, false},
		{models.JobStateDone, models.JobStateFailed, false},
	}

	for _, tt := range tests {
		name := string(tt.a) + " > " + string(tt.b)
		t.Run(name, func(t *testing.T) {
			got := IsLaterThan(tt.a, tt.b)
			if got != tt.expected {
				t.Errorf("IsLaterThan(%s, %s) = %v, want %v", tt.a, tt.b, got, tt.expected)
			}
		})
	}
}

func TestStateRankSQL(t *testing.T) {
	sql := StateRankSQL()
	if sql == "" {
		t.Error("StateRankSQL() returned empty string")
	}
	if len(sql) < 50 {
		t.Errorf("StateRankSQL() returned unexpectedly short string: %s", sql)
	}
}
