package db

import (
	"context"
	"fmt"
	"time"
)

// RecordExtractionStart records when extraction started.
func (d *Database) RecordExtractionStart(ctx context.Context, jobID string) error {
	query := `
		UPDATE jobs
		SET extraction_started_at = ?, updated_at = ?
		WHERE id = ?
	`

	now := time.Now()
	if err := d.execJobUpdate(ctx, jobID, query, now, now, jobID); err != nil {
		return fmt.Errorf("failed to record extraction start: %w", err)
	}

	return nil
}

// RecordExtractionComplete records when extraction completed.
func (d *Database) RecordExtractionComplete(ctx context.Context, jobID string) error {
	query := `
		UPDATE jobs
		SET extraction_completed_at = ?, updated_at = ?
		WHERE id = ?
	`

	now := time.Now()
	if err := d.execJobUpdate(ctx, jobID, query, now, now, jobID); err != nil {
		return fmt.Errorf("failed to record extraction complete: %w", err)
	}

	return nil
}

// RecordScanStart records when scanning started.
func (d *Database) RecordScanStart(ctx context.Context, jobID string) error {
	query := `
		UPDATE jobs
		SET scan_started_at = COALESCE(scan_started_at, ?),
		    updated_at = CASE WHEN scan_started_at IS NULL THEN ? ELSE updated_at END
		WHERE id = ?
	`

	now := time.Now()
	if err := d.execJobUpdate(ctx, jobID, query, now, now, jobID); err != nil {
		return fmt.Errorf("failed to record scan start: %w", err)
	}

	return nil
}

// RecordScanComplete records when scanning completed.
func (d *Database) RecordScanComplete(ctx context.Context, jobID string) error {
	query := `
		UPDATE jobs
		SET scan_completed_at = ?, updated_at = ?
		WHERE id = ?
	`

	now := time.Now()
	if err := d.execJobUpdate(ctx, jobID, query, now, now, jobID); err != nil {
		return fmt.Errorf("failed to record scan complete: %w", err)
	}

	return nil
}

// UpdateJobMetrics updates the issue metrics for a job.
func (d *Database) UpdateJobMetrics(
	ctx context.Context,
	jobID string,
	pagesScanned, total, critical, serious, moderate, minor int,
) error {
	query := `
		UPDATE jobs
		SET pages_scanned = ?, total_issues = ?, critical_issues = ?, serious_issues = ?, moderate_issues = ?, minor_issues = ?, updated_at = ?
		WHERE id = ?
	`

	if err := d.execJobUpdate(
		ctx,
		jobID,
		query,
		pagesScanned,
		total,
		critical,
		serious,
		moderate,
		minor,
		time.Now(),
		jobID,
	); err != nil {
		return fmt.Errorf("failed to update job metrics: %w", err)
	}

	return nil
}
