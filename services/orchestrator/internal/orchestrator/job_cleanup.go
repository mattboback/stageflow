package orchestrator

import (
	"context"
	"fmt"
	"log/slog"

	podman "github.com/mattboback/stageflow/services/orchestrator/internal/adapters/runtime"
)

// failJob marks the job as failed and cleans up resources.
func (o *Orchestrator) failJob(ctx context.Context, jobID, stage, errorMsg, errorDetails string) error {
	return o.newService().FailJob(ctx, jobID, stage, errorMsg, errorDetails)
}

func (o *Orchestrator) cleanupPod(ctx context.Context, podID string) error {
	if err := o.podmanClient.StopPod(ctx, podID); err != nil {
		slog.Warn("Failed to stop pod", "pod_id", podID, "error", err)
	}

	// Cleanup runs on paths that cannot know whether a pod was ever created, so an
	// absent pod is the desired end state rather than a failure. Treating it as an
	// error here would abort before private input cleanup.
	if err := o.podmanClient.RemovePod(ctx, podID, true); err != nil && !podman.IsNotFound(err) {
		return fmt.Errorf("failed to remove pod: %w", err)
	}

	slog.Debug("Cleaned up pod", "pod_id", podID)

	return nil
}

func (o *Orchestrator) failJobSafe(ctx context.Context, jobID, stage, message string) {
	o.failJobSafeWithDetails(ctx, jobID, stage, message, "")
}

func (o *Orchestrator) failJobSafeWithDetails(ctx context.Context, jobID, stage, message, details string) {
	if err := o.failJob(ctx, jobID, stage, message, details); err != nil {
		slog.Warn("Failed to mark job as failed", "job_id", jobID, "error", err)
	}
}
