package jobs

import (
	"strings"

	"github.com/mattboback/stageflow/libs/go/events"
)

func NormalizeFailureStage(stage string) string {
	switch stage {
	case events.JobFailStageExtraction, events.JobFailStageScanning, events.JobFailStageReporting:
		return stage
	case "":
		return events.JobFailStageScanning
	case "completing":
		return events.JobFailStageReporting
	case "setup":
		return events.JobFailStageScanning
	}

	if strings.HasPrefix(stage, "scanner-") {
		return events.JobFailStageScanning
	}

	return events.JobFailStageScanning
}
