package db

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

const defaultJobEventsPruneBatchSize = 1000

// JobEventsPrunerConfig configures periodic job_events pruning.
type JobEventsPrunerConfig struct {
	Retention time.Duration
	Interval  time.Duration
	BatchSize int
	Logger    *slog.Logger
}

// PruneJobEventsBefore deletes job_events older than cutoff in batches.
// It returns the total number of deleted rows.
func (d *Database) PruneJobEventsBefore(ctx context.Context, cutoff time.Time, batchSize int) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	if batchSize <= 0 {
		batchSize = defaultJobEventsPruneBatchSize
	}

	const query = `
		DELETE FROM job_events
		WHERE id IN (
			SELECT id
			FROM job_events
			WHERE timestamp < ?
			ORDER BY timestamp ASC
			LIMIT ?
		)
	`

	var totalDeleted int64

	for {
		result, err := d.db.ExecContext(ctx, query, cutoff.UTC(), batchSize)
		if err != nil {
			return totalDeleted, fmt.Errorf("failed pruning job events: %w", err)
		}

		deleted, err := result.RowsAffected()
		if err != nil {
			return totalDeleted, fmt.Errorf("failed reading pruned row count: %w", err)
		}

		totalDeleted += deleted
		if deleted < int64(batchSize) {
			return totalDeleted, nil
		}

		if ctxErr := ctx.Err(); ctxErr != nil {
			return totalDeleted, ctxErr
		}
	}
}

// StartJobEventsPruner starts a background pruning loop.
func (d *Database) StartJobEventsPruner(ctx context.Context, cfg JobEventsPrunerConfig) error {
	if cfg.Retention <= 0 {
		return errors.New("job events pruner retention must be > 0")
	}

	if cfg.Interval <= 0 {
		return errors.New("job events pruner interval must be > 0")
	}

	batchSize := cfg.BatchSize
	if batchSize <= 0 {
		batchSize = defaultJobEventsPruneBatchSize
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	pruneOnce := func() {
		cutoff := time.Now().Add(-cfg.Retention)

		deleted, err := d.PruneJobEventsBefore(ctx, cutoff, batchSize)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return
			}

			logger.Warn("Failed to prune job events", "error", err, "cutoff", cutoff, "batch_size", batchSize)

			return
		}

		if deleted > 0 {
			logger.Info("Pruned job events", "deleted", deleted, "cutoff", cutoff, "batch_size", batchSize)
		}
	}

	go func() {
		ticker := time.NewTicker(cfg.Interval)
		defer ticker.Stop()

		pruneOnce()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				pruneOnce()
			}
		}
	}()

	return nil
}
