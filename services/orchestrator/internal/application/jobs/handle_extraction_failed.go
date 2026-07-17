package jobs

import (
	"context"
	"log/slog"

	"github.com/mattboback/stageflow/libs/go/events"
)

func (s *Service) HandleExtractionFailed(
	ctx context.Context,
	payload *events.ExtractionFailedPayload,
) error {
	if err := validateInboundPayload(events.EventExtractionFailed, payload); err != nil {
		return err
	}

	slog.Error("Handling extraction.failed", "job_id", payload.JobID)

	if payload.StageLogPath != "" || payload.RecipePath != "" {
		slog.Debug(
			"Extraction failure artifacts",
			"job_id",
			payload.JobID,
			"stage_log",
			payload.StageLogPath,
			"recipe",
			payload.RecipePath,
		)
	}

	if _, err := s.store.GetJob(ctx, payload.JobID); err != nil {
		if shouldIgnoreMissingJob(events.EventExtractionFailed, payload.JobID, err) {
			return nil
		}

		return err
	}

	if err := s.store.UpdateJobExtractionArtifacts(
		ctx,
		payload.JobID,
		payload.StageLogPath,
		payload.RecipePath,
	); err != nil {
		slog.Warn(
			"Failed to persist extraction failure artifacts",
			"job_id",
			payload.JobID,
			"error",
			err,
		)
	}

	return s.FailJob(ctx, payload.JobID, "extraction", payload.Error, payload.ErrorDetails)
}
