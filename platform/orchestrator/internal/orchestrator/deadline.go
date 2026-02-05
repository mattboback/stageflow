package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mattboback/stageflow/packages/shared-go/models"
)

// watchDeadline fails a job if it stays in a given state longer than timeout.
// It polls periodically and resets the clock whenever the job's updated_at advances
// or the state moves forward. This acts like a heartbeat without extra messages.
func (o *Orchestrator) watchDeadline(
	ctx context.Context,
	jobID string,
	expectedState models.JobState,
	timeout time.Duration,
	stage string,
) {
	ticker := time.NewTicker(o.deadlinePollInterval)
	defer ticker.Stop()

	var lastUpdate time.Time

	start := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		job, err := o.database.GetJob(ctx, jobID)
		if err != nil {
			slog.Warn("watchDeadline: failed to load job", "job_id", jobID, "error", err)

			continue
		}

		if job.State == models.JobStateDone || job.State == models.JobStateFailed {
			return
		}
		// If state moved forward beyond expected, stop watching.
		if o.stateMachine.CanTransition(expectedState, job.State) {
			return
		}

		if job.UpdatedAt.After(lastUpdate) {
			lastUpdate = job.UpdatedAt
			start = time.Now()

			continue
		}

		if time.Since(start) >= timeout {
			slog.Warn("watchDeadline: job exceeded timeout", "job_id", jobID, "stage", stage, "timeout", timeout)
			o.failJobSafe(
				backgroundWithCorrelation(ctx),
				jobID,
				stage,
				fmt.Sprintf("%s timed out after %v", stage, timeout),
			)

			return
		}
	}
}
