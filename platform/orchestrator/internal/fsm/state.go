package fsm

import (
	"fmt"

	"github.com/mattboback/stageflow/packages/shared-go/domain/job"
	"github.com/mattboback/stageflow/packages/shared-go/models"
)

// StateMachine is a thin wrapper over shared job lifecycle rules.
type StateMachine struct{}

// NewStateMachine returns a stateless lifecycle helper.
func NewStateMachine() *StateMachine {
	return &StateMachine{}
}

// CanTransition delegates to shared job lifecycle rules.
func (sm *StateMachine) CanTransition(from, to models.JobState) bool {
	return job.CanTransition(from, to)
}

// ValidateTransition returns an error for transitions rejected by CanTransition.
func (sm *StateMachine) ValidateTransition(from, to models.JobState) error {
	if !sm.CanTransition(from, to) {
		return fmt.Errorf("invalid state transition from %s to %s", from, to)
	}

	return nil
}

// NextState maps lifecycle events to expected states; "error" always yields FAILED.
func (sm *StateMachine) NextState(current models.JobState, event string) (models.JobState, error) {
	//nolint:exhaustive // Terminal states (Done, Failed) are handled by the error fallback below.
	switch current {
	case models.JobStatePending:
		if event == "start_extraction" {
			return models.JobStateExtracting, nil
		}
	case models.JobStateExtracting:
		if event == "extraction_ready" {
			return models.JobStateReady, nil
		}
	case models.JobStateReady:
		if event == "start_scanning" {
			return models.JobStateScanning, nil
		}
	case models.JobStateScanning:
		if event == "scan_completed" {
			return models.JobStateCompleting, nil
		}
	case models.JobStateCompleting:
		if event == "artifacts_uploaded" {
			return models.JobStateDone, nil
		}
	}

	// Any state can transition to FAILED on error.
	if event == "error" {
		return models.JobStateFailed, nil
	}

	return "", fmt.Errorf("no valid transition from %s on event %s", current, event)
}

// IsTerminal delegates to shared job lifecycle rules.
func (sm *StateMachine) IsTerminal(state models.JobState) bool {
	return job.IsTerminal(state)
}

// GetAllowedTransitions delegates to shared job lifecycle rules.
func (sm *StateMachine) GetAllowedTransitions(from models.JobState) []models.JobState {
	return job.AllowedTransitions(from)
}
