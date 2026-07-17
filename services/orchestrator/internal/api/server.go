// Package api exposes the orchestrator's HTTP endpoints.
package api

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mattboback/stageflow/libs/go/httputil"
	"github.com/mattboback/stageflow/libs/go/models"
	db "github.com/mattboback/stageflow/services/orchestrator/internal/adapters/repository"
	podman "github.com/mattboback/stageflow/services/orchestrator/internal/adapters/runtime"
	"github.com/mattboback/stageflow/services/orchestrator/internal/metrics"
)

const (
	defaultJobsListLimit    = 50
	maxJobsListLimit        = 1000
	defaultJobEventsLimit   = 500
	maxJobEventsLimit       = 5000
	minPaginationLimitValue = 1
	defaultForcedDrainWait  = time.Second
)

// ErrRequestDrainTimeout means force-closing HTTP connections did not make
// every handler exit. Callers must not close handler dependencies before the
// process terminates.
var ErrRequestDrainTimeout = errors.New("orchestrator API handlers did not drain")

// Server provides HTTP API endpoints for job and pod status.
type Server struct {
	database        *db.Database
	podmanClient    PodmanClient
	apiToken        string
	port            string
	server          *http.Server
	rateLimiter     *adminRateLimiter
	metrics         *metrics.Collector
	requestsMu      sync.Mutex
	requestsWG      sync.WaitGroup
	requestsStop    bool
	forcedDrainWait time.Duration
}

// PodmanClient is the subset of Podman operations the API server needs.
type PodmanClient interface {
	ListPods(ctx context.Context) ([]podman.PodInfo, error)
	InspectPod(ctx context.Context, podID string) (*podman.PodInfo, error)
}

func parsePaginationParams(query url.Values, defaultLimit, maxLimit int) (limit, offset int, err error) {
	limit = defaultLimit

	if limitStr := query.Get("limit"); limitStr != "" {
		parsed, parseErr := strconv.Atoi(limitStr)
		if parseErr != nil || parsed < minPaginationLimitValue || parsed > maxLimit {
			return 0, 0, fmt.Errorf("limit must be an integer between %d and %d", minPaginationLimitValue, maxLimit)
		}

		limit = parsed
	}

	offset = 0

	if offsetStr := query.Get("offset"); offsetStr != "" {
		parsed, parseErr := strconv.Atoi(offsetStr)
		if parseErr != nil || parsed < 0 {
			return 0, 0, errors.New("offset must be a non-negative integer")
		}

		offset = parsed
	}

	return limit, offset, nil
}

func validJobStateFilter(state models.JobState) bool {
	for _, known := range models.AllJobStates {
		if state == known {
			return true
		}
	}

	return false
}

// Config contains configuration for the API server.
type Config struct {
	Database            *db.Database
	PodmanClient        PodmanClient
	APIToken            string
	Port                string             // Default: "8081"
	AdminRateLimitRPS   int                // 0 disables rate limiting.
	AdminRateLimitBurst int                // Defaults to AdminRateLimitRPS when rate limiting is enabled.
	Metrics             *metrics.Collector // Optional: shared collector for /metrics counters.
}

// NewServer creates a new API server.
func NewServer(cfg *Config) *Server {
	if cfg.Port == "" {
		cfg.Port = "8081"
	}

	s := &Server{
		database:        cfg.Database,
		podmanClient:    cfg.PodmanClient,
		apiToken:        strings.TrimSpace(cfg.APIToken),
		port:            cfg.Port,
		rateLimiter:     newAdminRateLimiter(cfg.AdminRateLimitRPS, cfg.AdminRateLimitBurst),
		metrics:         cfg.Metrics,
		forcedDrainWait: defaultForcedDrainWait,
	}

	mux := http.NewServeMux()

	// Admin endpoints only (public API is platform-api).
	mux.HandleFunc("/api/v1/jobs", s.requireAuth(s.handleListJobs))
	mux.HandleFunc("/api/v1/jobs/", s.requireAuth(s.handleJobRoutes))
	mux.HandleFunc("/api/v1/pods", s.requireAuth(s.handleListPods))
	mux.HandleFunc("/api/v1/pods/", s.requireAuth(s.handlePodDetails))
	mux.HandleFunc("/api/v1/status", s.requireAuth(s.handleSystemStatus))
	mux.HandleFunc("/metrics", s.requireAuth(s.handleMetrics))

	mux.HandleFunc("/healthz", s.handleHealth)

	s.server = &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           s.trackRequests(s.recoverPanic(s.recordMetrics(s.rateLimit(mux)))),
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

// handleJobRoutes handles job subroutes:
//   - GET /api/v1/jobs/{job_id}
//   - GET /api/v1/jobs/{job_id}/events
func (s *Server) handleJobRoutes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.RespondError(w, http.StatusMethodNotAllowed, "Method not allowed")

		return
	}

	trimmed := strings.TrimPrefix(r.URL.Path, "/api/v1/jobs/")
	if trimmed == "" {
		httputil.RespondError(w, http.StatusBadRequest, "Missing job ID")

		return
	}

	if strings.HasSuffix(trimmed, "/events") {
		jobID := strings.TrimSuffix(trimmed, "/events")
		jobID = strings.TrimSuffix(jobID, "/")

		if jobID == "" {
			httputil.RespondError(w, http.StatusBadRequest, "Missing job ID")

			return
		}

		limit, offset, pageErr := parsePaginationParams(r.URL.Query(), defaultJobEventsLimit, maxJobEventsLimit)
		if pageErr != nil {
			httputil.RespondError(w, http.StatusBadRequest, pageErr.Error())

			return
		}

		eventsList, err := s.database.ListJobEvents(
			r.Context(),
			jobID,
			db.ListJobEventsOptions{Limit: limit, Offset: offset},
		)
		if err != nil {
			slog.Error("Failed to list job events", "job_id", jobID, "error", err)
			httputil.RespondError(w, http.StatusInternalServerError, "Failed to retrieve job events")

			return
		}

		httputil.RespondOK(w, map[string]any{
			"job_id": jobID,
			"events": eventsList,
			"limit":  limit,
			"offset": offset,
		})

		return
	}

	// Default: GET /api/v1/jobs/{job_id}
	jobID := strings.TrimSuffix(trimmed, "/")

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

// handleListJobs handles GET /api/v1/jobs
// Query parameters:
//   - state: filter by job state (optional)
//   - limit: max number of results (default: 50)
//   - offset: skip N results (default: 0)
func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.RespondError(w, http.StatusMethodNotAllowed, "Method not allowed")

		return
	}

	query := r.URL.Query()

	var stateFilter *models.JobState

	if stateStr := query.Get("state"); stateStr != "" {
		state := models.JobState(stateStr)
		if !validJobStateFilter(state) {
			httputil.RespondError(w, http.StatusBadRequest, "Unknown job state")

			return
		}

		stateFilter = &state
	}

	limit, offset, pageErr := parsePaginationParams(query, defaultJobsListLimit, maxJobsListLimit)
	if pageErr != nil {
		httputil.RespondError(w, http.StatusBadRequest, pageErr.Error())

		return
	}

	jobs, err := s.database.ListJobs(r.Context(), db.ListJobsOptions{
		State:  stateFilter,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		slog.Error("Failed to list jobs", "error", err)
		httputil.RespondError(w, http.StatusInternalServerError, "Failed to retrieve jobs")

		return
	}

	totalCount, err := s.database.CountJobs(r.Context(), stateFilter)
	if err != nil {
		slog.Warn("Failed to count jobs", "error", err)

		totalCount = len(jobs) // fallback
	}

	response := map[string]any{
		"jobs":   jobs,
		"total":  totalCount,
		"limit":  limit,
		"offset": offset,
	}

	httputil.RespondOK(w, response)
}

// handleListPods handles GET /api/v1/pods
// Lists all pods from Podman with job associations.
func (s *Server) handleListPods(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.RespondError(w, http.StatusMethodNotAllowed, "Method not allowed")

		return
	}

	if s.podmanClient == nil {
		httputil.RespondError(w, http.StatusServiceUnavailable, "Podman client not configured")

		return
	}

	pods, err := s.podmanClient.ListPods(r.Context())
	if err != nil {
		slog.Error("Failed to list pods", "error", err)
		httputil.RespondError(w, http.StatusInternalServerError, "Failed to retrieve pods")

		return
	}

	enrichedPods := make([]map[string]any, 0, len(pods))

	for _, pod := range pods {
		podData := map[string]any{
			"id":     pod.ID,
			"name":   pod.Name,
			"status": pod.Status,
			"job_id": nil,
		}

		// Try to extract job ID from pod name (format: job-{jobID})
		if strings.HasPrefix(pod.Name, "job-") {
			jobID := strings.TrimPrefix(pod.Name, "job-")
			podData["job_id"] = jobID

			if job, getJobErr := s.database.GetJob(r.Context(), jobID); getJobErr == nil {
				podData["job_state"] = job.State
			}
		}

		enrichedPods = append(enrichedPods, podData)
	}

	httputil.RespondOK(w, map[string]any{
		"pods":  enrichedPods,
		"total": len(enrichedPods),
	})
}

// handlePodDetails handles GET /api/v1/pods/{id}
// Returns detailed pod information including containers.
func (s *Server) handlePodDetails(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.RespondError(w, http.StatusMethodNotAllowed, "Method not allowed")

		return
	}

	if s.podmanClient == nil {
		httputil.RespondError(w, http.StatusServiceUnavailable, "Podman client not configured")

		return
	}

	podID := strings.TrimPrefix(r.URL.Path, "/api/v1/pods/")
	if podID == "" {
		httputil.RespondError(w, http.StatusBadRequest, "Missing pod ID")

		return
	}

	podInfo, err := s.podmanClient.InspectPod(r.Context(), podID)
	if err != nil {
		slog.Error("Failed to inspect pod", "error", err, "pod_id", podID)
		httputil.RespondNotFound(w, "Pod")

		return
	}

	httputil.RespondOK(w, podInfo)
}

// handleSystemStatus handles GET /api/v1/status
// Returns overall system status and statistics.
func (s *Server) handleSystemStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.RespondError(w, http.StatusMethodNotAllowed, "Method not allowed")

		return
	}

	stateCounts := make(map[string]int)

	for _, state := range []models.JobState{
		models.JobStatePending,
		models.JobStateExtracting,
		models.JobStateReady,
		models.JobStateScanning,
		models.JobStateCompleting,
		models.JobStateDone,
		models.JobStateFailed,
	} {
		count, err := s.database.CountJobs(r.Context(), &state)
		if err != nil {
			slog.Warn("Failed to count jobs for state", "state", state, "error", err)

			count = 0
		}

		stateCounts[string(state)] = count
	}

	totalJobs, err := s.database.CountJobs(r.Context(), nil)
	if err != nil {
		slog.Warn("Failed to count total jobs", "error", err)

		totalJobs = 0
	}

	pods := []podman.PodInfo{}

	if s.podmanClient != nil {
		podList, listErr := s.podmanClient.ListPods(r.Context())
		if listErr != nil {
			slog.Warn("Failed to list pods", "error", listErr)
		} else {
			pods = podList
		}
	}

	podStatusCounts := make(map[string]int)
	for _, pod := range pods {
		podStatusCounts[pod.Status]++
	}

	response := map[string]any{
		"jobs": map[string]any{
			"total":     totalJobs,
			"by_state":  stateCounts,
			"active":    stateCounts["PENDING"] + stateCounts["EXTRACTING"] + stateCounts["READY_TO_SCAN"] + stateCounts["SCANNING"] + stateCounts["COMPLETING"],
			"completed": stateCounts["DONE"],
			"failed":    stateCounts["FAILED"],
		},
		"pods": map[string]any{
			"total":     len(pods),
			"by_status": podStatusCounts,
		},
	}

	httputil.RespondOK(w, response)
}

// handleHealth handles GET /healthz.
func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	httputil.RespondOK(w, map[string]string{
		"status": "healthy",
	})
}
