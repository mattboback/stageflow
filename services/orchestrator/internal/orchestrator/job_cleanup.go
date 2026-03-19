package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mattboback/stageflow/libs/go/models"
	"github.com/mattboback/stageflow/libs/go/storage"
)

// failJob marks the job as failed and cleans up resources.
func (o *Orchestrator) failJob(ctx context.Context, jobID, stage, errorMsg, errorDetails string) error {
	return o.newService().FailJob(ctx, jobID, stage, errorMsg, errorDetails)
}

func (o *Orchestrator) cleanupPod(ctx context.Context, podID string) error {
	if err := o.podmanClient.StopPod(ctx, podID); err != nil {
		slog.Warn("Failed to stop pod", "pod_id", podID, "error", err)
	}

	if err := o.podmanClient.RemovePod(ctx, podID, true); err != nil {
		return fmt.Errorf("failed to remove pod: %w", err)
	}

	slog.Debug("Cleaned up pod", "pod_id", podID)

	return nil
}

func (o *Orchestrator) cleanupVolumes(ctx context.Context, jobID string) {
	volumes := []string{
		"workspace-" + jobID,
		"results-" + jobID,
	}

	for _, vol := range volumes {
		if err := o.podmanClient.RemoveVolume(ctx, vol, true); err != nil {
			slog.Warn("Failed to remove volume", "volume", vol, "error", err)
		}
	}
}

func (o *Orchestrator) cleanupStaging(ctx context.Context, job *models.Job) {
	if job == nil || job.InputType != "zip" || job.InputPath == "" {
		return
	}

	if o.stagingStorage == nil {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := o.stagingStorage.DeleteFile(ctx, storage.BucketStaging, job.InputPath); err != nil {
		slog.Warn("Failed to delete staging object", "path", job.InputPath, "error", err)
	}
}

func (o *Orchestrator) failJobSafe(ctx context.Context, jobID, stage, message string) {
	o.failJobSafeWithDetails(ctx, jobID, stage, message, "")
}

func (o *Orchestrator) failJobSafeWithDetails(ctx context.Context, jobID, stage, message, details string) {
	if err := o.failJob(ctx, jobID, stage, message, details); err != nil {
		slog.Warn("Failed to mark job as failed", "job_id", jobID, "error", err)
	}
}
