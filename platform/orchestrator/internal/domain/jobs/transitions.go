package jobs

import (
	sharedjob "github.com/mattboback/stageflow/packages/shared-go/domain/job"
	"github.com/mattboback/stageflow/packages/shared-go/events"
	"github.com/mattboback/stageflow/packages/shared-go/models"
)

func CanTransitionTo(from, to models.JobState) bool {
	return sharedjob.CanTransition(from, to)
}

func CanEnterCompleting(state models.JobState) bool {
	return CanTransitionTo(state, models.JobStateCompleting)
}

func IsTerminalState(state models.JobState) bool {
	return sharedjob.IsTerminal(state)
}

func ShouldIgnoreTerminalEvent(state models.JobState, eventName string) bool {
	if !IsTerminalState(state) {
		return false
	}

	switch eventName {
	case events.EventScanCompleted, events.EventScanFailed, events.EventScanPageCompleted:
		return true
	default:
		return false
	}
}
