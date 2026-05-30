package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	jobstate "github.com/mattboback/stageflow/libs/go/domain/job"
	"github.com/mattboback/stageflow/libs/go/models"
)

func (d *Database) ensureJobExists(ctx context.Context, jobID string) error {
	var id string

	err := d.queryRowContext(ctx, `SELECT id FROM jobs WHERE id = ?`, jobID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("job not found: %s", jobID)
	}

	if err != nil {
		return fmt.Errorf("failed to check job exists: %w", err)
	}

	return nil
}

func (d *Database) getJobState(ctx context.Context, jobID string) (models.JobState, error) {
	var state models.JobState

	err := d.queryRowContext(ctx, `SELECT state FROM jobs WHERE id = ?`, jobID).Scan(&state)

	return state, err
}

func (d *Database) execJobUpdate(ctx context.Context, jobID, query string, args ...any) error {
	result, err := d.execContext(ctx, query, args...)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows != 0 {
		return nil
	}

	// Some updates may become no-ops when the DB stores timestamps at lower
	// precision. Treat "no change" as success as long as the job exists.
	return d.ensureJobExists(ctx, jobID)
}

// UpdateJobState updates the job state and updated_at timestamp.
func (d *Database) UpdateJobState(ctx context.Context, jobID string, state models.JobState) error {
	targetRank := jobstate.Order(state)
	if targetRank < 0 {
		return fmt.Errorf("unsupported job state: %s", state)
	}

	query := `
		UPDATE jobs
		SET state = ?, updated_at = ?
		WHERE id = ?
		  AND state NOT IN (?, ?)
		  AND ` + jobstate.StateRankSQL() + ` <= ?
	`

	result, err := d.execContext(
		ctx,
		query,
		state,
		time.Now(),
		jobID,
		models.JobStateDone,
		models.JobStateFailed,
		targetRank,
	)
	if err != nil {
		return fmt.Errorf("failed to update job state: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows != 0 {
		return nil
	}

	current, stateErr := d.getJobState(ctx, jobID)
	if errors.Is(stateErr, sql.ErrNoRows) {
		return fmt.Errorf("job not found: %s", jobID)
	}

	if stateErr != nil {
		return fmt.Errorf("failed to load job state: %w", stateErr)
	}

	if current == state {
		return nil
	}

	if jobstate.IsTerminal(current) {
		return fmt.Errorf("job %s is terminal (state=%s), refusing transition to %s", jobID, current, state)
	}

	return fmt.Errorf("job %s not eligible for transition to %s from %s", jobID, state, current)
}

// ClaimJobCompletion atomically grants ownership of the SCANNING -> COMPLETING
// work. Duplicate scanner terminal events see claimed=false and must not
// aggregate, cleanup, or publish terminal events.
func (d *Database) ClaimJobCompletion(ctx context.Context, jobID string) (bool, error) {
	now := time.Now()

	result, err := d.execContext(
		ctx,
		`
			UPDATE jobs
			SET state = ?, updated_at = ?
			WHERE id = ?
			  AND state = ?
		`,
		models.JobStateCompleting,
		now,
		jobID,
		models.JobStateScanning,
	)
	if err != nil {
		return false, fmt.Errorf("failed to claim job completion: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows != 0 {
		return true, nil
	}

	state, err := d.getJobState(ctx, jobID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, fmt.Errorf("job not found: %s", jobID)
	}

	if err != nil {
		return false, fmt.Errorf("failed to load job state: %w", err)
	}

	switch state {
	case models.JobStateCompleting, models.JobStateDone, models.JobStateFailed:
		return false, nil
	default:
		return false, fmt.Errorf("job %s not eligible for completion claim (state=%s)", jobID, state)
	}
}

// UpdateJobPodID updates the job's pod ID.
func (d *Database) UpdateJobPodID(ctx context.Context, jobID, podID string) error {
	query := `
		UPDATE jobs
		SET pod_id = ?, updated_at = ?
		WHERE id = ?
	`

	if err := d.execJobUpdate(ctx, jobID, query, podID, time.Now(), jobID); err != nil {
		return fmt.Errorf("failed to update job pod ID: %w", err)
	}

	return nil
}

// UpdateJobProvenance updates the job's provenance path.
func (d *Database) UpdateJobProvenance(ctx context.Context, jobID, path string) error {
	query := `
		UPDATE jobs
		SET provenance_path = ?, updated_at = ?
		WHERE id = ?
	`

	if err := d.execJobUpdate(ctx, jobID, query, path, time.Now(), jobID); err != nil {
		return fmt.Errorf("failed to update job provenance: %w", err)
	}

	return nil
}

// UpdateJobProvenanceKey stores the MinIO object key for provenance.json.
func (d *Database) UpdateJobProvenanceKey(ctx context.Context, jobID, key string) error {
	query := `
		UPDATE jobs
		SET provenance_key = ?, updated_at = ?
		WHERE id = ?
	`

	if err := d.execJobUpdate(ctx, jobID, query, key, time.Now(), jobID); err != nil {
		return fmt.Errorf("failed to update job provenance key: %w", err)
	}

	return nil
}

// CompleteJob marks a job as completed.
func (d *Database) CompleteJob(ctx context.Context, jobID string) error {
	query := `
		UPDATE jobs
		SET state = ?, completed_at = ?, updated_at = ?
		WHERE id = ?
		  AND state NOT IN (?, ?)
	`

	now := time.Now()

	result, err := d.execContext(
		ctx,
		query,
		models.JobStateDone,
		now,
		now,
		jobID,
		models.JobStateDone,
		models.JobStateFailed,
	)
	if err != nil {
		return fmt.Errorf("failed to complete job: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows != 0 {
		return nil
	}

	state, err := d.getJobState(ctx, jobID)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("job not found: %s", jobID)
	}

	if err != nil {
		return fmt.Errorf("failed to load job state: %w", err)
	}

	if state == models.JobStateDone || state == models.JobStateFailed {
		return nil
	}

	return fmt.Errorf("job %s not eligible for completion (state=%s)", jobID, state)
}

// FailJob marks a job as failed with an error message.
func (d *Database) FailJob(ctx context.Context, jobID, stage, errorMsg, errorDetails string) error {
	query := `
		UPDATE jobs
		SET state = ?, error = ?, error_details = ?, last_stage = ?, completed_at = ?, updated_at = ?
		WHERE id = ?
		  AND state NOT IN (?, ?)
	`

	now := time.Now()

	result, err := d.execContext(
		ctx,
		query,
		models.JobStateFailed,
		errorMsg,
		errorDetails,
		stage,
		now,
		now,
		jobID,
		models.JobStateFailed,
		models.JobStateDone,
	)
	if err != nil {
		return fmt.Errorf("failed to fail job: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows != 0 {
		return nil
	}

	state, err := d.getJobState(ctx, jobID)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("job not found: %s", jobID)
	}

	if err != nil {
		return fmt.Errorf("failed to load job state: %w", err)
	}

	if state == models.JobStateFailed || state == models.JobStateDone {
		return nil
	}

	return fmt.Errorf("job %s not eligible for failure (state=%s)", jobID, state)
}

// UpdateJobProgress persists page-by-page progress for in-flight jobs.
func (d *Database) UpdateJobProgress(ctx context.Context, jobID string, currentPage, totalPages int) error {
	query := `
		UPDATE jobs
		SET current_page = CASE WHEN ? > current_page THEN ? ELSE current_page END,
		    total_pages = CASE WHEN ? > total_pages THEN ? ELSE total_pages END,
		    updated_at = ?
		WHERE id = ?
		  AND state NOT IN (?, ?)
	`

	now := time.Now()

	if err := d.execJobUpdate(
		ctx,
		jobID,
		query,
		currentPage,
		currentPage,
		totalPages,
		totalPages,
		now,
		jobID,
		models.JobStateDone,
		models.JobStateFailed,
	); err != nil {
		return fmt.Errorf("failed to update job progress: %w", err)
	}

	return nil
}

// UpdateJobExtractionArtifacts stores extraction artifacts produced before scanning.
func (d *Database) UpdateJobExtractionArtifacts(
	ctx context.Context,
	jobID, stageLogKey, recipeKey string,
) error {
	query := `
		UPDATE jobs
		SET extraction_stage_log_key = CASE
		        WHEN ? <> '' THEN ?
		        ELSE extraction_stage_log_key
		    END,
		    extraction_recipe_key = CASE
		        WHEN ? <> '' THEN ?
		        ELSE extraction_recipe_key
		    END,
		    updated_at = ?
		WHERE id = ?
		  AND state NOT IN (?, ?)
	`

	if err := d.execJobUpdate(
		ctx,
		jobID,
		query,
		stageLogKey,
		stageLogKey,
		recipeKey,
		recipeKey,
		time.Now(),
		jobID,
		models.JobStateDone,
		models.JobStateFailed,
	); err != nil {
		return fmt.Errorf("failed to update extraction artifacts: %w", err)
	}

	return nil
}

// UpdateJobCompletionArtifacts stores primary completion artifacts and summary totals.
func (d *Database) UpdateJobCompletionArtifacts(
	ctx context.Context,
	jobID, reportJSONKey, reportKey, stageLogKey, recipeKey string,
	totalViolations int,
) error {
	query := `
		UPDATE jobs
		SET report_json_key = CASE
		        WHEN ? <> '' THEN ?
		        ELSE report_json_key
		    END,
		    report_key = CASE
		        WHEN ? <> '' THEN ?
		        ELSE report_key
		    END,
		    scan_stage_log_key = CASE
		        WHEN ? <> '' THEN ?
		        ELSE scan_stage_log_key
		    END,
		    scan_recipe_key = CASE
		        WHEN ? <> '' THEN ?
		        ELSE scan_recipe_key
		    END,
		    total_violations = CASE
		        WHEN ? > total_violations THEN ?
		        ELSE total_violations
		    END,
		    updated_at = ?
		WHERE id = ?
		  AND state NOT IN (?, ?)
	`

	if err := d.execJobUpdate(
		ctx,
		jobID,
		query,
		reportJSONKey,
		reportJSONKey,
		reportKey,
		reportKey,
		stageLogKey,
		stageLogKey,
		recipeKey,
		recipeKey,
		totalViolations,
		totalViolations,
		time.Now(),
		jobID,
		models.JobStateDone,
		models.JobStateFailed,
	); err != nil {
		return fmt.Errorf("failed to update completion artifacts: %w", err)
	}

	return nil
}
