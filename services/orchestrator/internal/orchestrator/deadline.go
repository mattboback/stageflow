package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/logging"
	"github.com/mattboback/stageflow/libs/go/models"
)

func (o *Orchestrator) startDeadlineSweeper(ctx context.Context) {
	ticker := time.NewTicker(o.deadlinePollInterval)
	defer ticker.Stop()

	for {
		if err := o.runDeadlineSweep(ctx); err != nil {
			slog.Warn("Deadline sweep failed", "error", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (o *Orchestrator) runDeadlineSweep(ctx context.Context) error {
	if err := o.failOverdueJobs(
		ctx,
		models.JobStateExtracting,
		o.extractionTimeout,
		events.JobFailStageExtraction,
	); err != nil {
		return err
	}

	if err := o.failOverdueJobs(
		ctx,
		models.JobStateScanning,
		o.scanTimeout,
		events.JobFailStageScanning,
	); err != nil {
		return err
	}

	return nil
}

func (o *Orchestrator) failOverdueJobs(
	ctx context.Context,
	state models.JobState,
	timeout time.Duration,
	stage string,
) error {
	if timeout <= 0 {
		return nil
	}

	jobs, err := o.database.ListJobsByState(ctx, state)
	if err != nil {
		return fmt.Errorf("list jobs by state %s: %w", state, err)
	}

	now := time.Now()

	for _, job := range jobs {
		if job == nil {
			continue
		}

		lastUpdate := job.UpdatedAt
		if lastUpdate.IsZero() {
			lastUpdate = job.CreatedAt
		}

		if now.Sub(lastUpdate) < timeout {
			continue
		}

		jobCtx := logging.WithJobID(o.backgroundWithCorrelation(ctx), job.ID)
		message := fmt.Sprintf("%s timed out after %v", stage, timeout)

		slog.Warn(
			"Job exceeded timeout",
			"job_id",
			job.ID,
			"state",
			state,
			"stage",
			stage,
			"timeout",
			timeout,
			"last_update",
			lastUpdate,
		)

		o.failJobSafe(jobCtx, job.ID, stage, message)
	}

	return nil
}
