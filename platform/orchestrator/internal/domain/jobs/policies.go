package jobs

import (
	"fmt"
	"sort"

	sharedjob "github.com/mattboback/stageflow/packages/shared-go/domain/job"
	"github.com/mattboback/stageflow/packages/shared-go/models"
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
		if !sharedjob.CanTransition(state, models.JobStateReady) {
			return "", fmt.Errorf("job cannot transition to READY from %s", state)
		}

		return ExtractionReadyAdvance, nil
	default:
		return "", fmt.Errorf("job cannot transition to READY from %s", state)
	}
}

type ScanFailureAction string

const (
	ScanFailureWait                       ScanFailureAction = "wait"
	ScanFailureCompleteWithPartialResults ScanFailureAction = "complete_with_partial_results"
	ScanFailureFailJob                    ScanFailureAction = "fail_job"
)

func DecideScanFailureCompletion(job *models.Job, allComplete bool) ScanFailureAction {
	if !allComplete {
		return ScanFailureWait
	}

	if job == nil {
		return ScanFailureFailJob
	}

	for _, result := range job.ScannerResults {
		if result != nil && result.Success {
			return ScanFailureCompleteWithPartialResults
		}
	}

	return ScanFailureFailJob
}

func SelectPrimaryScanner(job *models.Job, successfulScanners []string) (string, bool) {
	if len(successfulScanners) == 0 {
		return "", false
	}

	successSet := make(map[string]struct{}, len(successfulScanners))
	for _, scanner := range successfulScanners {
		successSet[scanner] = struct{}{}
	}

	if job != nil {
		for _, expected := range job.ExpectedScanners {
			if _, ok := successSet[expected]; ok {
				return expected, true
			}
		}
	}

	sort.Strings(successfulScanners)

	return successfulScanners[0], true
}
