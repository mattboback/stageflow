package jobs

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/models"
)

func (s *Service) HandleScanPageCompleted(
	ctx context.Context,
	payload *events.ScanPageCompletedPayload,
) error {
	if err := validateInboundPayload(events.EventScanPageCompleted, payload); err != nil {
		return err
	}

	job, err := s.store.GetJob(ctx, payload.JobID)
	if err != nil {
		if shouldIgnoreMissingJob(events.EventScanPageCompleted, payload.JobID, err) {
			return nil
		}

		return fmt.Errorf("failed to get job: %w", err)
	}

	if job.State == models.JobStateDone || job.State == models.JobStateFailed {
		slog.Debug("Ignoring scan.page.completed for terminal job", "job_id", payload.JobID, "state", job.State)

		return nil
	}

	if updateErr := s.store.UpdateJobProgress(
		ctx,
		payload.JobID,
		payload.PageIndex,
		payload.TotalPages,
	); updateErr != nil {
		return fmt.Errorf("failed to persist scan progress: %w", updateErr)
	}

	return nil
}
