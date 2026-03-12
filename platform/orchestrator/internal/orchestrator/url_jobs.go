package orchestrator

import (
	"context"
	"errors"
	"log/slog"

	"github.com/mattboback/stageflow/packages/shared-go/models"
)

func (o *Orchestrator) handleURLJob(ctx context.Context, job *models.Job) error {
	if job == nil {
		return errors.New("job is nil")
	}

	slog.Info("Setting up URL job", "job_id", job.ID, "url_count", len(job.URLs))

	return o.newService().RunURLJob(ctx, job)
}
