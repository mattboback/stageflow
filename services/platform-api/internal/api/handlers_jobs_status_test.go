// Job status, report, and results retrieval -- redirects, not-found, and
// method handling. Exercises handlers_jobs_status.go.

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/models"
	"github.com/mattboback/stageflow/services/platform-api/internal/jobstatus"
)

func TestHandleJobStatusFound(t *testing.T) {
	server, _, _ := newTestServer(t)

	jobID := "job-status-found"
	applyTestSignal(t, server, jobstatus.Signal{
		Kind: jobstatus.SignalJobCreated,
		JobCreated: &events.JobCreatedPayload{
			JobID:     jobID,
			InputType: models.JobInputTypeURLs,
			URLs:      []string{"https://example.com"},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/"+jobID, http.NoBody)
	rr := httptest.NewRecorder()

	server.handleJobStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var job models.JobStatus
	if err := json.NewDecoder(rr.Body).Decode(&job); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if job.State != models.JobStateScanning {
		t.Fatalf("expected SCANNING, got %s", job.State)
	}
}

func TestHandleJobStatusMethodNotAllowed(t *testing.T) {
	server, _, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/test", http.NoBody)
	rr := httptest.NewRecorder()

	server.handleJobStatus(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

// --- handleJobReport ---

func TestHandleJobReportNotCompleted(t *testing.T) {
	server, _, _ := newTestServer(t)

	jobID := "job-report-pending"
	applyTestSignal(t, server, jobstatus.Signal{
		Kind: jobstatus.SignalJobCreated,
		JobCreated: &events.JobCreatedPayload{
			JobID:     jobID,
			InputType: models.JobInputTypeURLs,
			URLs:      []string{"https://example.com"},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/"+jobID+"/report", http.NoBody)
	rr := httptest.NewRecorder()

	server.handleJobReport(rr, req, jobID)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleJobReportNotFound(t *testing.T) {
	server, _, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/missing/report", http.NoBody)
	rr := httptest.NewRecorder()

	server.handleJobReport(rr, req, "missing")

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestHandleJobReportRedirect(t *testing.T) {
	server, _, _ := newTestServer(t)

	jobID := "job-report-redirect"
	applyTestSignal(t, server, jobstatus.Signal{
		Kind: jobstatus.SignalJobCreated,
		JobCreated: &events.JobCreatedPayload{
			JobID:     jobID,
			InputType: models.JobInputTypeURLs,
			URLs:      []string{"https://example.com"},
		},
	})
	applyTestSignal(t, server, jobstatus.Signal{
		Kind: jobstatus.SignalJobCompleted,
		JobCompleted: &events.JobCompletedPayload{
			JobID: jobID,
			Artifacts: events.ArtifactLocations{
				ReportHTML: jobID + "/report.html",
				ReportJSON: jobID + "/report.json",
			},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/"+jobID+"/report", http.NoBody)
	rr := httptest.NewRecorder()

	server.handleJobReport(rr, req, jobID)

	if rr.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected 307, got %d: %s", rr.Code, rr.Body.String())
	}

	loc := rr.Header().Get("Location")
	if loc == "" {
		t.Fatal("expected Location header for redirect")
	}
}

// --- handleJobResults ---

func TestHandleJobResultsNotCompleted(t *testing.T) {
	server, _, _ := newTestServer(t)

	jobID := "job-results-pending"
	applyTestSignal(t, server, jobstatus.Signal{
		Kind: jobstatus.SignalJobCreated,
		JobCreated: &events.JobCreatedPayload{
			JobID:     jobID,
			InputType: models.JobInputTypeURLs,
			URLs:      []string{"https://example.com"},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/"+jobID+"/results", http.NoBody)
	rr := httptest.NewRecorder()

	server.handleJobResults(rr, req, jobID)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleJobResultsNotFound(t *testing.T) {
	server, _, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/missing/results", http.NoBody)
	rr := httptest.NewRecorder()

	server.handleJobResults(rr, req, "missing")

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestHandleJobResultsCompletedNoReport(t *testing.T) {
	server, _, _ := newTestServer(t)

	jobID := "job-results-no-report"
	applyTestSignal(t, server, jobstatus.Signal{
		Kind: jobstatus.SignalJobCreated,
		JobCreated: &events.JobCreatedPayload{
			JobID:     jobID,
			InputType: models.JobInputTypeURLs,
			URLs:      []string{"https://example.com"},
		},
	})
	applyTestSignal(t, server, jobstatus.Signal{
		Kind:         jobstatus.SignalJobCompleted,
		JobCompleted: &events.JobCompletedPayload{JobID: jobID, Artifacts: events.ArtifactLocations{}},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/"+jobID+"/results", http.NoBody)
	rr := httptest.NewRecorder()

	server.handleJobResults(rr, req, jobID)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestHandleJobResultsRedirect(t *testing.T) {
	server, _, _ := newTestServer(t)

	jobID := "job-results-redirect"
	applyTestSignal(t, server, jobstatus.Signal{
		Kind: jobstatus.SignalJobCreated,
		JobCreated: &events.JobCreatedPayload{
			JobID:     jobID,
			InputType: models.JobInputTypeURLs,
			URLs:      []string{"https://example.com"},
		},
	})
	applyTestSignal(t, server, jobstatus.Signal{
		Kind: jobstatus.SignalJobCompleted,
		JobCompleted: &events.JobCompletedPayload{
			JobID:     jobID,
			Artifacts: events.ArtifactLocations{ReportJSON: jobID + "/report.json"},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/"+jobID+"/results", http.NoBody)
	rr := httptest.NewRecorder()

	server.handleJobResults(rr, req, jobID)

	if rr.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected 307, got %d: %s", rr.Code, rr.Body.String())
	}

	loc := rr.Header().Get("Location")
	if loc == "" {
		t.Fatal("expected Location header for redirect")
	}
}

// --- buildJobStatusResponse ---
