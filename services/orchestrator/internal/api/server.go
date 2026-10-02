// Package api exposes the orchestrator's HTTP endpoints.
package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/mattboback/stageflow/libs/go/httputil"
)

const defaultForcedDrainWait = time.Second

// ErrRequestDrainTimeout means force-closing HTTP connections did not make
// every handler exit. Callers must not close handler dependencies before the
// process terminates.
var ErrRequestDrainTimeout = errors.New("orchestrator API handlers did not drain")

// Server provides private job status reads and a health endpoint.
type Server struct {
	health          func(context.Context) error
	port            string
	server          *http.Server
	requestsMu      sync.Mutex
	requestsWG      sync.WaitGroup
	requestsStop    bool
	forcedDrainWait time.Duration
}

// Config contains configuration for the API server.
type Config struct {
	Health func(context.Context) error
	Port   string // Default: 8081
}

// NewServer creates a new API server.
func NewServer(cfg *Config) *Server {
	if cfg.Port == "" {
		cfg.Port = "8081"
	}

	s := &Server{
		port:            cfg.Port,
		health:          cfg.Health,
		forcedDrainWait: defaultForcedDrainWait,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", s.handleHealth)

	s.server = &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           s.trackRequests(mux),
		ReadHeaderTimeout: 15 * time.Second,
	}

	return s
}

func (s *Server) trackRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.requestsMu.Lock()
		if s.requestsStop {
			s.requestsMu.Unlock()
			httputil.RespondError(w, http.StatusServiceUnavailable, "Service is shutting down")

			return
		}

		s.requestsWG.Add(1)
		s.requestsMu.Unlock()

		defer s.requestsWG.Done()

		next.ServeHTTP(w, r)
	})
}

func (s *Server) stopAcceptingRequests() {
	s.requestsMu.Lock()
	s.requestsStop = true
	s.requestsMu.Unlock()
}

func (s *Server) waitForRequests(ctx context.Context) error {
	// stopAcceptingRequests prevents Add calls after this Wait begins.
	done := make(chan struct{})

	go func() {
		s.requestsWG.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	slog.Info("Starting orchestrator API server", "port", s.port)

	if checkErr1 := s.server.ListenAndServe(); checkErr1 != nil && checkErr1 != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", checkErr1)
	}

	return nil
}

// Shutdown closes HTTP intake and waits for every active handler. When the
// graceful deadline expires, it force-closes connections and still joins the
// handlers so callers may safely close shared dependencies afterward.
func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("Shutting down orchestrator API server...")
	s.stopAcceptingRequests()

	shutdownErr := s.server.Shutdown(ctx)
	if shutdownErr == nil {
		// A successful http.Server shutdown has already joined active
		// connections; this barrier therefore returns immediately.
		return s.waitForRequests(context.Background())
	}

	closeErr := s.server.Close()
	if closeErr != nil && !errors.Is(closeErr, http.ErrServerClosed) {
		shutdownErr = errors.Join(shutdownErr, closeErr)
	}

	forcedDrainCtx, cancel := context.WithTimeout(context.Background(), s.forcedDrainWait)
	defer cancel()

	if drainErr := s.waitForRequests(forcedDrainCtx); drainErr != nil {
		return errors.Join(shutdownErr, ErrRequestDrainTimeout, drainErr)
	}

	return shutdownErr
}

// handleHealth handles GET /healthz.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if s.health != nil {
		if checkErr2 := s.health(r.Context()); checkErr2 != nil {
			httputil.RespondError(w, http.StatusServiceUnavailable, "Artifact storage unavailable")
			return
		}
	}

	httputil.RespondOK(w, map[string]string{
		"status": "healthy",
	})
}
