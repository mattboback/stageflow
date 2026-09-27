package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/mattboback/stageflow/services/platform-api/internal/jobstatus"
	"github.com/mattboback/stageflow/services/platform-api/internal/messaging"
)

func TestStartHTTPServerReturnsBindFailure(t *testing.T) {
	t.Parallel()

	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	defer occupied.Close()

	server := &http.Server{
		Addr:              occupied.Addr().String(),
		Handler:           http.NewServeMux(),
		ReadHeaderTimeout: time.Second,
	}

	if _, startErr := startHTTPServer(server); startErr == nil {
		t.Fatal("startHTTPServer() error = nil, want occupied-port error")
	}
}

type fakeStatusEventSubscriber struct {
	err    error
	called bool
}

func (f *fakeStatusEventSubscriber) SubscribeToStatusEvents(
	_ context.Context,
	handler messaging.EventHandler,
) error {
	f.called = true

	if handler == nil {
		return errors.New("handler is required")
	}

	return f.err
}

func TestSubscribeStatusEventsReturnsSubscriptionError(t *testing.T) {
	t.Parallel()

	expected := errors.New("jetstream unavailable")
	subscriber := &fakeStatusEventSubscriber{err: expected}

	err := subscribeStatusEvents(context.Background(), subscriber, jobstatus.New(&jobstatus.Config{}))
	if !errors.Is(err, expected) {
		t.Fatalf("expected subscription error to propagate, got %v", err)
	}

	if !subscriber.called {
		t.Fatal("expected subscriber to be called")
	}
}

func TestNewHTTPServer(t *testing.T) {
	t.Parallel()

	handler := http.NewServeMux()
	port := 8080

	server := newHTTPServer(port, handler)

	if server == nil {
		t.Fatal("expected server to be non-nil")
	}

	expectedAddr := ":8080"
	if server.Addr != expectedAddr {
		t.Errorf("expected Addr %s, got %s", expectedAddr, server.Addr)
	}

	if server.Handler != handler {
		t.Error("expected handler to match provided handler")
	}

	if server.ReadHeaderTimeout != 5*time.Second {
		t.Errorf("expected ReadHeaderTimeout 5s, got %v", server.ReadHeaderTimeout)
	}

	if server.ReadTimeout != 15*time.Second {
		t.Errorf("expected ReadTimeout 15s, got %v", server.ReadTimeout)
	}

	if server.WriteTimeout != 0 {
		t.Errorf("expected WriteTimeout 0 (disabled for SSE), got %v", server.WriteTimeout)
	}

	if server.IdleTimeout != 60*time.Second {
		t.Errorf("expected IdleTimeout 60s, got %v", server.IdleTimeout)
	}
}

func TestNewHTTPServer_DifferentPorts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		port         int
		expectedAddr string
	}{
		{port: 80, expectedAddr: ":80"},
		{port: 3000, expectedAddr: ":3000"},
		{port: 9999, expectedAddr: ":9999"},
	}

	for _, tt := range tests {
		t.Run(tt.expectedAddr, func(t *testing.T) {
			t.Parallel()

			handler := http.NewServeMux()
			server := newHTTPServer(tt.port, handler)

			if server.Addr != tt.expectedAddr {
				t.Errorf("expected Addr %s, got %s", tt.expectedAddr, server.Addr)
			}
		})
	}
}

func TestLoadScannerRegistry_LoadsBuiltins(t *testing.T) {
	logger := slog.Default()

	registry := loadScannerRegistry(logger)

	if registry == nil {
		t.Fatal("expected registry to be non-nil")
	}

	// Should have default scanners loaded
	if registry.Count() == 0 {
		t.Error("expected registry to have at least one scanner")
	}
}

func TestLoadScannerRegistry_CountsScannersCorrectly(t *testing.T) {
	logger := slog.Default()

	registry := loadScannerRegistry(logger)

	if registry == nil {
		t.Fatal("expected registry to be non-nil")
	}

	totalCount := registry.Count()
	enabledCount := registry.CountEnabled()

	if totalCount == 0 {
		t.Error("expected at least one scanner in registry")
	}

	if enabledCount > totalCount {
		t.Errorf("enabled count (%d) cannot exceed total count (%d)", enabledCount, totalCount)
	}
}

func TestNewHTTPServer_HandlerIsSet(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := newHTTPServer(8080, mux)

	if server.Handler == nil {
		t.Error("expected handler to be set")
	}

	// Verify handler works
	if server.Handler != mux {
		t.Error("handler should be the same as provided mux")
	}
}

func TestNewHTTPServer_TimeoutsForSSE(t *testing.T) {
	t.Parallel()

	server := newHTTPServer(8080, http.NewServeMux())

	// WriteTimeout should be 0 to support long-lived SSE connections
	if server.WriteTimeout != 0 {
		t.Errorf("WriteTimeout should be 0 for SSE support, got %v", server.WriteTimeout)
	}

	// Other timeouts should be set for security
	if server.ReadHeaderTimeout == 0 {
		t.Error("ReadHeaderTimeout should be set to prevent slowloris attacks")
	}

	if server.ReadTimeout == 0 {
		t.Error("ReadTimeout should be set")
	}

	if server.IdleTimeout == 0 {
		t.Error("IdleTimeout should be set to clean up idle connections")
	}
}
