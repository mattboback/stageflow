package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/mattboback/stageflow/packages/shared-go/events"
	"github.com/mattboback/stageflow/packages/shared-go/models"
	"github.com/mattboback/stageflow/packages/shared-go/storage"
)

func normalizeJobFailStage(stage string) string {
	switch stage {
	case events.JobFailStageExtraction, events.JobFailStageScanning, events.JobFailStageReporting:
		return stage
	case "":
		return events.JobFailStageScanning
	case "completing":
		// "Completing" is orchestrator-specific; expose it as "reporting" externally.
		return events.JobFailStageReporting
	case "setup":
		// Setup errors are currently pre-scan orchestration failures.
		return events.JobFailStageScanning
	}

	if strings.HasPrefix(stage, "scanner-") {
		return events.JobFailStageScanning
	}

	// Keep the public contract stable even if we add internal stage names.
	return events.JobFailStageScanning
}

// failJob marks the job as failed and cleans up resources.
func (o *Orchestrator) failJob(ctx context.Context, jobID, stage, errorMsg, errorDetails string) error {
	job, err := o.database.GetJob(ctx, jobID)
	if err != nil {
		return fmt.Errorf("failed to get job for failure handling: %w", err)
	}

	if job.State == models.JobStateDone || job.State == models.JobStateFailed {
		slog.Debug("Job already terminal, skipping fail transition", "job_id", job.ID, "state", job.State)

		return nil
	}

	if failErr := o.database.FailJob(ctx, jobID, errorMsg); failErr != nil {
		return fmt.Errorf("failed to fail job in database: %w", failErr)
	}

	job.State = models.JobStateFailed

	normalizedStage := normalizeJobFailStage(stage)

	o.recordInternalEvent(ctx, jobID, "orchestrator.job.failed", map[string]any{
		"stage":         normalizedStage,
		"stage_raw":     stage,
		"error":         errorMsg,
		"error_details": errorDetails,
	})

	if job.PodID != "" {
		if cleanupErr := o.cleanupPod(ctx, job.PodID); cleanupErr != nil {
			slog.Warn("Failed to cleanup pod", "pod_id", job.PodID, "error", cleanupErr)
		}
	}

	o.cleanupVolumes(ctx, jobID)
	o.cleanupStaging(ctx, job)

	payload := &events.JobFailedPayload{
		JobID:        jobID,
		Status:       "failed",
		Stage:        normalizedStage,
		Error:        errorMsg,
		ErrorDetails: errorDetails,
	}

	if publishErr := o.publisher.PublishJobFailed(ctx, payload); publishErr != nil {
		slog.Warn("Failed to publish job.failed event", "error", publishErr)
		o.recordInternalEvent(ctx, jobID, "orchestrator.event.publish_failed", map[string]any{
			"event": "job.failed",
			"error": publishErr.Error(),
		})
	} else {
		o.recordInternalEvent(ctx, jobID, "orchestrator.event.published", map[string]any{
			"event": "job.failed",
		})
	}

	if errorDetails != "" {
		slog.Error("Job failed", "job_id", jobID, "stage", stage, "error", errorMsg, "details", errorDetails)
	} else {
		slog.Error("Job failed", "job_id", jobID, "stage", stage, "error", errorMsg)
	}

	return nil
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

func stateTransitionDetails(from, to models.JobState) string {
	return fmt.Sprintf("current_state=%s target_state=%s", from, to)
}

func (o *Orchestrator) failJobSafeWithDetails(ctx context.Context, jobID, stage, message, details string) {
	if err := o.failJob(ctx, jobID, stage, message, details); err != nil {
		slog.Warn("Failed to mark job as failed", "job_id", jobID, "error", err)
	}
}
