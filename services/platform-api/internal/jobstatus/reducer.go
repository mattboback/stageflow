package jobstatus

import (
	"fmt"
	"reflect"
	"time"

	domainjob "github.com/mattboback/stageflow/libs/go/domain/job"
	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/models"
	"github.com/mattboback/stageflow/services/platform-api/internal/status"
)

func beginSnapshot(cmd BeginJob) (*status.JobRecord, error) {
	if cmd.Payload == nil {
		return nil, fmt.Errorf("jobstatus: begin payload is required")
	}

	observedAt := normalizeObservedAt(cmd.ObservedAt)
	payload := cmd.Payload

	rec := &status.JobRecord{
		JobID:             payload.JobID,
		State:             models.JobStatePending,
		InputType:         payload.InputType,
		CreatedAt:         observedAt,
		UpdatedAt:         observedAt,
		ExpectedScanners:  cloneStrings(payload.Config.Modules),
		CompletedScanners: nil,
	}

	switch payload.InputType {
	case models.JobInputTypeZip:
		rec.State = models.JobStateExtracting
	case models.JobInputTypeURLs:
		rec.State = models.JobStateScanning
		rec.TotalPages = len(payload.URLs)
		rec.CurrentPage = 0
	}

	return rec, nil
}

func reduceSnapshot(base *status.JobRecord, signal Signal) (*status.JobRecord, bool, error) {
	if base == nil {
		base = &status.JobRecord{}
	}

	before := cloneJobRecord(base)
	next := cloneJobRecord(base)
	observedAt := normalizeObservedAt(signal.ObservedAt)

	if next.JobID == "" {
		next.JobID = signalJobID(signal)
	}

	if next.CreatedAt.IsZero() {
		next.CreatedAt = observedAt
	}

	switch signal.Kind {
	case SignalJobCreated:
		if signal.JobCreated == nil {
			return nil, false, fmt.Errorf("jobstatus: job.created payload is required")
		}

		applyJobCreated(next, signal.JobCreated, observedAt)
	case SignalExtractionReady:
		if signal.ExtractionReady == nil {
			return nil, false, fmt.Errorf("jobstatus: extraction.ready payload is required")
		}

		applyExtractionReady(next, signal.ExtractionReady, observedAt)
	case SignalExtractionFailed:
		if signal.ExtractionFailed == nil {
			return nil, false, fmt.Errorf("jobstatus: extraction.failed payload is required")
		}

		applyExtractionFailed(next, signal.ExtractionFailed, observedAt)
	case SignalScanPageCompleted:
		if signal.ScanPageCompleted == nil {
			return nil, false, fmt.Errorf("jobstatus: scan.page.completed payload is required")
		}

		applyScanPageCompleted(next, signal.ScanPageCompleted, observedAt)
	case SignalScanCompleted:
		if signal.ScanCompleted == nil {
			return nil, false, fmt.Errorf("jobstatus: scan.completed payload is required")
		}

		applyScanCompleted(next, signal.ScanCompleted, observedAt)
	case SignalScanFailed:
		if signal.ScanFailed == nil {
			return nil, false, fmt.Errorf("jobstatus: scan.failed payload is required")
		}

		applyFailure(next, events.JobFailStageScanning, signal.ScanFailed.Error, signal.ScanFailed.ErrorDetails, observedAt)
	case SignalJobCompleted:
		if signal.JobCompleted == nil {
			return nil, false, fmt.Errorf("jobstatus: job.completed payload is required")
		}

		applyJobCompleted(next, signal.JobCompleted, observedAt)
	case SignalJobFailed:
		if signal.JobFailed == nil {
			return nil, false, fmt.Errorf("jobstatus: job.failed payload is required")
		}

		applyFailure(next, signal.JobFailed.Stage, signal.JobFailed.Error, signal.JobFailed.ErrorDetails, observedAt)
	default:
		return nil, false, fmt.Errorf("jobstatus: unsupported signal kind %q", signal.Kind)
	}

	return next, !reflect.DeepEqual(before, next), nil
}

func applyJobCreated(rec *status.JobRecord, payload *events.JobCreatedPayload, observedAt time.Time) {
	rec.InputType = payload.InputType
	rec.UpdatedAt = observedAt

	if len(payload.Config.Modules) > 0 {
		rec.ExpectedScanners = cloneStrings(payload.Config.Modules)
	}

	switch payload.InputType {
	case models.JobInputTypeZip:
		advanceState(rec, models.JobStateExtracting)
	case models.JobInputTypeURLs:
		advanceState(rec, models.JobStateScanning)
		rec.TotalPages = maxInt(rec.TotalPages, len(payload.URLs))
		if rec.CurrentPage < 0 {
			rec.CurrentPage = 0
		}
	default:
		advanceState(rec, models.JobStatePending)
	}
}

func applyExtractionReady(rec *status.JobRecord, payload *events.ExtractionReadyPayload, observedAt time.Time) {
	rec.TotalPages = maxInt(rec.TotalPages, payload.TotalPages)
	rec.ExtractionStageLogKey = coalesceString(payload.StageLogPath, rec.ExtractionStageLogKey)
	rec.ExtractionRecipeKey = coalesceString(payload.RecipePath, rec.ExtractionRecipeKey)
	rec.ProvenanceKey = coalesceString(payload.ProvenanceArtifactPath, rec.ProvenanceKey)
	if rec.CurrentPage < 0 {
		rec.CurrentPage = 0
	}
	rec.UpdatedAt = observedAt
	advanceState(rec, models.JobStateReady)
}

func applyExtractionFailed(rec *status.JobRecord, payload *events.ExtractionFailedPayload, observedAt time.Time) {
	rec.ExtractionStageLogKey = coalesceString(payload.StageLogPath, rec.ExtractionStageLogKey)
	rec.ExtractionRecipeKey = coalesceString(payload.RecipePath, rec.ExtractionRecipeKey)
	applyFailure(rec, events.JobFailStageExtraction, payload.Error, payload.ErrorDetails, observedAt)
}

func applyScanPageCompleted(rec *status.JobRecord, payload *events.ScanPageCompletedPayload, observedAt time.Time) {
	rec.CurrentPage = maxInt(rec.CurrentPage, payload.PageIndex)
	rec.TotalPages = maxInt(rec.TotalPages, payload.TotalPages)
	rec.UpdatedAt = observedAt
	advanceState(rec, models.JobStateScanning)
}

func applyScanCompleted(rec *status.JobRecord, payload *events.ScanCompletedPayload, observedAt time.Time) {
	rec.TotalPages = maxInt(rec.TotalPages, payload.TotalPagesScanned)
	rec.CurrentPage = maxInt(rec.CurrentPage, payload.TotalPagesScanned)
	rec.TotalViolations = payload.Summary.TotalViolations
	rec.UpdatedAt = observedAt

	if len(rec.ExpectedScanners) == 0 && payload.ScannerType != "" {
		rec.ExpectedScanners = []string{payload.ScannerType}
	}

	rec.CompletedScanners = uniqueStringsPreserveOrder(append(rec.CompletedScanners, payload.ScannerType))
	advanceState(rec, models.JobStateScanning)
}

func applyJobCompleted(rec *status.JobRecord, payload *events.JobCompletedPayload, observedAt time.Time) {
	if rec.State == models.JobStateFailed {
		return
	}

	rec.State = models.JobStateDone
	rec.ReportJSONKey = payload.Artifacts.ReportJSON
	rec.ReportKey = payload.Artifacts.ReportHTML
	rec.ScanStageLogKey = payload.Artifacts.ScanStageLog
	rec.ScanRecipeKey = payload.Artifacts.ScanRecipe
	rec.ProvenanceKey = coalesceString(payload.Artifacts.ProvenanceJSON, rec.ProvenanceKey)
	rec.Error = ""
	rec.LastStage = ""
	rec.LastErrorDetails = ""
	rec.UpdatedAt = observedAt
	if rec.CompletedAt == nil {
		completedAt := observedAt
		rec.CompletedAt = &completedAt
	}

	if len(payload.ScannerArtifacts) > 0 {
		rec.ScannerArtifacts = make(map[string]*status.ScannerArtifactRecord, len(payload.ScannerArtifacts))
		for scannerType, artifact := range payload.ScannerArtifacts {
			rec.ScannerArtifacts[scannerType] = &status.ScannerArtifactRecord{
				ScannerType: artifact.ScannerType,
				ResultsKey:  artifact.ResultsPath,
				ReportKey:   artifact.ReportPath,
				StageLogKey: artifact.StageLogPath,
				RecipeKey:   artifact.RecipePath,
			}
		}
	}
}

func applyFailure(rec *status.JobRecord, stage, message, details string, observedAt time.Time) {
	if rec.State == models.JobStateDone {
		return
	}

	rec.State = models.JobStateFailed
	rec.Error = message
	rec.LastStage = stage
	rec.LastErrorDetails = details
	rec.UpdatedAt = observedAt
	if rec.CompletedAt == nil {
		completedAt := observedAt
		rec.CompletedAt = &completedAt
	}
}

func advanceState(rec *status.JobRecord, target models.JobState) {
	if rec == nil || rec.State == models.JobStateDone || rec.State == models.JobStateFailed {
		return
	}

	if domainjob.Order(target) > domainjob.Order(rec.State) {
		rec.State = target
	}
}

func signalJobID(signal Signal) string {
	switch signal.Kind {
	case SignalJobCreated:
		if signal.JobCreated != nil {
			return signal.JobCreated.JobID
		}
	case SignalExtractionReady:
		if signal.ExtractionReady != nil {
			return signal.ExtractionReady.JobID
		}
	case SignalExtractionFailed:
		if signal.ExtractionFailed != nil {
			return signal.ExtractionFailed.JobID
		}
	case SignalScanPageCompleted:
		if signal.ScanPageCompleted != nil {
			return signal.ScanPageCompleted.JobID
		}
	case SignalScanCompleted:
		if signal.ScanCompleted != nil {
			return signal.ScanCompleted.JobID
		}
	case SignalScanFailed:
		if signal.ScanFailed != nil {
			return signal.ScanFailed.JobID
		}
	case SignalJobCompleted:
		if signal.JobCompleted != nil {
			return signal.JobCompleted.JobID
		}
	case SignalJobFailed:
		if signal.JobFailed != nil {
			return signal.JobFailed.JobID
		}
	}

	return ""
}

func normalizeObservedAt(observedAt time.Time) time.Time {
	if observedAt.IsZero() {
		return time.Now().UTC()
	}

	return observedAt.UTC()
}

func cloneJobRecord(rec *status.JobRecord) *status.JobRecord {
	if rec == nil {
		return nil
	}

	cloned := *rec
	cloned.ExpectedScanners = cloneStrings(rec.ExpectedScanners)
	cloned.CompletedScanners = cloneStrings(rec.CompletedScanners)

	if rec.CompletedAt != nil {
		completedAt := *rec.CompletedAt
		cloned.CompletedAt = &completedAt
	}

	if rec.ScannerArtifacts != nil {
		cloned.ScannerArtifacts = make(map[string]*status.ScannerArtifactRecord, len(rec.ScannerArtifacts))
		for scannerType, artifact := range rec.ScannerArtifacts {
			if artifact == nil {
				continue
			}

			copyArtifact := *artifact
			cloned.ScannerArtifacts[scannerType] = &copyArtifact
		}
	}

	return &cloned
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	cloned := make([]string, len(values))
	copy(cloned, values)

	return cloned
}

func uniqueStringsPreserveOrder(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(values))
	unique := make([]string, 0, len(values))

	for _, value := range values {
		if value == "" {
			continue
		}

		if _, ok := seen[value]; ok {
			continue
		}

		seen[value] = struct{}{}
		unique = append(unique, value)
	}

	if len(unique) == 0 {
		return nil
	}

	return unique
}

func coalesceString(value, fallback string) string {
	if value != "" {
		return value
	}

	return fallback
}

func maxInt(left, right int) int {
	if right > left {
		return right
	}

	return left
}
