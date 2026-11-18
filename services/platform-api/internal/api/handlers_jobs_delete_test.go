package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattboback/stageflow/internal/jobstore"
	"github.com/mattboback/stageflow/libs/go/models"
)

type staticJobReader struct {
	rec *jobstore.JobRecord
}

type failingJobDeletionStore struct {
	JobStatusReader
	err error
}

func (s *failingJobDeletionStore) GetJob(context.Context, string) (*jobstore.JobRecord, error) {
	return nil, s.err
}

func (r *staticJobReader) GetJob(_ context.Context, jobID string) (*jobstore.JobRecord, error) {
	if r.rec == nil || r.rec.JobID != jobID {
		return nil, jobstore.ErrJobNotFound
	}

	clone := *r.rec

	return &clone, nil
}

func seedReadableJob(t *testing.T, server *Server, jobID string, state models.JobState) {
	t.Helper()

	testJobs(server).put(&models.Job{ID: jobID, State: state})
}

func TestHandleJobDeleteRemovesArtifactsAndHidesJob(t *testing.T) {
	t.Parallel()

	server, objectStore, _ := newTestServer(t)
	jobID := "job-delete-1"
	seedReadableJob(t, server, jobID, models.JobStateDone)

	objectStore.uploads[jobID+"/report.json"] = []byte(`{"ok":true}`)
	objectStore.uploads[jobID+"/input.zip"] = []byte("zip")

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/jobs/"+jobID, http.NoBody)
	rr := httptest.NewRecorder()
	server.handleJobDelete(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("delete status: want 204, got %d: %s", rr.Code, rr.Body.String())
	}

	if len(objectStore.uploads) != 0 {
		t.Fatalf("job files left after delete: %d", len(objectStore.uploads))
	}

	statusReq := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/"+jobID, http.NoBody)
	statusRR := httptest.NewRecorder()
	server.handleJobStatus(statusRR, statusReq)

	if statusRR.Code != http.StatusNotFound {
		t.Fatalf("get after delete: want 404, got %d: %s", statusRR.Code, statusRR.Body.String())
	}

	replay := httptest.NewRecorder()
	server.handleJobDelete(replay, httptest.NewRequest(http.MethodDelete, "/api/v1/jobs/"+jobID, http.NoBody))

	if replay.Code != http.StatusNoContent {
		t.Fatalf("idempotent delete: want 204, got %d", replay.Code)
	}
}

func TestHandleJobDeleteRejectsRunningJob(t *testing.T) {
	t.Parallel()

	server, _, _ := newTestServer(t)
	jobID := "job-running-1"
	seedReadableJob(t, server, jobID, models.JobStateScanning)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/jobs/"+jobID, http.NoBody)
	rr := httptest.NewRecorder()
	server.handleJobDelete(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("want 409, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleJobDeleteRejectsSuffixPaths(t *testing.T) {
	t.Parallel()

	server, _, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/jobs/job-1/results", http.NoBody)
	rr := httptest.NewRecorder()
	server.handleJobDelete(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestJobReadsFailClosedWhenDeletionLookupFails(t *testing.T) {
	t.Parallel()

	server, _, _ := newTestServer(t)
	jobID := "job-read-error"
	seedReadableJob(t, server, jobID, models.JobStateDone)
	server.statusReader = &failingJobDeletionStore{
		JobStatusReader: server.statusReader,
		err:             errors.New("job database unavailable"),
	}

	paths := []string{
		"/api/v1/jobs/" + jobID,
		"/api/v1/jobs/" + jobID + "/results",
		"/api/v1/jobs/" + jobID + "/stream",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, path, http.NoBody)

			rr := httptest.NewRecorder()
			server.Router().ServeHTTP(rr, req)

			if rr.Code != http.StatusInternalServerError {
				t.Fatalf("want 500, got %d: %s", rr.Code, rr.Body.String())
			}
		})
	}
}

func TestHandleJobDeleteFailsClosedWhenDeletionLookupFails(t *testing.T) {
	t.Parallel()

	server, objectStore, _ := newTestServer(t)
	jobID := "job-delete-lookup-error"
	seedReadableJob(t, server, jobID, models.JobStateDone)
	server.statusReader = &failingJobDeletionStore{
		JobStatusReader: server.statusReader,
		err:             errors.New("job database unavailable"),
	}

	const artifact = "report.json"

	objectStore.uploads[jobID+"/"+artifact] = []byte(`{"ok":true}`)

	rr := httptest.NewRecorder()
	server.Router().ServeHTTP(
		rr,
		httptest.NewRequest(http.MethodDelete, "/api/v1/jobs/"+jobID, http.NoBody),
	)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d: %s", rr.Code, rr.Body.String())
	}

	if _, exists := objectStore.uploads[jobID+"/"+artifact]; !exists {
		t.Fatal("artifact was deleted after database lookup failed")
	}
}
