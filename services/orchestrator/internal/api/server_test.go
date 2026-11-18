package api

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestServer(t *testing.T) (*Server, func()) {
	t.Helper()
	return NewServer(&Config{Port: "0"}), func() {}
}

func TestRequestShutdownGateRejectsNewWorkAndJoinsActiveHandler(t *testing.T) {
	t.Parallel()

	srv := NewServer(&Config{Port: "0"})
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

	srv := NewServer(&Config{Port: "0"})
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

	srv := NewServer(&Config{Port: "0"})
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

func TestRemovedRoutesReturnNotFound(t *testing.T) {
	t.Parallel()

	srv := NewServer(&Config{Port: "0"})

	for _, path := range []string{
		"/api/v1/jobs", "/api/v1/jobs/",
		"/api/v1/jobs/123/events", "/api/v1/jobs/123/events/",
		"/api/v1/pods", "/api/v1/pods/", "/api/v1/pods/pod-1",
		"/api/v1/status", "/api/v1/status/", "/metrics", "/metrics/",
	} {
		for _, token := range []string{"", "obsolete-token"} {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.Header.Set("X-Api-Key", token)

			rec := httptest.NewRecorder()
			srv.server.Handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusNotFound {
				t.Errorf("GET %s with token %q: got %d, want 404", path, token, rec.Code)
			}

			if location := rec.Header().Get("Location"); location != "" {
				t.Errorf("GET %s redirected to %q", path, location)
			}
		}
	}
}
