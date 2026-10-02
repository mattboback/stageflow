package jobs

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/mattboback/stageflow/libs/go/models"
	provenanceauth "github.com/mattboback/stageflow/libs/go/provenance"
)

func (s *Service) cleanupAuthStorageState(ctx context.Context, job *models.Job) {
	if s == nil || s.authCleaner == nil || job == nil || len(job.Config.Auth) == 0 {
		return
	}

	var auth provenanceauth.Auth
	if err := json.Unmarshal(job.Config.Auth, &auth); err != nil {
		slog.Warn("Failed to parse job auth for cleanup", "job_id", job.ID, "error", err)

		return
	}

	if auth.Mode != provenanceauth.AuthModeStorageState {
		return
	}

	cleanupCtx := ctx
	if cleanupCtx == nil || cleanupCtx.Err() != nil {
		cleanupCtx = context.Background()
	}

	cleanupCtx, cancel := context.WithTimeout(cleanupCtx, 5*time.Second)
	defer cancel()

	if err := s.authCleaner.DeleteAuthStorageState(cleanupCtx, job.ID+"/auth.json"); err != nil {
		slog.Warn(
			"Failed to clean up auth storage_state",
			"job_id",
			job.ID,
			"key",
			job.ID+"/auth.json",
			"error",
			err,
		)
	}
}
