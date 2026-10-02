package db

import (
	"context"
	"fmt"
	"time"

	"github.com/mattboback/stageflow/libs/go/models"
)

// DeleteJob preserves a durable replay guard while the event dispatcher exists.
// The guard and deletion share a transaction, so concurrent replay cannot leave
// a recreated row behind after a successful deletion.
func (d *Database) DeleteJob(ctx context.Context, id string) (err error) {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollbackOnError(tx, &err, "delete job")

	if _, execErr := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1,0))", id); execErr != nil {
		return execErr
	}

	if _, execErr := tx.ExecContext(
		ctx,
		"INSERT INTO deleted_jobs(id) VALUES($1) ON CONFLICT DO NOTHING",
		id,
	); execErr != nil {
		return execErr
	}

	if _, execErr := tx.ExecContext(ctx, "DELETE FROM jobs WHERE id=$1", id); execErr != nil {
		return execErr
	}

	return tx.Commit()
}

func (d *Database) JobExists(ctx context.Context, id string) (bool, error) {
	var exists bool

	err := d.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM jobs WHERE id=$1)", id).Scan(&exists)

	return exists, err
}

func (d *Database) TerminalJobsBefore(ctx context.Context, cutoff time.Time) ([]*models.Job, error) {
	var result []*models.Job

	for _, state := range []models.JobState{models.JobStateDone, models.JobStateFailed} {
		jobs, err := d.ListJobsByState(ctx, state)
		if err != nil {
			return nil, fmt.Errorf("terminal retention: %w", err)
		}

		for _, job := range jobs {
			if job.CompletedAt != nil && job.CompletedAt.Before(cutoff) {
				result = append(result, job)
			}
		}
	}

	return result, nil
}
