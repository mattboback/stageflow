// Package api exposes the orchestrator's HTTP endpoints.
package api

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/mattboback/stageflow/libs/go/httputil"
	db "github.com/mattboback/stageflow/services/orchestrator/internal/adapters/repository"
)

const defaultForcedDrainWait = time.Second

// ErrRequestDrainTimeout means force-closing HTTP connections did not make
// every handler exit. Callers must not close handler dependencies before the
// process terminates.
var ErrRequestDrainTimeout = errors.New("orchestrator API handlers did not drain")

// Server provides private job status reads and a health endpoint.
type Server struct {
	database        *db.Database
	apiToken        string
	port            string
	server          *http.Server
	rateLimiter     *adminRateLimiter
	requestsMu      sync.Mutex
	requestsWG      sync.WaitGroup
	requestsStop    bool
	forcedDrainWait time.Duration
}

// Config contains configuration for the API server.
type Config struct {
	Database            *db.Database
	APIToken            string
	Port                string // Default: "8081"
	AdminRateLimitRPS   int    // 0 disables rate limiting.
	AdminRateLimitBurst int    // Defaults to AdminRateLimitRPS when rate limiting is enabled.
}

// NewServer creates a new API server.
func NewServer(cfg *Config) *Server {
	if cfg.Port == "" {
		cfg.Port = "8081"
	}

	s := &Server{
		database:        cfg.Database,
		apiToken:        strings.TrimSpace(cfg.APIToken),
		port:            cfg.Port,
		rateLimiter:     newAdminRateLimiter(cfg.AdminRateLimitRPS, cfg.AdminRateLimitBurst),
		forcedDrainWait: defaultForcedDrainWait,
	}

	mux := http.NewServeMux()

	// Platform-api reads individual jobs through this private endpoint.
	mux.HandleFunc("/api/v1/jobs/{job_id}", s.requireAuth(s.handleGetJob))

	mux.HandleFunc("/healthz", s.handleHealth)

	s.server = &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           s.trackRequests(s.recoverPanic(s.rateLimit(mux))),
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

func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.apiToken == "" {
			httputil.RespondError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		provided := strings.TrimSpace(r.Header.Get("X-Api-Key"))
		if provided == "" {
			auth := strings.TrimSpace(r.Header.Get("Authorization"))
			if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
				provided = strings.TrimSpace(auth[len("bearer "):])
			}
		}

		if subtle.ConstantTimeCompare([]byte(provided), []byte(s.apiToken)) != 1 {
			httputil.RespondError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		next.ServeHTTP(w, r)
	}
}

// handleGetJob handles GET /api/v1/jobs/{job_id}.
func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.RespondError(w, http.StatusMethodNotAllowed, "Method not allowed")

		return
	}

	jobID := r.PathValue("job_id")

	job, err := s.database.GetJob(r.Context(), jobID)
	if err != nil {
		slog.Warn("Job not found", "job_id", jobID, "error", err)
		httputil.RespondNotFound(w, "Job")

		return
	}

	httputil.RespondOK(w, map[string]any{
		"job": job,
	})
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	slog.Info("Starting orchestrator API server", "port", s.port)

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
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
func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	httputil.RespondOK(w, map[string]string{
		"status": "healthy",
	})
}
