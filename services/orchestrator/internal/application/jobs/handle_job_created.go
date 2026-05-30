package jobs

import (
	"context"
	"log/slog"

	"github.com/mattboback/stageflow/libs/go/events"
)

func (s *Service) HandleJobCreated(ctx context.Context, payload *events.JobCreatedPayload) error {
	if err := validateInboundPayload(events.EventJobCreated, payload); err != nil {
		return err
	}

	slog.Info("Handling job.created", "job_id", payload.JobID)

	return s.CreateJob(ctx, payload)
}
