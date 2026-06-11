package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
	"github.com/mattboback/stageflow/libs/go/diff"
	"github.com/mattboback/stageflow/libs/go/httputil"
	"github.com/mattboback/stageflow/libs/go/logging"
	"github.com/mattboback/stageflow/libs/go/models"
	"github.com/mattboback/stageflow/libs/go/storage"
	"github.com/mattboback/stageflow/services/platform-api/internal/project"
	"github.com/mattboback/stageflow/services/platform-api/internal/status"
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
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

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

	if len(parts) > 1 {
		if len(parts) != 2 {
			httputil.RespondNotFound(w, "Endpoint")

			return
		}

		switch parts[1] {
		case "report":
			s.handleJobReport(w, r, jobID)

			return
		case "results":
			s.handleJobResults(w, r, jobID)

			return
		case "diff":
			s.handleJobDiff(w, r, jobID)

			return
		}

		httputil.RespondNotFound(w, "Endpoint")

		return
	}

	ctx := logging.WithJobID(r.Context(), jobID)

	rec, err := s.jobStatus.Current(ctx, jobID)
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

func (s *Server) handleJobReport(w http.ResponseWriter, r *http.Request, jobID string) {
	ctx := logging.WithJobID(r.Context(), jobID)

	rec, err := s.jobStatus.Current(ctx, jobID)
	if err != nil {
		if errors.Is(err, status.ErrJobNotFound) {
			httputil.RespondStructuredError(w, http.StatusNotFound, httputil.NewJobNotFoundError(jobID))

			return
		}

		logging.Error(ctx, "Failed to fetch job status", "error", err)
		httputil.RespondStructuredError(w, http.StatusInternalServerError, httputil.NewDatabaseError())

		return
	}

	if rec.State != models.JobStateDone {
		httputil.RespondStructuredError(w, http.StatusBadRequest, httputil.ErrorDetail{
			Code:       httputil.ErrCodeInvalidRequest,
			Message:    "Job is not completed yet",
			Details:    "Current state: " + string(rec.State),
			Suggestion: "Wait for the job to complete before requesting the report. Check job status at GET /api/v1/jobs/" + jobID,
		})

		return
	}

	if rec.ReportKey == "" {
		details := "No standalone HTML report artifact is available for this job."
		suggestion := "Use GET /api/v1/jobs/" + jobID + "/results or open /scan/" + jobID + "/report."

		if rec.ReportJSONKey == "" {
			details = "No report artifacts have been generated for this job."
			suggestion = "This job may have failed report generation. Check GET /api/v1/jobs/" + jobID + "."
		}

		httputil.RespondStructuredError(w, http.StatusNotFound, httputil.ErrorDetail{
			Code:       httputil.ErrCodeNotFound,
			Message:    "Report not found",
			Details:    details,
			Suggestion: suggestion,
		})

		return
	}

	reportKey, ok := jobScopedKey(jobID, rec.ReportKey)
	if !ok {
		logging.Warn(ctx, "Refusing to presign non-job-scoped report key", "key", rec.ReportKey)
		httputil.RespondNotFound(w, "Report")

		return
	}

	url, err := s.config.Storage.GetPresignedURL(ctx, storage.BucketArtifacts, reportKey, 15*time.Minute)
	if err != nil {
		logging.Error(ctx, "Failed to generate presigned URL", "error", err, "report_key", rec.ReportKey)
		httputil.RespondError(w, http.StatusInternalServerError, "Failed to generate download URL")

		return
	}

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (s *Server) handleJobResults(w http.ResponseWriter, r *http.Request, jobID string) {
	ctx := logging.WithJobID(r.Context(), jobID)

	rec, err := s.jobStatus.Current(ctx, jobID)
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

	url, err := s.config.Storage.GetPresignedURL(ctx, storage.BucketArtifacts, reportJSONKey, 15*time.Minute)
	if err != nil {
		logging.Error(ctx, "Failed to generate presigned URL", "error", err, "report_json_key", rec.ReportJSONKey)
		httputil.RespondError(w, http.StatusInternalServerError, "Failed to generate download URL")

		return
	}

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (s *Server) handleJobDiff(w http.ResponseWriter, r *http.Request, jobID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

		return
	}

	ctx := logging.WithJobID(r.Context(), jobID)

	p, err := s.projectStore.GetProjectForJob(ctx, jobID)
	if err != nil {
		if errors.Is(err, project.ErrNotFound) {
			httputil.RespondStructuredError(w, http.StatusNotFound, httputil.ErrorDetail{
				Code:    httputil.ErrCodeNotFound,
				Message: "Not a project scan",
				Details: "This job is not associated with any project. Diff is only available for project scans.",
			})

			return
		}

		logging.Error(ctx, "Failed to look up project for job", "error", err)
		httputil.RespondError(w, http.StatusInternalServerError, "Failed to look up project")

		return
	}

	if p.BaselineJobID == "" {
		httputil.RespondStructuredError(w, http.StatusNotFound, httputil.ErrorDetail{
			Code:       httputil.ErrCodeNotFound,
			Message:    "No baseline set",
			Details:    fmt.Sprintf("Project %q has no baseline. Promote a scan first.", p.Slug),
			Suggestion: fmt.Sprintf("POST /api/v1/projects/%s/promote with {\"job_id\": \"%s\"}", p.Slug, jobID),
		})

		return
	}

	if p.BaselineJobID == jobID {
		httputil.RespondStructuredError(w, http.StatusBadRequest, httputil.ErrorDetail{
			Code:    httputil.ErrCodeInvalidRequest,
			Message: "Cannot diff against self",
			Details: "This job is the current baseline. Run a new scan to compare.",
		})

		return
	}

	currentRec, err := s.jobStatus.Current(ctx, jobID)
	if err != nil {
		httputil.RespondError(w, http.StatusNotFound, "Current job not found")

		return
	}

	if currentRec.State != models.JobStateDone {
		httputil.RespondError(w, http.StatusBadRequest, "Current job is not completed")

		return
	}

	baselineRec, err := s.jobStatus.Current(ctx, p.BaselineJobID)
	if err != nil {
		httputil.RespondError(w, http.StatusInternalServerError, "Baseline job not found")

		return
	}

	currentReport, err := s.downloadReport(ctx, jobID, currentRec.ReportJSONKey)
	if err != nil {
		logging.Error(ctx, "Failed to download current report", "error", err)
		httputil.RespondError(w, http.StatusInternalServerError, "Failed to fetch current report")

		return
	}

	baselineReport, err := s.downloadReport(ctx, p.BaselineJobID, baselineRec.ReportJSONKey)
	if err != nil {
		logging.Error(ctx, "Failed to download baseline report", "error", err)
		httputil.RespondError(w, http.StatusInternalServerError, "Failed to fetch baseline report")

		return
	}

	result := diff.ComputeDiff(p.BaselineJobID, baselineReport, jobID, currentReport)

	type diffResponse struct {
		diff.Result
		Project projectMeta `json:"project"`
	}

	httputil.RespondOK(w, diffResponse{
		Result: result,
		Project: projectMeta{
			Slug: p.Slug,
			Name: p.Name,
		},
	})
}

type projectMeta struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
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

	var rpt report.UnifiedReportV2

	err = json.Unmarshal(data, &rpt)
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
