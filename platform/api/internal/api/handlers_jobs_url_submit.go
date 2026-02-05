package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/mattboback/stageflow/packages/shared-go/events"
	"github.com/mattboback/stageflow/packages/shared-go/httputil"
	"github.com/mattboback/stageflow/packages/shared-go/logging"
	"github.com/mattboback/stageflow/packages/shared-go/models"
)

const (
	maxURLSubmitBodySize = 2 * 1024 * 1024 // 2 MB
	maxURLCount          = 100
	maxURLLength         = 2048
)

func (s *Server) handleJobURLSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxURLSubmitBodySize)

	var req struct {
		URLs           []string                  `json:"urls"`
		Modules        []string                  `json:"modules"`
		ScannerConfigs map[string]map[string]any `json:"scanner_configs,omitempty"`
		Screenshot     bool                      `json:"screenshot"`
		HighlightStyle string                    `json:"highlight_style"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			httputil.RespondError(w, http.StatusRequestEntityTooLarge,
				fmt.Sprintf("Request body too large (max %d bytes)", maxURLSubmitBodySize))

			return
		}

		httputil.RespondError(w, http.StatusBadRequest, "Invalid JSON body")

		return
	}

	if len(req.URLs) == 0 {
		httputil.RespondStructuredError(w, http.StatusBadRequest, httputil.NewValidationError(
			"urls",
			"URLs list cannot be empty",
			"Please provide at least one URL to scan. Example: ['https://example.com']",
		))

		return
	}

	if len(req.URLs) > maxURLCount {
		httputil.RespondStructuredError(w, http.StatusBadRequest, httputil.NewValidationError(
			"urls",
			fmt.Sprintf("Too many URLs (max %d)", maxURLCount),
			fmt.Sprintf("Please provide at most %d URLs per job.", maxURLCount),
		))

		return
	}

	for _, raw := range req.URLs {
		trimmed := strings.TrimSpace(raw)
		if len(trimmed) > maxURLLength {
			httputil.RespondStructuredError(w, http.StatusBadRequest, httputil.NewValidationError(
				"urls",
				fmt.Sprintf("URL exceeds maximum length of %d characters", maxURLLength),
				"Shorten the URL or split the job into smaller batches.",
			))

			return
		}
	}

	if err := validateTargetURLsWithResolver(r.Context(), s.ipResolver, req.URLs); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, err.Error())

		return
	}

	modules, err := s.normalizeModules(req.Modules)
	if err != nil {
		if strings.Contains(err.Error(), "unsupported scanner module") {
			supported := s.listSupportedModuleIDs()

			httputil.RespondStructuredError(w, http.StatusBadRequest,
				httputil.NewUnsupportedModuleError(extractModuleName(err.Error()), supported))
		} else {
			httputil.RespondError(w, http.StatusBadRequest, err.Error())
		}

		return
	}

	if detail := validateScannerConfigs(modules, req.ScannerConfigs); detail != nil {
		httputil.RespondStructuredError(w, http.StatusBadRequest, *detail)

		return
	}

	highlightStyle := normalizeHighlightStyle(req.HighlightStyle)

	jobID := uuid.New().String()
	ctx := logging.WithJobID(r.Context(), jobID)
	runID := uuid.New().String()[:8]
	ctx = logging.WithRunID(ctx, runID)

	payload := &events.JobCreatedPayload{
		JobID:     jobID,
		InputType: "urls",
		URLs:      req.URLs,
		Config: models.JobConfig{
			Modules:        modules,
			ScannerConfigs: req.ScannerConfigs,
			Screenshot:     req.Screenshot,
			HighlightStyle: highlightStyle,
		},
	}

	if persistErr := s.statusStore.HandleJobCreated(ctx, payload); persistErr != nil {
		logging.Error(ctx, "Failed to persist job status", "error", persistErr)
		httputil.RespondError(w, http.StatusInternalServerError, "Failed to persist job")

		return
	}

	envelope := events.NewEnvelope(events.EventJobCreated, jobID, "platform-api", payload)
	envelope.RequestID = logging.RequestID(ctx)
	envelope.RunID = logging.RunID(ctx)

	if publishErr := s.config.Publisher.PublishJobCreated(ctx, envelope); publishErr != nil {
		logging.Error(ctx, "Failed to publish job.created event", "error", publishErr)
		httputil.RespondError(w, http.StatusInternalServerError, "Failed to queue job")

		return
	}

	logging.Info(ctx, "Job created", "url_count", len(req.URLs), "input_type", "urls")

	httputil.RespondCreated(w, map[string]any{
		"job_id":  jobID,
		"status":  "pending",
		"message": "Job created successfully",
	})
}
