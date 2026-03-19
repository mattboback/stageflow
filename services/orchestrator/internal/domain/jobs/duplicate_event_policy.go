package jobs

import (
	"fmt"

	"github.com/mattboback/stageflow/libs/go/models"
)

type DuplicateJobCreatedAction string

const (
	DuplicateJobCreatedRetryOrchestration DuplicateJobCreatedAction = "retry_orchestration"
	DuplicateJobCreatedIgnore             DuplicateJobCreatedAction = "ignore"
)

func DecideDuplicateJobCreated(state models.JobState) (DuplicateJobCreatedAction, error) {
	switch state {
	case models.JobStatePending:
		return DuplicateJobCreatedRetryOrchestration, nil
	case models.JobStateExtracting,
		models.JobStateReady,
		models.JobStateScanning,
		models.JobStateCompleting,
		models.JobStateDone,
		models.JobStateFailed:
		return DuplicateJobCreatedIgnore, nil
	default:
		return "", fmt.Errorf("unsupported job state for duplicate job.created: %s", state)
	}
}
