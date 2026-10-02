// Job status, report, and results retrieval -- redirects, not-found, and
// method handling. Exercises handlers_jobs_status.go.

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattboback/stageflow/libs/go/models"
)

func TestHandleJobStatusFound(t *testing.T) {
	server, _, _ := newTestServer(t)

	jobID := "job-status-found"
	testJobs(server).put(scanningURLJob(jobID))

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

// --- /report route removed in PR 1.7d ---

func TestHandleJobReportRouteGone(t *testing.T) {
	server, _, _ := newTestServer(t)

	jobID := "job-report-gone"
	testJobs(server).put(scanningURLJob(jobID))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/"+jobID+"/report", http.NoBody)
	rr := httptest.NewRecorder()

	server.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for the removed report route, got %d: %s", rr.Code, rr.Body.String())
	}
}

// --- handleJobResults ---

func TestHandleJobResultsNotCompleted(t *testing.T) {
	server, _, _ := newTestServer(t)

	jobID := "job-results-pending"
	testJobs(server).put(scanningURLJob(jobID))

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
	testJobs(server).put(scanningURLJob(jobID))
	testJobs(server).finish(t, jobID, "")

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
	testJobs(server).put(scanningURLJob(jobID))
	testJobs(server).finish(t, jobID, jobID+"/report.json")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/"+jobID+"/results", http.NoBody)
	rr := httptest.NewRecorder()

	server.handleJobResults(rr, req, jobID)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	if rr.Header().Get("Content-Type") != "application/json" || rr.Body.Len() == 0 {
		t.Fatal("expected JSON report body")
	}
}

// --- buildJobStatusResponse ---
