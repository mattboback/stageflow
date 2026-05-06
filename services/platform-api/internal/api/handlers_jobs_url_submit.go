package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/httputil"
	"github.com/mattboback/stageflow/libs/go/logging"
	"github.com/mattboback/stageflow/libs/go/models"
	"github.com/mattboback/stageflow/services/platform-api/internal/jobstatus"
)

const (
	maxURLSubmitBodySize = 2 * 1024 * 1024 // 2 MB
	maxURLCount          = 100
	maxURLLength         = 2048
	defaultScreenshot    = true
)

type jobURLSubmitRequest struct {
	URLs                []string                  `json:"urls"`
	Modules             []string                  `json:"modules"`
	ScannerConfigs      map[string]map[string]any `json:"scanner_configs,omitempty"`
	Screenshot          *bool                     `json:"screenshot"`
	HighlightStyle      string                    `json:"highlight_style"`
	AllowPrivateTargets bool                      `json:"allow_private_targets"`
}

func (s *Server) handleJobURLSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxURLSubmitBodySize)

	var req jobURLSubmitRequest

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

	if detail := validateURLSubmitRequest(req.URLs); detail != nil {
		httputil.RespondStructuredError(w, http.StatusBadRequest, *detail)

		return
	}

	validationMode, detail := s.resolveTargetValidationMode(req.AllowPrivateTargets)
	if detail != nil {
		httputil.RespondStructuredError(w, http.StatusBadRequest, *detail)

		return
	}

	if err := validateTargetURLsWithResolver(r.Context(), s.ipResolver, req.URLs, validationMode); err != nil {
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

	if scannerConfigDetail := validateScannerConfigs(modules, req.ScannerConfigs); scannerConfigDetail != nil {
		httputil.RespondStructuredError(w, http.StatusBadRequest, *scannerConfigDetail)

		return
	}

	highlightStyle := normalizeHighlightStyle(req.HighlightStyle)
	screenshot := defaultScreenshot
	if req.Screenshot != nil {
		screenshot = *req.Screenshot
	}

	jobID := uuid.New().String()
	ctx := logging.WithJobID(r.Context(), jobID)
	runID := uuid.New().String()[:8]
	ctx = logging.WithRunID(ctx, runID)

	payload := &events.JobCreatedPayload{
		JobID:     jobID,
		InputType: "urls",
		URLs:      req.URLs,
		Config: models.JobConfig{
			Modules:             modules,
			ScannerConfigs:      req.ScannerConfigs,
			Screenshot:          screenshot,
			HighlightStyle:      highlightStyle,
			AllowPrivateTargets: req.AllowPrivateTargets,
		},
	}

	envelope := events.NewEnvelope(events.EventJobCreated, jobID, "platform-api", payload)
	envelope.RequestID = logging.RequestID(ctx)
	envelope.RunID = logging.RunID(ctx)

	if ctxErr := ctx.Err(); ctxErr != nil {
		if errors.Is(ctxErr, context.DeadlineExceeded) {
			httputil.RespondError(w, http.StatusServiceUnavailable, "Request timeout")
		} else {
			httputil.RespondError(w, http.StatusRequestTimeout, "Request canceled")
		}

		return
	}

	if publishErr := s.config.Publisher.PublishJobCreated(ctx, envelope); publishErr != nil {
		logging.Error(ctx, "Failed to publish job.created event", "error", publishErr)
		httputil.RespondError(w, http.StatusInternalServerError, "Failed to queue job")

		return
	}

	if _, beginErr := s.jobStatus.Begin(ctx, jobstatus.BeginJob{
		Payload:    payload,
		ObservedAt: envelope.Timestamp,
	}); beginErr != nil {
		logging.Warn(ctx, "Failed to seed provisional job status", "error", beginErr)
	}

	logging.Info(ctx, "Job created", "url_count", len(req.URLs), "input_type", "urls")

	httputil.RespondCreated(w, map[string]any{
		"job_id":  jobID,
		"status":  "pending",
		"message": "Job created successfully",
	})
}

func validateURLSubmitRequest(urls []string) *httputil.ErrorDetail {
	if len(urls) == 0 {
		detail := httputil.NewValidationError(
			"urls",
			"URLs list cannot be empty",
			"Please provide at least one URL to scan. Example: ['https://example.com']",
		)

		return &detail
	}

	if len(urls) > maxURLCount {
		detail := httputil.NewValidationError(
			"urls",
			fmt.Sprintf("Too many URLs (max %d)", maxURLCount),
			fmt.Sprintf("Please provide at most %d URLs per job.", maxURLCount),
		)

		return &detail
	}

	for _, raw := range urls {
		trimmed := strings.TrimSpace(raw)
		if len(trimmed) > maxURLLength {
			detail := httputil.NewValidationError(
				"urls",
				fmt.Sprintf("URL exceeds maximum length of %d characters", maxURLLength),
				"Shorten the URL or split the job into smaller batches.",
			)

			return &detail
		}
	}

	return nil
}

func (s *Server) resolveTargetValidationMode(
	allowPrivateTargets bool,
) (targetValidationMode, *httputil.ErrorDetail) {
	if !allowPrivateTargets {
		return targetValidationModePublic, nil
	}

	if !s.config.AllowPrivateTargets {
		detail := httputil.NewValidationError(
			"allow_private_targets",
			"This API instance does not permit private target scans",
			"Retry without allow_private_targets or enable private target scans on this API instance.",
		)

		return targetValidationModePublic, &detail
	}

	return targetValidationModePrivate, nil
}
