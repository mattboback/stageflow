package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/mattboback/stageflow/packages/shared-go/models"
	"github.com/mattboback/stageflow/platform/orchestrator/internal/db"
	"github.com/mattboback/stageflow/platform/orchestrator/internal/podman"
)

type fakePodmanClient struct {
	pods     []podman.PodInfo
	podsByID map[string]podman.PodInfo
}

func newFakePodmanClient() *fakePodmanClient {
	pods := []podman.PodInfo{
		{ID: "pod-1", Name: "job-123", Status: "running"},
		{ID: "pod-2", Name: "misc", Status: "exited"},
	}

	podsByID := make(map[string]podman.PodInfo, len(pods))
	for _, pod := range pods {
		podsByID[pod.ID] = pod
	}

	return &fakePodmanClient{
		pods:     pods,
		podsByID: podsByID,
	}
}

func (f *fakePodmanClient) ListPods(_ context.Context) ([]podman.PodInfo, error) {
	return f.pods, nil
}

func (f *fakePodmanClient) InspectPod(_ context.Context, podID string) (*podman.PodInfo, error) {
	pod, ok := f.podsByID[podID]
	if !ok {
		return nil, errors.New("pod not found")
	}

	return &pod, nil
}

// newTestDatabase creates a fresh SQLite database and seeds two jobs plus a single event.
func newTestDatabase(t *testing.T) *db.Database {
	t.Helper()

	path := filepath.Join(t.TempDir(), "jobs.db")
	database, err := db.NewDatabase(&db.Config{Path: path})
	if err != nil {
		t.Fatalf("create database: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	jobs := []*models.Job{
		{
			ID:        "123",
			State:     models.JobStatePending,
			InputType: "zip",
			InputPath: "/tmp/a.zip",
			Config:    models.JobConfig{Modules: []string{"axe"}, Screenshot: true},
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        "456",
			State:     models.JobStateDone,
			InputType: "urls",
			URLs:      []string{"https://example.com"},
			Config:    models.JobConfig{Modules: []string{"axe", "kb"}, Screenshot: false},
			CreatedAt: now.Add(-time.Minute),
			UpdatedAt: now.Add(-time.Minute),
		},
	}

	for _, job := range jobs {
		if err := database.CreateJob(context.Background(), job); err != nil {
			t.Fatalf("seed job %s: %v", job.ID, err)
		}
	}

	if err := database.InsertJobEvent(context.Background(), &db.JobEventInsert{
		JobID:     "123",
		Event:     "job.created",
		Timestamp: time.Now().UTC(),
		Payload:   "{}",
	}); err != nil {
		t.Fatalf("seed job event: %v", err)
	}

	return database
}

// newTestServer wires a Server with a temporary database and Podman test double.
func newTestServer(t *testing.T) (*Server, func()) {
	t.Helper()

	dbase := newTestDatabase(t)
	podClient := newFakePodmanClient()

	server := NewServer(&Config{
		Database:     dbase,
		PodmanClient: podClient,
		Port:         "0",
	})

	cleanup := func() {
		_ = dbase.Close()
	}

	return server, cleanup
}

func TestHandleListJobsFiltersAndPagination(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs?state=PENDING&limit=1&offset=0", nil)
	rec := httptest.NewRecorder()

	srv.server.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp struct {
		Jobs   []*models.Job `json:"jobs"`
		Total  int           `json:"total"`
		Limit  int           `json:"limit"`
		Offset int           `json:"offset"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Total != 1 {
		t.Fatalf("expected total 1 pending job, got %d", resp.Total)
	}
	if len(resp.Jobs) != 1 || resp.Jobs[0].ID != "123" {
		t.Fatalf("unexpected jobs payload: %+v", resp.Jobs)
	}
	if resp.Limit != 1 || resp.Offset != 0 {
		t.Fatalf("limit/offset not echoed back (limit=%d offset=%d)", resp.Limit, resp.Offset)
	}
}

func TestHandleJobRoutes_GetJob(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/123", nil)
	rec := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp struct {
		Job *models.Job `json:"job"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Job == nil || resp.Job.ID != "123" {
		t.Fatalf("unexpected job payload: %#v", resp.Job)
	}
}

func TestHandleJobRoutes_GetJobEvents(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/123/events?limit=10&offset=0", nil)
	rec := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp struct {
		JobID  string        `json:"job_id"`
		Events []db.JobEvent `json:"events"`
		Limit  int           `json:"limit"`
		Offset int           `json:"offset"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.JobID != "123" {
		t.Fatalf("expected job_id 123, got %q", resp.JobID)
	}
	if resp.Limit != 10 || resp.Offset != 0 {
		t.Fatalf("limit/offset not echoed back (limit=%d offset=%d)", resp.Limit, resp.Offset)
	}
	if len(resp.Events) != 1 || resp.Events[0].Event != "job.created" {
		t.Fatalf("unexpected events payload: %#v", resp.Events)
	}
}

func TestHandleJobRoutes_JobNotFound(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/missing", nil)
	rec := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}
}

func TestHandleListPodsEnrichesJobState(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pods", nil)
	rec := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp struct {
		Pods  []map[string]any `json:"pods"`
		Total int              `json:"total"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Total != 2 {
		t.Fatalf("expected 2 pods, got %d", resp.Total)
	}

	first := resp.Pods[0]
	if got := first["job_id"]; got != "123" {
		t.Fatalf("expected pod to map to job 123, got %#v", got)
	}
	if got := first["job_state"]; got != string(models.JobStatePending) {
		t.Fatalf("expected job_state %q, got %#v", models.JobStatePending, got)
	}
}

func TestHandlePodDetails(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pods/pod-1", nil)
	rec := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var pod podman.PodInfo
	if err := json.NewDecoder(rec.Body).Decode(&pod); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if pod.ID != "pod-1" || pod.Name != "job-123" {
		t.Fatalf("unexpected pod payload: %#v", pod)
	}
}

func TestHandleSystemStatusAggregatesCounts(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	rec := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	jobs, ok := resp["jobs"].(map[string]any)
	if !ok {
		t.Fatalf("jobs payload missing or wrong type: %#v", resp["jobs"])
	}
	if total := intFromAny(jobs["total"]); total != 2 {
		t.Fatalf("expected total jobs 2, got %d", total)
	}
	byState, ok := jobs["by_state"].(map[string]any)
	if !ok {
		t.Fatalf("by_state payload missing")
	}
	if pending := intFromAny(byState[string(models.JobStatePending)]); pending != 1 {
		t.Fatalf("expected 1 pending job, got %d", pending)
	}

	pods, ok := resp["pods"].(map[string]any)
	if !ok {
		t.Fatalf("pods payload missing")
	}
	if totalPods := intFromAny(pods["total"]); totalPods != 2 {
		t.Fatalf("expected 2 pods, got %d", totalPods)
	}
}

func TestHandleHealth(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["status"] != "healthy" {
		t.Fatalf("unexpected health payload: %#v", resp)
	}
}

func TestHandleListJobsRejectsWrongMethod(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", nil)
	rec := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for POST, got %d", rec.Code)
	}
}

// intFromAny safely converts JSON numbers to int for assertions.
func intFromAny(v any) int {
	switch value := v.(type) {
	case float64:
		return int(value)
	case int:
		return value
	default:
		return 0
	}
}
