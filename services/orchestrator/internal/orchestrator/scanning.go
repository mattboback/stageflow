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
) (string, error) {
	if plan == nil {
		return "", errors.New("scanner launch plan is required")
	}

	scannerType := plan.Labels["scanner_type"]

	result, err := o.jobRuntime.StartScanner(ctx, job, plan)
	if result != nil && result.ContainerID != "" && !result.Existing {
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

	if result != nil && result.ContainerID != "" && result.Existing {
		slog.Info(
			"Adopted scanner container",
			"scanner", scannerType,
			"container_id", result.ContainerID,
			"job_id", job.ID,
		)
		o.recordInternalEvent(ctx, job.ID, "orchestrator.container.adopted", map[string]any{
			"component":    "scanner",
			"scanner_type": scannerType,
			"container_id": result.ContainerID,
		})
	}

	if err != nil {
		return "", err
	}

	if result == nil || result.ContainerID == "" {
		return "", errors.New("scanner runtime returned no container")
	}

	if result.Started {
		slog.Info(
			"Scanner container is running",
			"scanner", scannerType,
			"container_id", result.ContainerID,
			"job_id", job.ID,
		)
		o.recordInternalEvent(ctx, job.ID, "orchestrator.container.started", map[string]any{
			"component":    "scanner",
			"scanner_type": scannerType,
			"container_id": result.ContainerID,
		})
	} else {
		slog.Info(
			"Reattached terminal scanner container without restarting it",
			"scanner", scannerType,
			"container_id", result.ContainerID,
			"job_id", job.ID,
		)
		o.recordInternalEvent(ctx, job.ID, "orchestrator.container.reattached", map[string]any{
			"component":    "scanner",
			"scanner_type": scannerType,
			"container_id": result.ContainerID,
		})
	}

	o.spawnMonitorContainer(o.backgroundWithCorrelation(ctx), result.ContainerID, job.ID, "scanner-"+scannerType)

	return result.ContainerID, nil
}
