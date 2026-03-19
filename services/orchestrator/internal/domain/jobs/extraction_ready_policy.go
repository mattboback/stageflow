package jobs

import (
	"fmt"

	"github.com/mattboback/stageflow/libs/go/models"
)

type ExtractionReadyAction string

const (
	ExtractionReadyAdvance      ExtractionReadyAction = "advance"
	ExtractionReadyAlreadyReady ExtractionReadyAction = "already_ready"
	ExtractionReadyIgnore       ExtractionReadyAction = "ignore"
)

func DecideExtractionReady(state models.JobState) (ExtractionReadyAction, error) {
	switch state {
	case models.JobStateReady:
		return ExtractionReadyAlreadyReady, nil
	case models.JobStateScanning,
		models.JobStateCompleting,
		models.JobStateDone,
		models.JobStateFailed:
		return ExtractionReadyIgnore, nil
	case models.JobStatePending, models.JobStateExtracting:
		if !CanTransitionTo(state, models.JobStateReady) {
			return "", fmt.Errorf("job cannot transition to READY from %s", state)
		}

		return ExtractionReadyAdvance, nil
	default:
		return "", fmt.Errorf("job cannot transition to READY from %s", state)
	}
}
