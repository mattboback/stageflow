package jobs

import (
	"context"
	"log/slog"

	"github.com/mattboback/stageflow/packages/shared-go/events"
)

func (s *Service) HandleJobCreated(ctx context.Context, payload *events.JobCreatedPayload) error {
	slog.Info("Handling job.created", "job_id", payload.JobID)
	return s.CreateJob(ctx, payload)
}
