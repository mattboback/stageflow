package api

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattboback/stageflow/libs/go/models"
	storagepkg "github.com/mattboback/stageflow/libs/go/storage"
	"github.com/mattboback/stageflow/services/platform-api/internal/jobstatus"
	"github.com/mattboback/stageflow/services/platform-api/internal/project"
	"github.com/mattboback/stageflow/services/platform-api/internal/status"
)

type staticJobReader struct {
	rec *status.JobRecord
}

type failingJobDeletionStore struct {
	*project.Store
	err error
}

func (s *failingJobDeletionStore) JobIsDeleted(context.Context, string) (bool, error) {
	return false, s.err
}

func (r *staticJobReader) GetJob(_ context.Context, jobID string) (*status.JobRecord, error) {
	if r.rec == nil || r.rec.JobID != jobID {
		return nil, status.ErrJobNotFound
	}

	clone := *r.rec

	return &clone, nil
}

func seedReadableJob(t *testing.T, server *Server, jobID string, state models.JobState) {
	t.Helper()

	reader := &staticJobReader{rec: &status.JobRecord{JobID: jobID, State: state}}
	server.config.StatusReader = reader
	server.jobStatus = jobstatus.New(&jobstatus.Config{CurrentReader: reader})
}

func TestHandleJobDeleteRemovesArtifactsAndHidesJob(t *testing.T) {
	t.Parallel()

	server, objectStore, _ := newTestServer(t)
	jobID := "job-delete-1"
	seedReadableJob(t, server, jobID, models.JobStateDone)

	if err := objectStore.UploadFile(
		context.Background(),
		storagepkg.BucketArtifacts,
		jobID+"/report.json",
		bytes.NewReader([]byte(`{"ok":true}`)),
		10,
	); err != nil {
		t.Fatalf("upload artifact: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/jobs/"+jobID, http.NoBody)
	rr := httptest.NewRecorder()
	server.handleJobDelete(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("delete status: want 204, got %d: %s", rr.Code, rr.Body.String())
	}

	exists, err := objectStore.FileExists(context.Background(), storagepkg.BucketArtifacts, jobID+"/report.json")
	if err != nil {
		t.Fatalf("FileExists: %v", err)
	}

	if exists {
		t.Fatal("expected artifact prefix to be removed")
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
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/jobs/job-1/report", http.NoBody)
	rr := httptest.NewRecorder()
	server.handleJobDelete(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestJobReadsFailClosedWhenDeletionLookupFails(t *testing.T) {
	t.Parallel()

	server, _, _ := newTestServer(t)
	jobID := "job-tombstone-error"
	seedReadableJob(t, server, jobID, models.JobStateDone)
	server.projectStore = &failingJobDeletionStore{
		Store: requireProjectStore(t, server),
		err:   errors.New("tombstone database unavailable"),
	}

	paths := []string{
		"/api/v1/jobs/" + jobID,
		"/api/v1/jobs/" + jobID + "/report",
		"/api/v1/jobs/" + jobID + "/results",
		"/api/v1/jobs/" + jobID + "/diff",
		"/api/v1/jobs/" + jobID + "/stream",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			rr := httptest.NewRecorder()
			server.Router().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, path, http.NoBody))

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
	server.projectStore = &failingJobDeletionStore{
		Store: requireProjectStore(t, server),
		err:   errors.New("tombstone database unavailable"),
	}

	const artifact = "report.json"
	if err := objectStore.UploadFile(
		context.Background(),
		storagepkg.BucketArtifacts,
		jobID+"/"+artifact,
		bytes.NewReader([]byte(`{"ok":true}`)),
		10,
	); err != nil {
		t.Fatalf("upload artifact: %v", err)
	}

	rr := httptest.NewRecorder()
	server.Router().ServeHTTP(
		rr,
		httptest.NewRequest(http.MethodDelete, "/api/v1/jobs/"+jobID, http.NoBody),
	)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d: %s", rr.Code, rr.Body.String())
	}

	exists, err := objectStore.FileExists(
		context.Background(),
		storagepkg.BucketArtifacts,
		jobID+"/"+artifact,
	)
	if err != nil {
		t.Fatalf("FileExists: %v", err)
	}
	if !exists {
		t.Fatal("artifact was deleted after tombstone lookup failed")
	}
}
