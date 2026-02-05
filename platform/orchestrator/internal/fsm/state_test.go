package fsm

import (
	"testing"

	"github.com/mattboback/stageflow/packages/shared-go/models"
)

func TestNewStateMachine(t *testing.T) {
	sm := NewStateMachine()
	if sm == nil {
		t.Fatal("Expected state machine to be non-nil")
	}
}

func TestCanTransition_Valid(t *testing.T) {
	sm := NewStateMachine()

	tests := []struct {
		from models.JobState
		to   models.JobState
		want bool
	}{
		{models.JobStatePending, models.JobStateExtracting, true},
		{models.JobStatePending, models.JobStateReady, true},
		{models.JobStateExtracting, models.JobStateReady, true},
		{models.JobStateReady, models.JobStateScanning, true},
		{models.JobStateScanning, models.JobStateCompleting, true},
		{models.JobStateCompleting, models.JobStateDone, true},
		{models.JobStatePending, models.JobStateFailed, true},
		{models.JobStateExtracting, models.JobStateFailed, true},
		{models.JobStateScanning, models.JobStateFailed, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.from)+"->"+string(tt.to), func(t *testing.T) {
			got := sm.CanTransition(tt.from, tt.to)
			if got != tt.want {
				t.Errorf("CanTransition(%s, %s) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}

func TestCanTransition_Invalid(t *testing.T) {
	sm := NewStateMachine()

	tests := []struct {
		from models.JobState
		to   models.JobState
	}{
		{models.JobStatePending, models.JobStateScanning},    // Can't skip directly to scanning
		{models.JobStateExtracting, models.JobStateScanning}, // Can't skip ready state
		{models.JobStateDone, models.JobStatePending},        // Can't go back from terminal
		{models.JobStateFailed, models.JobStatePending},      // Can't go back from terminal
		{models.JobStateScanning, models.JobStatePending},    // Can't go backwards
	}

	for _, tt := range tests {
		t.Run(string(tt.from)+"->"+string(tt.to), func(t *testing.T) {
			if sm.CanTransition(tt.from, tt.to) {
				t.Errorf("CanTransition(%s, %s) should be false", tt.from, tt.to)
			}
		})
	}
}

func TestValidateTransition(t *testing.T) {
	sm := NewStateMachine()

	// Valid transition
	err := sm.ValidateTransition(models.JobStatePending, models.JobStateExtracting)
	if err != nil {
		t.Errorf("Expected valid transition to have no error, got: %v", err)
	}

	// Invalid transition
	err = sm.ValidateTransition(models.JobStatePending, models.JobStateScanning)
	if err == nil {
		t.Error("Expected invalid transition to return error")
	}
}

func TestNextState(t *testing.T) {
	sm := NewStateMachine()

	tests := []struct {
		current models.JobState
		event   string
		want    models.JobState
		wantErr bool
	}{
		{models.JobStatePending, "start_extraction", models.JobStateExtracting, false},
		{models.JobStateExtracting, "extraction_ready", models.JobStateReady, false},
		{models.JobStateReady, "start_scanning", models.JobStateScanning, false},
		{models.JobStateScanning, "scan_completed", models.JobStateCompleting, false},
		{models.JobStateCompleting, "artifacts_uploaded", models.JobStateDone, false},
		{models.JobStatePending, "error", models.JobStateFailed, false},
		{models.JobStateScanning, "error", models.JobStateFailed, false},
		{models.JobStatePending, "invalid_event", "", true},
	}

	for _, tt := range tests {
		t.Run(string(tt.current)+":"+tt.event, func(t *testing.T) {
			got, err := sm.NextState(tt.current, tt.event)
			if (err != nil) != tt.wantErr {
				t.Errorf("NextState(%s, %s) error = %v, wantErr %v", tt.current, tt.event, err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("NextState(%s, %s) = %s, want %s", tt.current, tt.event, got, tt.want)
			}
		})
	}
}

func TestIsTerminal(t *testing.T) {
	sm := NewStateMachine()

	tests := []struct {
		state    models.JobState
		terminal bool
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
			got := sm.IsTerminal(tt.state)
			if got != tt.terminal {
				t.Errorf("IsTerminal(%s) = %v, want %v", tt.state, got, tt.terminal)
			}
		})
	}
}

func TestGetAllowedTransitions(t *testing.T) {
	sm := NewStateMachine()

	// Test pending state has 3 allowed transitions
	transitions := sm.GetAllowedTransitions(models.JobStatePending)
	if len(transitions) != 3 {
		t.Errorf("Expected 3 transitions from PENDING, got %d", len(transitions))
	}

	// Test terminal states have no transitions
	doneTransitions := sm.GetAllowedTransitions(models.JobStateDone)
	if len(doneTransitions) != 0 {
		t.Errorf("Expected 0 transitions from DONE, got %d", len(doneTransitions))
	}

	failedTransitions := sm.GetAllowedTransitions(models.JobStateFailed)
	if len(failedTransitions) != 0 {
		t.Errorf("Expected 0 transitions from FAILED, got %d", len(failedTransitions))
	}
}

func TestStateTransitionFlow(t *testing.T) {
	sm := NewStateMachine()

	// Test happy path flow
	flow := []struct {
		from  models.JobState
		event string
		to    models.JobState
	}{
		{models.JobStatePending, "start_extraction", models.JobStateExtracting},
		{models.JobStateExtracting, "extraction_ready", models.JobStateReady},
		{models.JobStateReady, "start_scanning", models.JobStateScanning},
		{models.JobStateScanning, "scan_completed", models.JobStateCompleting},
		{models.JobStateCompleting, "artifacts_uploaded", models.JobStateDone},
	}

	currentState := models.JobStatePending
	for _, step := range flow {
		if currentState != step.from {
			t.Fatalf("Expected current state %s, got %s", step.from, currentState)
		}

		nextState, err := sm.NextState(currentState, step.event)
		if err != nil {
			t.Fatalf("Unexpected error at %s with event %s: %v", currentState, step.event, err)
		}

		if nextState != step.to {
			t.Errorf("Expected next state %s, got %s", step.to, nextState)
		}

		// Validate transition is allowed
		if !sm.CanTransition(currentState, nextState) {
			t.Errorf("Transition %s -> %s should be allowed", currentState, nextState)
		}

		currentState = nextState
	}

	// Verify we ended in terminal state
	if !sm.IsTerminal(currentState) {
		t.Errorf("Expected final state %s to be terminal", currentState)
	}
}

func TestFailureFromAnyState(t *testing.T) {
	sm := NewStateMachine()

	states := []models.JobState{
		models.JobStatePending,
		models.JobStateExtracting,
		models.JobStateReady,
		models.JobStateScanning,
		models.JobStateCompleting,
	}

	for _, state := range states {
		t.Run(string(state), func(t *testing.T) {
			// All non-terminal states should be able to transition to FAILED
			if !sm.CanTransition(state, models.JobStateFailed) {
				t.Errorf("State %s should be able to transition to FAILED", state)
			}

			// Verify error event triggers FAILED state
			nextState, err := sm.NextState(state, "error")
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if nextState != models.JobStateFailed {
				t.Errorf("Expected error event to lead to FAILED, got %s", nextState)
			}
		})
	}
}
