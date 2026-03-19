package jobs

import (
	"sort"

	"github.com/mattboback/stageflow/libs/go/models"
)

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
