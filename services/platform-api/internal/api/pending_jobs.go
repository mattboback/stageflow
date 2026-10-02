package api

import (
	"context"
	"errors"
	"time"

	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/models"
)

func (s *Server) publishDurableJob(ctx context.Context, envelope *events.Envelope) error {
	if s.config.PendingJobs == nil {
		return errors.New("durable job intake is unavailable")
	}

	payload, ok := envelope.Payload.(*events.JobCreatedPayload)
	if !ok {
		return errors.New("invalid job.created payload")
	}

	now := time.Now().UTC()
	job := &models.Job{
		ID: payload.JobID, State: models.JobStatePending, InputType: payload.InputType,
		InputPath: payload.InputPath, URLs: payload.URLs, Config: payload.Config,
		ExpectedScanners: append([]string(nil), payload.Config.Modules...), CreatedAt: now, UpdatedAt: now,
	}

	return s.config.PendingJobs.PublishPendingJob(ctx, job, func(publishCtx context.Context) error {
		return s.config.Publisher.PublishJobCreated(publishCtx, envelope)
	})
}
