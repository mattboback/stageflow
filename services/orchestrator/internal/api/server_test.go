package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mattboback/stageflow/libs/go/models"
	db "github.com/mattboback/stageflow/services/orchestrator/internal/adapters/repository"
	podman "github.com/mattboback/stageflow/services/orchestrator/internal/adapters/runtime"
)

type fakePodmanClient struct {
	pods     []podman.PodInfo
	podsByID map[string]podman.PodInfo
}

var apiTestSchemaCounter uint64

const testAPIToken = "test-token"

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

// newTestDatabase creates an isolated PostgreSQL schema and seeds two jobs plus a single event.
func newTestDatabase(t *testing.T) (*db.Database, func()) {
	t.Helper()

	admin, err := sql.Open("pgx", testDatabaseURL)
	if err != nil {
		t.Fatalf("connect admin database: %v", err)
	}

	schema := fmt.Sprintf("t_%d_%d", time.Now().UnixNano(), atomic.AddUint64(&apiTestSchemaCounter, 1))

	createSchemaQuery := fmt.Sprintf("CREATE SCHEMA %s", quoteIdentifier(schema))
	if _, execErr := admin.ExecContext(context.Background(), createSchemaQuery); execErr != nil {
		t.Fatalf("create test schema: %v", execErr)
	}

	databaseURL := fmt.Sprintf("%s&search_path=%s", testDatabaseURL, schema)

	database, err := db.NewDatabase(&db.Config{URL: databaseURL})
	if err != nil {
		t.Fatalf("create test database: %v", err)
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
		if createErr := database.CreateJob(context.Background(), job); createErr != nil {
			t.Fatalf("seed job %s: %v", job.ID, createErr)
		}
	}

	if insertErr := database.InsertJobEvent(context.Background(), &db.JobEventInsert{
		JobID:     "123",
		Event:     "job.created",
		Timestamp: time.Now().UTC(),
		Payload:   "{}",
	}); insertErr != nil {
		t.Fatalf("seed job event: %v", insertErr)
	}

	cleanup := func() {
		_ = database.Close()
		dropSchemaQuery := fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", quoteIdentifier(schema))
		_, _ = admin.ExecContext(context.Background(), dropSchemaQuery)
		_ = admin.Close()
	}

	return database, cleanup
}

// newTestServer wires a Server with a temporary database and Podman test double.
func newTestServer(t *testing.T) (*Server, func()) {
	t.Helper()

	dbase, databaseCleanup := newTestDatabase(t)
	podClient := newFakePodmanClient()

	server := NewServer(&Config{
		Database:     dbase,
		PodmanClient: podClient,
		APIToken:     testAPIToken,
		Port:         "0",
	})

	cleanup := func() {
		databaseCleanup()
	}

	return server, cleanup
}

func quoteIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}

func newAuthedRequest(method, target string) *http.Request {
	req := httptest.NewRequest(method, target, nil)
	req.Header.Set("X-Api-Key", testAPIToken)

	return req
}

func TestRequireAuthRejectsEmptyConfiguredToken(t *testing.T) {
	srv := NewServer(&Config{APIToken: "", Port: "0"})
	called := false
	handler := srv.requireAuth(func(w http.ResponseWriter, _ *http.Request) {
		called = true

		w.WriteHeader(http.StatusNoContent)
	})

	requests := []*http.Request{
		httptest.NewRequest(http.MethodGet, "/api/v1/jobs", nil),
		httptest.NewRequest(http.MethodGet, "/api/v1/jobs", nil),
		httptest.NewRequest(http.MethodGet, "/api/v1/jobs", nil),
	}
	requests[1].Header.Set("Authorization", "Bearer ")
	requests[2].Header.Set("X-Api-Key", "anything")

	for _, req := range requests {
		called = false
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", rec.Code)
		}

		if called {
			t.Fatal("handler was called with empty configured API token")
		}
	}
}

func TestRequireAuthAllowsValidToken(t *testing.T) {
	srv := NewServer(&Config{APIToken: testAPIToken, Port: "0"})
	handler := srv.requireAuth(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, newAuthedRequest(http.MethodGet, "/api/v1/jobs"))

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rec.Code)
	}
}

func TestRequestShutdownGateRejectsNewWorkAndJoinsActiveHandler(t *testing.T) {
	t.Parallel()

	srv := NewServer(&Config{APIToken: testAPIToken, Port: "0"})
	started := make(chan struct{})
	release := make(chan struct{})
	waited := make(chan struct{})

	handler := srv.trackRequests(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(started)
		<-release
		w.WriteHeader(http.StatusNoContent)
	}))

	firstResponse := httptest.NewRecorder()

	go handler.ServeHTTP(firstResponse, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	<-started

	srv.stopAcceptingRequests()

	go func() {
		_ = srv.waitForRequests(context.Background())

		close(waited)
	}()

	secondResponse := httptest.NewRecorder()
	handler.ServeHTTP(secondResponse, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if secondResponse.Code != http.StatusServiceUnavailable {
		t.Fatalf("new request status = %d, want %d", secondResponse.Code, http.StatusServiceUnavailable)
	}

	select {
	case <-waited:
		t.Fatal("waitForRequests returned before the active handler exited")
	case <-time.After(25 * time.Millisecond):
	}

	close(release)

	select {
	case <-waited:
	case <-time.After(time.Second):
		t.Fatal("waitForRequests did not return after the active handler exited")
	}

	if firstResponse.Code != http.StatusNoContent {
		t.Fatalf("active request status = %d, want %d", firstResponse.Code, http.StatusNoContent)
	}
}

func TestShutdownForceClosesAndJoinsTimedOutHandler(t *testing.T) {
	t.Parallel()

	srv := NewServer(&Config{APIToken: testAPIToken, Port: "0"})
	started := make(chan struct{})
	exited := make(chan struct{})

	srv.server.Handler = srv.trackRequests(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
		close(exited)
	}))

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}

	serveDone := make(chan error, 1)

	go func() {
		serveDone <- srv.server.Serve(listener)
	}()

	requestDone := make(chan error, 1)

	go func() {
		client := &http.Client{Timeout: time.Second}

		response, requestErr := client.Get("http://" + listener.Addr().String() + "/healthz")
		if response != nil {
			_ = response.Body.Close()
		}

		requestDone <- requestErr
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for the request handler to start")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()

	if shutdownErr := srv.Shutdown(shutdownCtx); !errors.Is(shutdownErr, context.DeadlineExceeded) {
		t.Fatalf("Shutdown() error = %v, want context deadline exceeded", shutdownErr)
	}

	select {
	case <-exited:
	default:
		t.Fatal("Shutdown returned before the force-closed handler exited")
	}

	select {
	case requestErr := <-requestDone:
		if requestErr == nil {
			t.Fatal("force-closed request unexpectedly completed without an error")
		}
	case <-time.After(time.Second):
		t.Fatal("force-closed client request did not return")
	}

	select {
	case serveErr := <-serveDone:
		if !errors.Is(serveErr, http.ErrServerClosed) {
			t.Fatalf("Serve() error = %v, want http.ErrServerClosed", serveErr)
		}
	case <-time.After(time.Second):
		t.Fatal("HTTP server did not stop")
	}
}

//nolint:gocyclo // The test asserts every bounded-shutdown outcome explicitly.
func TestShutdownBoundsHandlerDrainAfterForceClose(t *testing.T) {
	t.Parallel()

	srv := NewServer(&Config{APIToken: testAPIToken, Port: "0"})
	srv.forcedDrainWait = 25 * time.Millisecond

	started := make(chan struct{})
	release := make(chan struct{})
	exited := make(chan struct{})
	released := false

	defer func() {
		if !released {
			close(release)
		}
	}()

	srv.server.Handler = srv.trackRequests(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		close(started)
		<-release
		close(exited)
	}))

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}

	serveDone := make(chan error, 1)

	go func() {
		serveDone <- srv.server.Serve(listener)
	}()

	requestDone := make(chan error, 1)

	go func() {
		client := &http.Client{Timeout: time.Second}

		response, requestErr := client.Get("http://" + listener.Addr().String() + "/healthz")
		if response != nil {
			_ = response.Body.Close()
		}

		requestDone <- requestErr
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for the request handler to start")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	shutdownErr := srv.Shutdown(shutdownCtx)
	if !errors.Is(shutdownErr, context.DeadlineExceeded) {
		t.Fatalf("Shutdown() error = %v, want context deadline exceeded", shutdownErr)
	}

	if !errors.Is(shutdownErr, ErrRequestDrainTimeout) {
		t.Fatalf("Shutdown() error = %v, want ErrRequestDrainTimeout", shutdownErr)
	}

	select {
	case <-exited:
		t.Fatal("handler that ignores cancellation unexpectedly exited")
	default:
	}

	close(release)

	released = true

	select {
	case <-exited:
	case <-time.After(time.Second):
		t.Fatal("handler did not exit after release")
	}

	select {
	case <-requestDone:
	case <-time.After(time.Second):
		t.Fatal("force-closed client request did not return")
	}

	select {
	case serveErr := <-serveDone:
		if !errors.Is(serveErr, http.ErrServerClosed) {
			t.Fatalf("Serve() error = %v, want http.ErrServerClosed", serveErr)
		}
	case <-time.After(time.Second):
		t.Fatal("HTTP server did not stop")
	}
}

func TestHandleListJobsFiltersAndPagination(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	req := newAuthedRequest(http.MethodGet, "/api/v1/jobs?state=PENDING&limit=1&offset=0")
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

	req := newAuthedRequest(http.MethodGet, "/api/v1/jobs/123")
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

	req := newAuthedRequest(http.MethodGet, "/api/v1/jobs/123/events?limit=10&offset=0")
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

	req := newAuthedRequest(http.MethodGet, "/api/v1/jobs/missing")
	rec := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}
}

func TestHandleListPodsEnrichesJobState(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	req := newAuthedRequest(http.MethodGet, "/api/v1/pods")
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

	req := newAuthedRequest(http.MethodGet, "/api/v1/pods/pod-1")
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

	req := newAuthedRequest(http.MethodGet, "/api/v1/status")
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

	req := newAuthedRequest(http.MethodPost, "/api/v1/jobs")
	rec := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for POST, got %d", rec.Code)
	}
}

func TestRecoveryMiddlewareReturnsStructured500(t *testing.T) {
	t.Parallel()

	srv := NewServer(&Config{APIToken: testAPIToken, Port: "0"})
	handler := srv.recoverPanic(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/jobs/123", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "Internal server error") {
		t.Fatalf("expected structured error response, got %q", rec.Body.String())
	}
}

func TestRateLimitMiddlewareRejectsExcessRequests(t *testing.T) {
	t.Parallel()

	srv := NewServer(&Config{APIToken: testAPIToken, Port: "0", AdminRateLimitRPS: 1, AdminRateLimitBurst: 1})
	handler := srv.rateLimit(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	first := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs", nil)
	req.RemoteAddr = "192.0.2.10:12345"
	handler.ServeHTTP(first, req)

	if first.Code != http.StatusNoContent {
		t.Fatalf("first request status = %d, want 204", first.Code)
	}

	second := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/jobs", nil)
	req.RemoteAddr = "192.0.2.10:54321"
	handler.ServeHTTP(second, req)

	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second request status = %d, want 429", second.Code)
	}
}

func TestHandleMetrics(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	req := newAuthedRequest(http.MethodGet, "/metrics")
	rec := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	for _, want := range []string{
		"stageflow_orchestrator_jobs_total 2",
		"stageflow_orchestrator_jobs_by_state{state=\"PENDING\"} 1",
		"stageflow_orchestrator_pods_total 2",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics body missing %q; body:\n%s", want, body)
		}
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
