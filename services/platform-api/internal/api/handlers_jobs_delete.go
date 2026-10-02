package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/mattboback/stageflow/internal/artifacts"
	"github.com/mattboback/stageflow/internal/jobstore"
	"github.com/mattboback/stageflow/libs/go/httputil"
	"github.com/mattboback/stageflow/libs/go/logging"
)

func (s *Server) handleJobDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		httputil.RespondError(w, http.StatusMethodNotAllowed, "Method not allowed")

		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/jobs/")

	parts := strings.Split(path, "/")
	if len(parts) != 1 || parts[0] == "" {
		httputil.RespondNotFound(w, "Endpoint")

		return
	}

	jobID := parts[0]
	if !validJobID(jobID) {
		httputil.RespondError(w, http.StatusBadRequest, "Invalid job ID")

		return
	}

	ctx := logging.WithJobID(r.Context(), jobID)

	rec, err := s.statusReader.GetJob(ctx, jobID)
	if err != nil {
		if errors.Is(err, jobstore.ErrJobNotFound) {
			if cleanupErr := s.deleteJobObjects(ctx, jobID); cleanupErr != nil {
				httputil.RespondError(w, http.StatusInternalServerError, "Failed to delete scan artifacts")
				return
			}

			w.WriteHeader(http.StatusNoContent)

			return
		}

		logging.Error(ctx, "Failed to fetch job status for delete", "error", err)
		httputil.RespondStructuredError(w, http.StatusInternalServerError, httputil.NewDatabaseError())

		return
	}

	if !rec.State.IsTerminal() {
		httputil.RespondError(w, http.StatusConflict, "Job is still running")

		return
	}

	if s.config.JobDeleter == nil {
		httputil.RespondError(w, http.StatusInternalServerError, "Job deletion is unavailable")
		return
	}

	if deletionErr := s.config.JobDeleter.DeleteJob(ctx, jobID); deletionErr != nil {
		if errors.Is(deletionErr, jobstore.ErrCleanupPending) || errors.Is(deletionErr, jobstore.ErrJobBusy) {
			httputil.RespondError(w, http.StatusConflict, "Job cleanup is still running")
		} else {
			httputil.RespondError(w, http.StatusInternalServerError, "Failed to delete job record")
		}

		return
	}

	cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if cleanupErr := s.deleteJobObjects(cleanupCtx, jobID); cleanupErr != nil {
		logging.Error(ctx, "Failed to delete scan artifacts", "error", cleanupErr)
		httputil.RespondError(w, http.StatusInternalServerError, "Failed to delete scan artifacts")

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) deleteJobObjects(ctx context.Context, jobID string) error {
	if s.config == nil || s.config.Storage == nil {
		return nil
	}

	prefix := jobID + "/"
	if err := s.config.Storage.DeletePrefix(ctx, artifacts.BucketStaging, prefix); err != nil {
		return err
	}

	return s.config.Storage.DeletePrefix(ctx, artifacts.BucketArtifacts, prefix)
}
