package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/mattboback/stageflow/packages/shared-go/models"
)

func (d *Database) ensureJobExists(ctx context.Context, jobID string) error {
	var id string

	err := d.db.QueryRowContext(ctx, `SELECT id FROM jobs WHERE id = ?`, jobID).Scan(&id)
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

	err := d.db.QueryRowContext(ctx, `SELECT state FROM jobs WHERE id = ?`, jobID).Scan(&state)

	return state, err
}

func (d *Database) execJobUpdate(ctx context.Context, jobID, query string, args ...any) error {
	result, err := d.db.ExecContext(ctx, query, args...)
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
	query := `
		UPDATE jobs
		SET state = ?, updated_at = ?
		WHERE id = ?
	`

	if err := d.execJobUpdate(ctx, jobID, query, state, time.Now(), jobID); err != nil {
		return fmt.Errorf("failed to update job state: %w", err)
	}

	return nil
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

	result, err := d.db.ExecContext(ctx, query, models.JobStateDone, now, now, jobID, models.JobStateDone, models.JobStateFailed)
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
func (d *Database) FailJob(ctx context.Context, jobID, errorMsg string) error {
	query := `
		UPDATE jobs
		SET state = ?, error = ?, completed_at = ?, updated_at = ?
		WHERE id = ?
		  AND state NOT IN (?, ?)
	`

	now := time.Now()

	result, err := d.db.ExecContext(ctx, query, models.JobStateFailed, errorMsg, now, now, jobID, models.JobStateFailed, models.JobStateDone)
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
