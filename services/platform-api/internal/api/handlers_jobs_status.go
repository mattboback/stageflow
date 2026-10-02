package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	storage "github.com/mattboback/stageflow/internal/artifacts"
	status "github.com/mattboback/stageflow/internal/jobstore"
	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
	"github.com/mattboback/stageflow/libs/go/httputil"
	"github.com/mattboback/stageflow/libs/go/logging"
	"github.com/mattboback/stageflow/libs/go/models"
)

const maxReportJSONBytes = 64 * 1024 * 1024

// jobIDPattern bounds a job-ID path parameter to a safe charset and length.
// Job IDs are minted as UUIDs, but validating by charset rather than strict
// UUID format stays robust if that ever changes while still rejecting the
// inputs that matter: path separators (traversal), SQL/quote metacharacters,
// control characters (log injection), and unbounded-length strings.
var jobIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// validJobID reports whether id is a safe job-ID path parameter. Rejecting
// malformed IDs up front avoids pointless store/cache lookups and keeps
// attacker-controlled strings out of logs.
func validJobID(id string) bool {
	return jobIDPattern.MatchString(id)
}

func (s *Server) handleJobStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.RespondError(w, http.StatusMethodNotAllowed, "Method not allowed")

		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/jobs/")

	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		httputil.RespondError(w, http.StatusBadRequest, "Missing job ID")

		return
	}

	jobID := parts[0]

	if !validJobID(jobID) {
		httputil.RespondError(w, http.StatusBadRequest, "Invalid job ID")

		return
	}

	if s.respondIfJobDeleted(w, r, jobID) {
		return
	}

	if len(parts) > 1 {
		if len(parts) >= 3 && parts[1] == "files" {
			s.handleJobFile(w, r, jobID, strings.Join(parts[2:], "/"))
			return
		}

		if len(parts) == 2 && parts[1] == "results" {
			s.handleJobResults(w, r, jobID)

			return
		}

		httputil.RespondNotFound(w, "Endpoint")

		return
	}

	ctx := logging.WithJobID(r.Context(), jobID)

	rec, err := s.statusReader.GetJob(ctx, jobID)
	if err != nil {
		if errors.Is(err, status.ErrJobNotFound) {
			httputil.RespondStructuredError(w, http.StatusNotFound, httputil.NewJobNotFoundError(jobID))

			return
		}

		logging.Error(ctx, "Failed to fetch job status", "error", err)
		httputil.RespondStructuredError(w, http.StatusInternalServerError, httputil.NewDatabaseError())

		return
	}

	job, err := s.buildJobStatusResponse(ctx, rec)
	if err != nil {
		logging.Error(ctx, "Failed to build job status", "error", err)
		httputil.RespondError(w, http.StatusInternalServerError, "Failed to fetch job status")

		return
	}

	httputil.RespondOK(w, job)
}

func (s *Server) handleJobResults(w http.ResponseWriter, r *http.Request, jobID string) {
	ctx := logging.WithJobID(r.Context(), jobID)

	rec, err := s.statusReader.GetJob(ctx, jobID)
	if err != nil {
		if errors.Is(err, status.ErrJobNotFound) {
			httputil.RespondNotFound(w, "Job")

			return
		}

		logging.Error(ctx, "Failed to fetch job status", "error", err)
		httputil.RespondError(w, http.StatusInternalServerError, "Failed to fetch job status")

		return
	}

	if rec.State != models.JobStateDone {
		httputil.RespondError(w, http.StatusBadRequest, "Job not completed")

		return
	}

	if rec.ReportJSONKey == "" {
		httputil.RespondNotFound(w, "Report JSON")

		return
	}

	reportJSONKey, ok := jobScopedKey(jobID, rec.ReportJSONKey)
	if !ok {
		logging.Warn(ctx, "Refusing to presign non-job-scoped report JSON key", "key", rec.ReportJSONKey)
		httputil.RespondNotFound(w, "Report JSON")

		return
	}

	s.handleJobFile(w, r, jobID, strings.TrimPrefix(reportJSONKey, jobID+"/"))
}

func (s *Server) downloadReport(ctx context.Context, jobID, reportJSONKey string) (report.UnifiedReportV2, error) {
	if reportJSONKey == "" {
		return report.UnifiedReportV2{}, fmt.Errorf("no report JSON key for job %s", jobID)
	}

	key, ok := jobScopedKey(jobID, reportJSONKey)
	if !ok {
		return report.UnifiedReportV2{}, fmt.Errorf("non-job-scoped report key for %s", jobID)
	}

	reader, err := s.config.Storage.DownloadFile(ctx, storage.BucketArtifacts, key)
	if err != nil {
		return report.UnifiedReportV2{}, fmt.Errorf("download report for %s: %w", jobID, err)
	}
	defer reader.Close()

	data, err := readLimitedReportJSON(reader, maxReportJSONBytes)
	if err != nil {
		return report.UnifiedReportV2{}, fmt.Errorf("read report for %s: %w", jobID, err)
	}

	rpt, err := decodeReport(data, jobID)
	if err != nil {
		return report.UnifiedReportV2{}, fmt.Errorf("parse report for %s: %w", jobID, err)
	}

	return rpt, nil
}

func readLimitedReportJSON(reader io.Reader, limit int64) ([]byte, error) {
	if limit < 1 {
		return nil, errors.New("report size limit must be positive")
	}

	data, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}

	if int64(len(data)) > limit {
		return nil, fmt.Errorf("report JSON exceeds %d byte limit", limit)
	}

	return data, nil
}
