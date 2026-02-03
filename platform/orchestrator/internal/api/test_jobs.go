package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/mattboback/stageflow/packages/shared-go/events"
	"github.com/mattboback/stageflow/packages/shared-go/httputil"
	"github.com/mattboback/stageflow/packages/shared-go/models"
)

// CreateTestJobRequest defines the payload for POST /api/v1/test/jobs.
type CreateTestJobRequest struct {
	URLs    []string `json:"urls"`    // For URL-based scans
	Modules []string `json:"modules"` // Scanner modules to run (e.g., ["axe", "lighthouse"])
}

// handleCreateTestJob handles POST /api/v1/test/jobs.
// Creates a URL scan job directly for testing purposes.
func (s *Server) handleCreateTestJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.RespondError(w, http.StatusMethodNotAllowed, "Method not allowed")

		return
	}

	var req CreateTestJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "Invalid request body")

		return
	}

	if len(req.URLs) == 0 {
		httputil.RespondError(w, http.StatusBadRequest, "At least one URL is required")

		return
	}

	// Default to axe scanner if no modules specified
	if len(req.Modules) == 0 {
		req.Modules = []string{"axe"}
	}

	jobID := uuid.New().String()

	// Don't create the job in the database here - let the orchestrator do it when it handles the job.created event.
	// Publishing the event is enough to trigger job processing.

	// Publish job.created event to NATS so orchestrator picks it up and creates the job
	if s.publisher == nil {
		slog.Error("Publisher not configured")
		httputil.RespondError(w, http.StatusInternalServerError, "Job queue not available")

		return
	}

	payload := &events.JobCreatedPayload{
		JobID:     jobID,
		InputType: "urls",
		URLs:      req.URLs,
		Config: models.JobConfig{
			Modules: req.Modules,
		},
	}

	if err := s.publisher.PublishJobCreated(r.Context(), payload); err != nil {
		slog.Error("Failed to publish job.created event", "error", err, "job_id", jobID)
		httputil.RespondError(w, http.StatusInternalServerError, "Failed to queue job")

		return
	}

	slog.Info("Created test job via API", "job_id", jobID, "urls", req.URLs, "modules", req.Modules)

	httputil.RespondOK(w, map[string]any{
		"job_id":  jobID,
		"state":   models.JobStatePending,
		"urls":    req.URLs,
		"modules": req.Modules,
		"message": "Job created. Use GET /api/v1/jobs to check status.",
	})
}
