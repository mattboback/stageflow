package jobstatus

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	domainjob "github.com/mattboback/stageflow/libs/go/domain/job"
	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/models"
	"github.com/mattboback/stageflow/services/platform-api/internal/status"
)

// beginSnapshot seeds the cache with a PENDING record. It does NOT advance the
// state; that is the job of Apply + applyJobCreated so the broker detects a
// change and publishes to SSE watchers.
func beginSnapshot(cmd BeginJob) (*status.JobRecord, error) {
	if cmd.Payload == nil {
		return nil, errors.New("jobstatus: begin payload is required")
	}

	observedAt := normalizeObservedAt(cmd.ObservedAt)
	payload := cmd.Payload

	rec := &status.JobRecord{
		JobID:             payload.JobID,
		State:             models.JobStatePending,
		InputType:         payload.InputType,
		CreatedAt:         observedAt,
		UpdatedAt:         observedAt,
		ExpectedScanners:  status.CloneStrings(payload.Config.Modules),
		CompletedScanners: nil,
	}

	if payload.InputType == models.JobInputTypeURLs {
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

	if err := applySignal(next, signal, observedAt); err != nil {
		return nil, false, err
	}

	return next, !reflect.DeepEqual(before, next), nil
}

func applySignal(rec *status.JobRecord, signal Signal, observedAt time.Time) error {
	switch signal.Kind {
	case SignalJobCreated:
		return applyJobCreatedSignal(rec, signal.JobCreated, observedAt)
	case SignalExtractionReady:
		return applyExtractionReadySignal(rec, signal.ExtractionReady, observedAt)
	case SignalExtractionFailed:
		return applyExtractionFailedSignal(rec, signal.ExtractionFailed, observedAt)
	case SignalScanPageCompleted:
		return applyScanPageCompletedSignal(rec, signal.ScanPageCompleted, observedAt)
	case SignalScanCompleted:
		return applyScanCompletedSignal(rec, signal.ScanCompleted, observedAt)
	case SignalScanFailed:
		return applyScanFailedSignal(rec, signal.ScanFailed, observedAt)
	case SignalJobCompleted:
		return applyJobCompletedSignal(rec, signal.JobCompleted, observedAt)
	case SignalJobFailed:
		return applyJobFailedSignal(rec, signal.JobFailed, observedAt)
	default:
		return fmt.Errorf("jobstatus: unsupported signal kind %q", signal.Kind)
	}
}

func applyJobCreatedSignal(
	rec *status.JobRecord,
	payload *events.JobCreatedPayload,
	observedAt time.Time,
) error {
	if payload == nil {
		return errors.New("jobstatus: job.created payload is required")
	}

	applyJobCreated(rec, payload, observedAt)

	return nil
}

func applyExtractionReadySignal(
	rec *status.JobRecord,
	payload *events.ExtractionReadyPayload,
	observedAt time.Time,
) error {
	if payload == nil {
		return errors.New("jobstatus: extraction.ready payload is required")
	}

	applyExtractionReady(rec, payload, observedAt)

	return nil
}

func applyExtractionFailedSignal(
	rec *status.JobRecord,
	payload *events.ExtractionFailedPayload,
	observedAt time.Time,
) error {
	if payload == nil {
		return errors.New("jobstatus: extraction.failed payload is required")
	}

	applyExtractionFailed(rec, payload, observedAt)

	return nil
}

func applyScanPageCompletedSignal(
	rec *status.JobRecord,
	payload *events.ScanPageCompletedPayload,
	observedAt time.Time,
) error {
	if payload == nil {
		return errors.New("jobstatus: scan.page.completed payload is required")
	}

	applyScanPageCompleted(rec, payload, observedAt)

	return nil
}

func applyScanCompletedSignal(
	rec *status.JobRecord,
	payload *events.ScanCompletedPayload,
	observedAt time.Time,
) error {
	if payload == nil {
		return errors.New("jobstatus: scan.completed payload is required")
	}

	applyScanCompleted(rec, payload, observedAt)

	return nil
}

func applyScanFailedSignal(
	rec *status.JobRecord,
	payload *events.ScanFailedPayload,
	observedAt time.Time,
) error {
	if payload == nil {
		return errors.New("jobstatus: scan.failed payload is required")
	}

	applyFailure(
		rec,
		events.JobFailStageScanning,
		payload.Error,
		payload.ErrorDetails,
		observedAt,
	)

	return nil
}

func applyJobCompletedSignal(
	rec *status.JobRecord,
	payload *events.JobCompletedPayload,
	observedAt time.Time,
) error {
	if payload == nil {
		return errors.New("jobstatus: job.completed payload is required")
	}

	applyJobCompleted(rec, payload, observedAt)

	return nil
}

func applyJobFailedSignal(
	rec *status.JobRecord,
	payload *events.JobFailedPayload,
	observedAt time.Time,
) error {
	if payload == nil {
		return errors.New("jobstatus: job.failed payload is required")
	}

	applyFailure(rec, payload.Stage, payload.Error, payload.ErrorDetails, observedAt)

	return nil
}

func applyJobCreated(rec *status.JobRecord, payload *events.JobCreatedPayload, observedAt time.Time) {
	rec.InputType = payload.InputType
	rec.UpdatedAt = observedAt

	if len(payload.Config.Modules) > 0 {
		rec.ExpectedScanners = status.CloneStrings(payload.Config.Modules)
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

	alreadyCompleted := containsString(rec.CompletedScanners, payload.ScannerType)
	if !alreadyCompleted {
		rec.TotalViolations += payload.Summary.TotalViolations
	}

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
		return jobCreatedJobID(signal.JobCreated)
	case SignalExtractionReady:
		return extractionReadyJobID(signal.ExtractionReady)
	case SignalExtractionFailed:
		return extractionFailedJobID(signal.ExtractionFailed)
	case SignalScanPageCompleted:
		return scanPageCompletedJobID(signal.ScanPageCompleted)
	case SignalScanCompleted:
		return scanCompletedJobID(signal.ScanCompleted)
	case SignalScanFailed:
		return scanFailedJobID(signal.ScanFailed)
	case SignalJobCompleted:
		return jobCompletedJobID(signal.JobCompleted)
	case SignalJobFailed:
		return jobFailedJobID(signal.JobFailed)
	}

	return ""
}

func jobCreatedJobID(payload *events.JobCreatedPayload) string {
	if payload == nil {
		return ""
	}

	return payload.JobID
}

func extractionReadyJobID(payload *events.ExtractionReadyPayload) string {
	if payload == nil {
		return ""
	}

	return payload.JobID
}

func extractionFailedJobID(payload *events.ExtractionFailedPayload) string {
	if payload == nil {
		return ""
	}

	return payload.JobID
}

func scanPageCompletedJobID(payload *events.ScanPageCompletedPayload) string {
	if payload == nil {
		return ""
	}

	return payload.JobID
}

func scanCompletedJobID(payload *events.ScanCompletedPayload) string {
	if payload == nil {
		return ""
	}

	return payload.JobID
}

func scanFailedJobID(payload *events.ScanFailedPayload) string {
	if payload == nil {
		return ""
	}

	return payload.JobID
}

func jobCompletedJobID(payload *events.JobCompletedPayload) string {
	if payload == nil {
		return ""
	}

	return payload.JobID
}

func jobFailedJobID(payload *events.JobFailedPayload) string {
	if payload == nil {
		return ""
	}

	return payload.JobID
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
	cloned.ExpectedScanners = status.CloneStrings(rec.ExpectedScanners)
	cloned.CompletedScanners = status.CloneStrings(rec.CompletedScanners)

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

func containsString(values []string, needle string) bool {
	if needle == "" {
		return false
	}

	for _, value := range values {
		if value == needle {
			return true
		}
	}

	return false
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
