package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	storage "github.com/mattboback/stageflow/internal/artifacts"
	jobstore "github.com/mattboback/stageflow/internal/jobstore"
	"github.com/mattboback/stageflow/libs/go/httputil"
	"github.com/mattboback/stageflow/libs/go/logging"
	"github.com/mattboback/stageflow/libs/go/models"
)

func (s *Server) ensureAdmissionDisk(ctx context.Context, w http.ResponseWriter) bool {
	if s.config.CheckDisk == nil {
		s.respondAdmissionError(w, errors.New("disk admission check is unavailable"))
		return false
	}

	if err := s.config.CheckDisk(ctx); err != nil {
		s.respondAdmissionError(w, err)
		return false
	}

	return true
}

func (s *Server) admitJob(ctx context.Context, job *models.Job) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if s.config.Admitter == nil || s.config.CheckDisk == nil {
		return errors.New("job admission is unavailable")
	}

	if err := s.config.CheckDisk(ctx); err != nil {
		return err
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	if err := s.config.Admitter.AdmitJob(ctx, job); err != nil {
		if errors.Is(err, jobstore.ErrQueueFull) {
			return fmt.Errorf("admit job: %w", err)
		}
		// A failed Commit response can arrive after PostgreSQL persisted the row.
		// Only positive durable evidence resolves that ambiguity. Absence can be
		// temporary, so it never authorizes deleting input needed by a late commit.
		if s.admissionRecorded(job.ID) {
			return nil
		}

		return &uncertainAdmissionError{cause: err}
	}

	return nil
}

func (s *Server) respondAdmissionError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, jobstore.ErrQueueFull), errors.Is(err, storage.ErrLowDisk):
		w.Header().Set("Retry-After", "30")
		httputil.RespondError(
			w,
			http.StatusServiceUnavailable,
			"Scan capacity is temporarily unavailable. Retry in 30 seconds.",
		)
	case errors.Is(err, context.DeadlineExceeded):
		httputil.RespondError(w, http.StatusServiceUnavailable, "Request timeout")
	case errors.Is(err, context.Canceled):
		httputil.RespondError(w, http.StatusRequestTimeout, "Request canceled")
	default:
		var clientErr *clientError
		if errors.As(err, &clientErr) {
			httputil.RespondError(w, clientErr.status, clientErr.message)
			return
		}

		httputil.RespondStructuredError(w, http.StatusInternalServerError, httputil.NewDatabaseError())
	}
}

func (s *Server) cleanupRejectedJob(id string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.deleteJobObjects(ctx, id); err != nil {
		logging.Warn(ctx, "Failed to clean rejected job files", "job_id", id, "error", err)
	}
}

type uncertainAdmissionError struct{ cause error }

func (e *uncertainAdmissionError) Error() string {
	return "job admission outcome is uncertain: " + e.cause.Error()
}
func (e *uncertainAdmissionError) Unwrap() error { return e.cause }
func admissionMayBeCommitted(err error) bool {
	var uncertain *uncertainAdmissionError
	return errors.As(err, &uncertain)
}
func (s *Server) admissionRecorded(id string) bool {
	if s.statusReader == nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	record, err := s.statusReader.GetJob(ctx, id)

	return err == nil && record != nil && record.JobID == id
}
