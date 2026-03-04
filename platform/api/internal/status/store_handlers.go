package status

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/mattboback/stageflow/packages/shared-go/events"
	"github.com/mattboback/stageflow/packages/shared-go/models"
)

// HandleJobCreated inserts the initial projection row for a job.
func (s *Store) HandleJobCreated(ctx context.Context, payload *events.JobCreatedPayload) error {
	now := time.Now().UTC()

	// Improve UX by moving jobs out of the PENDING state immediately. Otherwise URL jobs with long-running
	// first-page scans (e.g., Lighthouse) appear "stuck" until the first scan.page.completed event arrives.
	var (
		state            models.JobState
		totalPages       int
		currentPage      int
		expectedScanners string
	)

	switch payload.InputType {
	case models.JobInputTypeZip:
		state = models.JobStateExtracting
	case models.JobInputTypeURLs:
		state = models.JobStateScanning
		totalPages = len(payload.URLs)
		currentPage = 0
	default:
		state = models.JobStatePending
	}

	if len(payload.Config.Modules) > 0 {
		expected, err := json.Marshal(payload.Config.Modules)
		if err != nil {
			return err
		}

		expectedScanners = string(expected)
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO job_status (job_id, state, input_type, created_at, updated_at, total_pages, current_page, expected_scanners, completed_scanners)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(job_id) DO NOTHING
	`, payload.JobID, state, payload.InputType, now, now, totalPages, currentPage, expectedScanners, "[]")

	return err
}

// HandleExtractionReady applies extraction.ready updates.
func (s *Store) HandleExtractionReady(ctx context.Context, payload *events.ExtractionReadyPayload) error {
	now := time.Now().UTC()
	if err := s.ensureJobRow(ctx, payload.JobID, now); err != nil {
		return err
	}

	if _, err := s.db.ExecContext(ctx, `
		UPDATE job_status
		SET total_pages = CASE
				WHEN total_pages IS NULL OR total_pages < ? THEN ?
				ELSE total_pages
			END,
			extraction_stage_log_key = COALESCE(NULLIF(?, ''), extraction_stage_log_key),
			extraction_recipe_key = COALESCE(NULLIF(?, ''), extraction_recipe_key),
			provenance_key = COALESCE(NULLIF(?, ''), provenance_key),
			current_page = CASE
				WHEN current_page IS NULL OR current_page < 0 THEN 0
				ELSE current_page
			END,
			updated_at = ?
		WHERE job_id = ?
		`, payload.TotalPages, payload.TotalPages, payload.StageLogPath, payload.RecipePath, payload.ProvenanceArtifactPath, now, payload.JobID); err != nil {
		return err
	}

	return s.advanceState(ctx, payload.JobID, models.JobStateReady, now)
}

// HandleExtractionFailed marks a job as failed during extraction.
func (s *Store) HandleExtractionFailed(ctx context.Context, payload *events.ExtractionFailedPayload) error {
	now := time.Now().UTC()
	if err := s.ensureJobRow(ctx, payload.JobID, now); err != nil {
		return err
	}

	if _, err := s.db.ExecContext(ctx, `
		UPDATE job_status
		SET extraction_stage_log_key = COALESCE(NULLIF(?, ''), extraction_stage_log_key),
		    extraction_recipe_key = COALESCE(NULLIF(?, ''), extraction_recipe_key)
		WHERE job_id = ?
		`, payload.StageLogPath, payload.RecipePath, payload.JobID); err != nil {
		return err
	}

	return s.setFailure(ctx, payload.JobID, "extraction", payload.Error, payload.ErrorDetails, now)
}

// HandleScanPageCompleted updates per-page progress.
func (s *Store) HandleScanPageCompleted(ctx context.Context, payload *events.ScanPageCompletedPayload) error {
	now := time.Now().UTC()
	if err := s.ensureJobRow(ctx, payload.JobID, now); err != nil {
		return err
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE job_status
		SET current_page = CASE
				WHEN ? > IFNULL(current_page, 0) THEN ?
				ELSE current_page
			END,
			total_pages = CASE
				WHEN ? > IFNULL(total_pages, 0) THEN ?
				ELSE total_pages
			END,
			updated_at = ?
		WHERE job_id = ?
	`, payload.PageIndex, payload.PageIndex, payload.TotalPages, payload.TotalPages, now, payload.JobID)
	if err != nil {
		return err
	}

	return s.advanceState(ctx, payload.JobID, models.JobStateScanning, now)
}

// HandleScanCompleted records per-scanner totals while keeping the overall job in
// SCANNING until the orchestrator publishes the canonical job.completed/job.failed event.
func (s *Store) HandleScanCompleted(ctx context.Context, payload *events.ScanCompletedPayload) error {
	now := time.Now().UTC()
	if err := s.ensureJobRow(ctx, payload.JobID, now); err != nil {
		return err
	}

	rec, err := s.GetJob(ctx, payload.JobID)
	if err != nil {
		return err
	}

	completedScanners := append(cloneStringSlice(rec.CompletedScanners), payload.ScannerType)
	completedScanners = uniqueStringsPreserveOrder(completedScanners)

	completedJSON, err := json.Marshal(completedScanners)
	if err != nil {
		return err
	}

	if len(rec.ExpectedScanners) == 0 && payload.ScannerType != "" {
		rec.ExpectedScanners = []string{payload.ScannerType}
	}

	expectedJSON, err := json.Marshal(rec.ExpectedScanners)
	if err != nil {
		return err
	}

	if _, execErr := s.db.ExecContext(ctx, `
		UPDATE job_status
		SET total_pages = CASE
				WHEN ? > IFNULL(total_pages, 0) THEN ?
				ELSE total_pages
			END,
			current_page = CASE
				WHEN ? > IFNULL(current_page, 0) THEN ?
				ELSE current_page
			END,
			total_violations = ?,
			expected_scanners = COALESCE(NULLIF(?, ''), expected_scanners),
			completed_scanners = ?,
			updated_at = ?
		WHERE job_id = ?
		`, payload.TotalPagesScanned, payload.TotalPagesScanned, payload.TotalPagesScanned, payload.TotalPagesScanned, payload.Summary.TotalViolations, string(expectedJSON), string(completedJSON), now, payload.JobID); execErr != nil {
		return execErr
	}

	return s.advanceState(ctx, payload.JobID, models.JobStateScanning, now)
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

// HandleScanFailed marks a job as failed during scanning.
func (s *Store) HandleScanFailed(ctx context.Context, payload *events.ScanFailedPayload) error {
	now := time.Now().UTC()
	if err := s.ensureJobRow(ctx, payload.JobID, now); err != nil {
		return err
	}

	return s.setFailure(ctx, payload.JobID, "scanning", payload.Error, payload.ErrorDetails, now)
}

// HandleJobCompleted records final artifacts and marks the job DONE.
func (s *Store) HandleJobCompleted(ctx context.Context, payload *events.JobCompletedPayload) error {
	now := time.Now().UTC()
	if err := s.ensureJobRow(ctx, payload.JobID, now); err != nil {
		return err
	}

	var scannerArtifactsJSON sql.NullString

	if len(payload.ScannerArtifacts) > 0 {
		// scanner_artifacts is stored as a JSON map for forward-compatible per-scanner URLs.
		artifacts := make(map[string]*ScannerArtifactRecord)

		for scannerType, sa := range payload.ScannerArtifacts {
			artifacts[scannerType] = &ScannerArtifactRecord{
				ScannerType: sa.ScannerType,
				ResultsKey:  sa.ResultsPath,
				ReportKey:   sa.ReportPath,
				StageLogKey: sa.StageLogPath,
				RecipeKey:   sa.RecipePath,
			}
		}

		jsonBytes, err := json.Marshal(artifacts)
		if err != nil {
			slog.Warn("Failed to marshal scanner artifacts", "error", err, "job_id", payload.JobID)
		} else {
			scannerArtifactsJSON = sql.NullString{String: string(jsonBytes), Valid: true}
		}
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE job_status
		SET state = ?,
			report_json_key = ?,
			report_key = ?,
			scan_stage_log_key = ?,
			scan_recipe_key = ?,
			provenance_key = COALESCE(NULLIF(?, ''), provenance_key),
			scanner_artifacts = COALESCE(?, scanner_artifacts),
			error = '',
			last_stage = '',
			last_error_details = '',
			completed_at = COALESCE(completed_at, ?),
			updated_at = ?
		WHERE job_id = ?
		  AND state != ?
	`, models.JobStateDone,
		payload.Artifacts.ReportJSON,
		payload.Artifacts.ReportHTML,
		payload.Artifacts.ScanStageLog,
		payload.Artifacts.ScanRecipe,
		payload.Artifacts.ProvenanceJSON,
		scannerArtifactsJSON,
		now,
		now,
		payload.JobID,
		models.JobStateFailed,
	)

	return err
}

// HandleJobFailed records the canonical failure event.
func (s *Store) HandleJobFailed(ctx context.Context, payload *events.JobFailedPayload) error {
	now := time.Now().UTC()
	if err := s.ensureJobRow(ctx, payload.JobID, now); err != nil {
		return err
	}

	return s.setFailure(ctx, payload.JobID, payload.Stage, payload.Error, payload.ErrorDetails, now)
}
