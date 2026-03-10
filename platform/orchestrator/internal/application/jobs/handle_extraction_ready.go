package jobs

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mattboback/stageflow/packages/shared-go/events"
)

func (s *Service) HandleExtractionReady(
	ctx context.Context,
	payload *events.ExtractionReadyPayload,
) error {
	slog.Info("Handling extraction.ready", "job_id", payload.JobID)

	if payload.StageLogPath != "" || payload.RecipePath != "" {
		slog.Debug(
			"Extraction artifacts",
			"job_id",
			payload.JobID,
			"stage_log",
			payload.StageLogPath,
			"recipe",
			payload.RecipePath,
		)
	}

	if err := s.PrepareExtractedJob(ctx, payload); err != nil {
		return fmt.Errorf("prepare extracted job: %w", err)
	}

	return nil
}
