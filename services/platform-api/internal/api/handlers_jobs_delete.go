package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/mattboback/stageflow/libs/go/httputil"
	"github.com/mattboback/stageflow/libs/go/logging"
	"github.com/mattboback/stageflow/libs/go/storage"
	"github.com/mattboback/stageflow/services/platform-api/internal/status"
)

func (s *Server) handleJobDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

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

	deleted, err := s.jobDeleted(ctx, jobID)
	if err != nil {
		logging.Error(ctx, "Failed to check job deletion tombstone", "error", err)
		httputil.RespondStructuredError(w, http.StatusInternalServerError, httputil.NewDatabaseError())

		return
	}

	if deleted {
		w.WriteHeader(http.StatusNoContent)

		return
	}

	rec, err := s.jobStatus.Current(ctx, jobID)
	if err != nil {
		if errors.Is(err, status.ErrJobNotFound) {
			httputil.RespondStructuredError(w, http.StatusNotFound, httputil.NewJobNotFoundError(jobID))

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

	if deleteErr := s.deleteJobObjects(ctx, jobID); deleteErr != nil {
		logging.Error(ctx, "Failed to delete job objects", "error", deleteErr)
		httputil.RespondError(w, http.StatusInternalServerError, "Failed to delete scan artifacts")

		return
	}

	if s.projectStore != nil {
		if tombstoneErr := s.projectStore.TombstoneJob(ctx, jobID); tombstoneErr != nil {
			logging.Error(ctx, "Failed to tombstone deleted job", "error", tombstoneErr)
			httputil.RespondStructuredError(w, http.StatusInternalServerError, httputil.NewDatabaseError())

			return
		}

		if unmapErr := s.projectStore.DeleteProjectJob(ctx, jobID); unmapErr != nil {
			logging.Error(ctx, "Failed to detach deleted job from project", "error", unmapErr)
		}
	}

	s.jobStatus.Forget(jobID)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) deleteJobObjects(ctx context.Context, jobID string) error {
	if s.config == nil || s.config.Storage == nil {
		return nil
	}

	prefix := jobID + "/"
	if err := s.config.Storage.DeletePrefix(ctx, storage.BucketStaging, prefix); err != nil {
		return err
	}

	return s.config.Storage.DeletePrefix(ctx, storage.BucketArtifacts, prefix)
}

func (s *Server) jobDeleted(ctx context.Context, jobID string) (bool, error) {
	if s.projectStore == nil {
		return false, nil
	}

	return s.projectStore.JobIsDeleted(ctx, jobID)
}

func (s *Server) respondIfJobDeleted(w http.ResponseWriter, r *http.Request, jobID string) bool {
	deleted, err := s.jobDeleted(r.Context(), jobID)
	if err != nil {
		logging.Error(r.Context(), "Failed to check job deletion tombstone", "error", err)
		httputil.RespondStructuredError(w, http.StatusInternalServerError, httputil.NewDatabaseError())

		return true
	}

	if !deleted {
		return false
	}

	httputil.RespondStructuredError(w, http.StatusNotFound, httputil.NewJobNotFoundError(jobID))

	return true
}
