package orchestrator

import (
	"context"
	"errors"
	"log/slog"

	"github.com/mattboback/stageflow/libs/go/models"
	appjobs "github.com/mattboback/stageflow/services/orchestrator/internal/application/jobs"
)

func (o *Orchestrator) startPlannedScanner(
	ctx context.Context,
	job *models.Job,
	plan *appjobs.ScannerLaunchPlan,
) error {
	if plan == nil {
		return errors.New("scanner launch plan is required")
	}

	scannerType := plan.Labels["scanner_type"]

	result, err := o.jobRuntime.StartScanner(ctx, job, plan)
	if result != nil && result.ContainerID != "" {
		slog.Info(
			"Created scanner container",
			"scanner",
			scannerType,
			"container_id",
			result.ContainerID,
			"job_id",
			job.ID,
		)
		o.recordInternalEvent(ctx, job.ID, "orchestrator.container.created", map[string]any{
			"component":    "scanner",
			"scanner_type": scannerType,
			"container_id": result.ContainerID,
		})
	}

	if err != nil {
		return err
	}

	slog.Info(
		"Started scanner container",
		"scanner",
		scannerType,
		"container_id",
		result.ContainerID,
		"job_id",
		job.ID,
	)
	o.recordInternalEvent(ctx, job.ID, "orchestrator.container.started", map[string]any{
		"component":    "scanner",
		"scanner_type": scannerType,
		"container_id": result.ContainerID,
	})

	o.spawnMonitorContainer(backgroundWithCorrelation(ctx), result.ContainerID, job.ID, "scanner-"+scannerType)

	return nil
}
