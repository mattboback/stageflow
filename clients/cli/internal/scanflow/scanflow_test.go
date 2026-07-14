package scanflow

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
	"github.com/mattboback/stageflow/clients/cli/internal/testsupport"
)

func TestSubmitURLsAndWait(t *testing.T) {
	t.Parallel()

	const jobID = "job-123"
	var submitted apiclient.SubmitJobRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/jobs/urls":
			if err := json.NewDecoder(r.Body).Decode(&submitted); err != nil {
				t.Errorf("decode submission: %v", err)
			}
			writeJSON(t, w, apiclient.SubmitJobResponse{JobID: jobID})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/jobs/"+jobID:
			writeJSON(t, w, apiclient.JobStatus{ID: jobID, State: apiclient.JobStateDone})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/jobs/"+jobID+"/results":
			writeJSON(t, w, testsupport.SampleReport(jobID))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	var progress bytes.Buffer
	request := apiclient.SubmitJobRequest{URLs: []string{"https://example.com"}, Modules: []string{"axe"}}
	result, err := SubmitURLsAndWait(
		context.Background(),
		apiclient.NewClient(server.URL, "", server.Client()),
		request,
		time.Second,
		WaitOptions{Progress: &progress, NoStream: true},
	)
	testsupport.RequireNoErr(t, err)
	testsupport.RequireEqual(t, result.Status.ID, jobID, "status ID")
	testsupport.RequireDeepEqual(t, submitted.URLs, request.URLs, "submitted URLs")
	testsupport.RequireEqual(
		t,
		progress.String(),
		"Job submitted: job-123\nWaiting for completion...\nstate: DONE\n",
		"progress output",
	)
}

func TestSubmitURLsAndWaitEnhancesPrivateTargetError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "disallowed address", http.StatusBadRequest)
	}))
	defer server.Close()

	_, err := SubmitURLsAndWait(
		context.Background(),
		apiclient.NewClient(server.URL, "", server.Client()),
		apiclient.SubmitJobRequest{URLs: []string{"http://127.0.0.1"}},
		time.Second,
		WaitOptions{Progress: &bytes.Buffer{}, NoStream: true},
	)
	if err == nil || !strings.Contains(err.Error(), "local/private targets require allow_private_targets") {
		t.Fatalf("error = %v", err)
	}
}

func TestWaitForReportRejectsFailedTerminalState(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, apiclient.JobStatus{ID: "failed-job", State: apiclient.JobStateFailed, Error: "boom"})
	}))
	defer server.Close()

	_, err := WaitForReport(
		context.Background(),
		apiclient.NewClient(server.URL, "", server.Client()),
		"failed-job",
		WaitOptions{Progress: &bytes.Buffer{}, NoStream: true},
	)
	if err == nil || err.Error() != "job failed: boom" {
		t.Fatalf("error = %v", err)
	}
}

func TestRequireJobID(t *testing.T) {
	t.Parallel()

	if _, err := RequireJobID(apiclient.SubmitJobResponse{}); err == nil || err.Error() != "missing job_id in response" {
		t.Fatalf("error = %v", err)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Errorf("encode response: %v", err)
	}
}
