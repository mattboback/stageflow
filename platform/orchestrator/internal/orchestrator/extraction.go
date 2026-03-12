package orchestrator

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mattboback/stageflow/packages/shared-go/models"
)

func (o *Orchestrator) createJobPod(ctx context.Context, job *models.Job) (string, error) {
	podID, err := o.runtimeAdapter().CreateJobPod(ctx, job)
	if err != nil {
		return "", err
	}

	slog.Info("Created pod for job", "pod_id", podID, "job_id", job.ID)
	o.recordInternalEvent(ctx, job.ID, "orchestrator.pod.created", map[string]any{
		"pod_id": podID,
	})

	return podID, nil
}

// startExtractionWorkerWithTimeout starts the extraction worker and enforces a hard startup budget.
func (o *Orchestrator) startExtractionWorkerWithTimeout(ctx context.Context, job *models.Job, podID string) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, o.extractionTimeout)
	defer cancel()

	resultChan := make(chan error, 1)

	go func() {
		resultChan <- o.startExtractionWorker(timeoutCtx, job, podID)
	}()

	select {
	case err := <-resultChan:
		return err
	case <-timeoutCtx.Done():
		return fmt.Errorf("extraction worker start timed out after %v", o.extractionTimeout)
	}
}

func (o *Orchestrator) startExtractionWorker(ctx context.Context, job *models.Job, podID string) error {
	result, err := o.runtimeAdapter().StartExtractionWorker(ctx, job, podID)
	if result != nil && result.ContainerID != "" {
		slog.Info("Created extraction worker container", "container_id", result.ContainerID, "job_id", job.ID)
		o.recordInternalEvent(ctx, job.ID, "orchestrator.container.created", map[string]any{
			"component":    "extraction-worker",
			"container_id": result.ContainerID,
		})
	}

	if err != nil {
		return err
	}

	slog.Info("Started extraction worker container", "container_id", result.ContainerID, "job_id", job.ID)
	o.recordInternalEvent(ctx, job.ID, "orchestrator.container.started", map[string]any{
		"component":    "extraction-worker",
		"container_id": result.ContainerID,
	})

	o.spawnMonitorContainer(backgroundWithCorrelation(ctx), result.ContainerID, job.ID, "extraction")

	return nil
}
