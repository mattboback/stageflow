package api

import (
	"errors"
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	"github.com/mattboback/stageflow/libs/go/httputil"
	"github.com/mattboback/stageflow/libs/go/logging"
	"github.com/mattboback/stageflow/libs/go/models"
	"github.com/mattboback/stageflow/libs/go/storage"
	"github.com/mattboback/stageflow/services/platform-api/internal/status"
)

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

	if len(parts) > 1 {
		switch parts[1] {
		case "report":
			s.handleJobReport(w, r, jobID)

			return
		case "results":
			s.handleJobResults(w, r, jobID)

			return
		}
	}

	ctx := logging.WithJobID(r.Context(), jobID)

	rec, err := s.loadJobRecord(ctx, jobID)
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

	rec, err := s.loadJobRecord(ctx, jobID)
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
		// Many scanners do not currently upload a standalone HTML report artifact.
		// Instead of failing the endpoint, return a minimal HTML "report" that
		// links to the aggregated JSON report and (when available) the SPA report view.
		if rec.ReportJSONKey == "" {
			httputil.RespondStructuredError(w, http.StatusNotFound, httputil.ErrorDetail{
				Code:       httputil.ErrCodeNotFound,
				Message:    "Report not found",
				Details:    "No report has been generated for this job",
				Suggestion: "This job may not have generated a report, or the report generation failed.",
			})

			return
		}

		escapedJobID := html.EscapeString(jobID)
		resultsPath := "/api/v1/jobs/" + escapedJobID + "/results"
		spaPath := "/scan/" + escapedJobID + "/report"
		statusPath := "/api/v1/jobs/" + escapedJobID

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <link rel="icon" href="/favicon.svg" type="image/svg+xml" />
    <title>StageFlow Report %s</title>
    <style>
      :root {
        color-scheme: light;
        --ink: #0f172a;
        --muted: #475569;
        --accent: #0f766e;
        --accent-soft: #ccfbf1;
        --border: #e2e8f0;
        --card: #ffffff;
        --bg: #f8fafc;
        --bg-2: #eef2ff;
      }
      * {
        box-sizing: border-box;
      }
      body {
        margin: 0;
        font-family: "Space Grotesk", "Manrope", "Segoe UI", "Trebuchet MS", sans-serif;
        color: var(--ink);
        background:
          radial-gradient(circle at 10%% 10%%, var(--bg-2), transparent 45%%),
          radial-gradient(circle at 85%% 0%%, #ecfeff, transparent 40%%),
          var(--bg);
      }
      .page {
        min-height: 100vh;
        padding: 32px 20px;
        display: flex;
        align-items: center;
        justify-content: center;
      }
      .card {
        width: min(860px, 100%%);
        background: var(--card);
        border: 1px solid var(--border);
        border-radius: 20px;
        box-shadow: 0 12px 30px rgba(15, 23, 42, 0.08);
        padding: 32px;
      }
      h1 {
        margin: 0 0 12px;
        font-size: 28px;
        letter-spacing: -0.02em;
      }
      p {
        margin: 0 0 14px;
        color: var(--muted);
      }
      .badge {
        display: inline-flex;
        align-items: center;
        gap: 8px;
        background: var(--accent-soft);
        color: #134e4a;
        padding: 6px 10px;
        border-radius: 999px;
        font-weight: 600;
        font-size: 12px;
      }
      .meta {
        margin-top: 16px;
        font-family: "JetBrains Mono", "SFMono-Regular", Menlo, Consolas, monospace;
        font-size: 13px;
        color: #334155;
        background: #f1f5f9;
        padding: 10px 12px;
        border-radius: 12px;
      }
      .actions {
        margin-top: 20px;
        display: flex;
        flex-wrap: wrap;
        gap: 12px;
      }
      a.button {
        display: inline-flex;
        align-items: center;
        gap: 8px;
        padding: 10px 14px;
        border-radius: 12px;
        border: 1px solid var(--border);
        text-decoration: none;
        color: var(--ink);
        background: #f8fafc;
        font-weight: 600;
      }
      a.button.primary {
        background: var(--accent);
        color: #ffffff;
        border-color: var(--accent);
      }
      .note {
        margin-top: 18px;
        padding: 14px;
        background: #f8fafc;
        border: 1px dashed var(--border);
        border-radius: 12px;
        color: var(--muted);
        font-size: 14px;
      }
      @media (max-width: 600px) {
        .card {
          padding: 24px;
        }
        h1 {
          font-size: 24px;
        }
      }
    </style>
  </head>
  <body>
    <main class="page">
      <section class="card" aria-labelledby="report-title">
        <span class="badge">Scanner report placeholder</span>
        <h1 id="report-title">StageFlow report</h1>
        <p>This job does not include a standalone HTML report from a scanner.</p>
        <p>Use the aggregated JSON or the interactive UI to explore issues and artifacts.</p>
        <div class="meta">Job ID: %s</div>
        <div class="actions">
          <a class="button primary" href="%s">Open interactive report UI</a>
          <a class="button" href="%s">Download aggregated report JSON</a>
          <a class="button" href="%s">View job status</a>
        </div>
        <div class="note">
          Tip: The interactive UI link works when the frontend is served on the same origin as this API. Some scanners only emit JSON artifacts; configure a scanner that produces HTML output if you need a standalone report file.
        </div>
      </section>
    </main>
  </body>
</html>`, escapedJobID, escapedJobID, spaPath, resultsPath, statusPath)

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

	rec, err := s.loadJobRecord(ctx, jobID)
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
