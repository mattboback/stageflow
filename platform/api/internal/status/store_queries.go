package status

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/mattboback/stageflow/packages/shared-go/domain/job"
	"github.com/mattboback/stageflow/packages/shared-go/models"
)

// GetJob retrieves a projection entry.
func (s *Store) GetJob(ctx context.Context, jobID string) (*JobRecord, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT job_id, state, input_type, created_at, updated_at, completed_at,
		       IFNULL(error, ''), IFNULL(total_pages, 0), IFNULL(current_page, 0), IFNULL(total_violations, 0),
		       IFNULL(report_json_key, ''), IFNULL(report_key, ''), IFNULL(scan_stage_log_key, ''), IFNULL(scan_recipe_key, ''),
		       IFNULL(extraction_stage_log_key, ''), IFNULL(extraction_recipe_key, ''), IFNULL(provenance_key, ''),
		       IFNULL(last_stage, ''), IFNULL(last_error_details, ''), IFNULL(scanner_artifacts, '')
		FROM job_status
		WHERE job_id = ?
	`, jobID)

	var (
		rec                  JobRecord
		completedAt          sql.NullTime
		scannerArtifactsJSON string
	)
	if err := row.Scan(
		&rec.JobID,
		&rec.State,
		&rec.InputType,
		&rec.CreatedAt,
		&rec.UpdatedAt,
		&completedAt,
		&rec.Error,
		&rec.TotalPages,
		&rec.CurrentPage,
		&rec.TotalViolations,
		&rec.ReportJSONKey,
		&rec.ReportKey,
		&rec.ScanStageLogKey,
		&rec.ScanRecipeKey,
		&rec.ExtractionStageLogKey,
		&rec.ExtractionRecipeKey,
		&rec.ProvenanceKey,
		&rec.LastStage,
		&rec.LastErrorDetails,
		&scannerArtifactsJSON,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrJobNotFound
		}

		return nil, fmt.Errorf("scan job record: %w", err)
	}

	if completedAt.Valid {
		rec.CompletedAt = &completedAt.Time
	}

	if scannerArtifactsJSON != "" {
		if err := json.Unmarshal([]byte(scannerArtifactsJSON), &rec.ScannerArtifacts); err != nil {
			slog.Warn("Failed to unmarshal scanner artifacts", "error", err, "job_id", jobID)
		}
	}

	return &rec, nil
}

func (s *Store) ensureJobRow(ctx context.Context, jobID string, ts time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO job_status (job_id, state, input_type, created_at, updated_at)
		VALUES (?, ?, '', ?, ?)
		ON CONFLICT(job_id) DO NOTHING
	`, jobID, models.JobStatePending, ts, ts)

	return err
}

func (s *Store) advanceState(ctx context.Context, jobID string, target models.JobState, ts time.Time) error {
	rank := job.Order(target)
	if rank < 0 {
		return nil
	}
	// #nosec G201 -- StateRankSQL returns a static CASE expression; safe to inline for ordering.
	query := fmt.Sprintf(`
		UPDATE job_status
		SET state = ?, updated_at = ?
		WHERE job_id = ?
		  AND state NOT IN (?, ?)
		  AND %s < ?
	`, job.StateRankSQL())

	_, err := s.db.ExecContext(ctx, query, target, ts, jobID, models.JobStateDone, models.JobStateFailed, rank)

	return err
}

func (s *Store) setFailure(ctx context.Context, jobID, stage, message, details string, ts time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE job_status
		SET state = ?,
			error = ?,
			last_stage = ?,
			last_error_details = ?,
			completed_at = COALESCE(completed_at, ?),
			updated_at = ?
		WHERE job_id = ?
		  AND state != ?
	`, models.JobStateFailed, message, stage, details, ts, ts, jobID, models.JobStateDone)

	return err
}
