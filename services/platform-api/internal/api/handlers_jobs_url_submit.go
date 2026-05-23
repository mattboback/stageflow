package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/httputil"
	"github.com/mattboback/stageflow/libs/go/logging"
	"github.com/mattboback/stageflow/libs/go/models"
	"github.com/mattboback/stageflow/services/platform-api/internal/jobstatus"
)

const (
	maxURLSubmitBodySize  = 2 * 1024 * 1024 // 2 MB
	maxURLCount           = 100
	maxURLLength          = 2048
	defaultScreenshot     = true
	maxAuthStateBytes     = 1 << 20 // 1 MiB cap on storage-state payload
	authModeForm          = "form"
	authModeStorageState  = "storage_state"
)

var authFromEnvNamePattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

type jobURLSubmitRequest struct {
	URLs                []string                  `json:"urls"`
	Modules             []string                  `json:"modules"`
	ScannerConfigs      map[string]map[string]any `json:"scanner_configs,omitempty"`
	Screenshot          *bool                     `json:"screenshot"`
	HighlightStyle      string                    `json:"highlight_style"`
	AllowPrivateTargets bool                      `json:"allow_private_targets"`
	Auth                *jobURLAuthInput          `json:"auth,omitempty"`
}

// jobURLAuthInput mirrors the wire shape the CLI ships in
// SubmitJobRequest.Auth. Discriminated by mode.
//
// For storage_state the CLI base64-encodes the captured Playwright JSON
// inline; the platform-api validates it parses + enforces a 1 MiB cap, then
// forwards on as-is in JobConfig.Auth. The orchestrator uploads the bytes to
// MinIO under the job's prefix and rewrites JobConfig.Auth so only the
// artifact_key is persisted in Postgres.
//
// For form, values are either literal strings or {from_env: NAME} references.
// The platform-api does NOT resolve from_env (it has no business doing so); it
// forwards the recipe untouched and the orchestrator builds the env-var
// allow-list at scanner-launch time.
type jobURLAuthInput struct {
	Mode         string                       `json:"mode"`
	StorageState *jobURLAuthStorageStateInput `json:"storage_state,omitempty"`
	Form         *jobURLAuthFormRecipe        `json:"form,omitempty"`
}

type jobURLAuthStorageStateInput struct {
	ContentBase64 string `json:"content_b64"`
}

type jobURLAuthFormRecipe struct {
	LoginURL string           `json:"login_url"`
	Steps    []map[string]any `json:"steps"`
	Success  map[string]any   `json:"success"`
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

	authRaw, authErr := normalizeJobURLAuth(req.Auth)
	if authErr != nil {
		httputil.RespondError(w, http.StatusBadRequest, authErr.Error())

		return
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
			Auth:                authRaw,
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


// normalizeJobURLAuth validates the optional auth block carried in the URL
// submit request and returns its canonical JSON form (Provenance.auth shape)
// for storage in JobConfig.Auth.
//
// The platform-api enforces structure but never resolves credentials: form
// recipes round-trip with from_env references intact, and storage_state keeps
// the inline content_b64 blob for the orchestrator to upload to MinIO before
// the job is persisted.
func normalizeJobURLAuth(in *jobURLAuthInput) (json.RawMessage, error) {
	if in == nil {
		return nil, nil
	}

	switch in.Mode {
	case authModeStorageState:
		if in.Form != nil {
			return nil, errors.New(`auth.form is only valid with mode="form"`)
		}

		if in.StorageState == nil || strings.TrimSpace(in.StorageState.ContentBase64) == "" {
			return nil, errors.New(`auth.storage_state.content_b64 is required when mode="storage_state"`)
		}

		decoded, err := base64.StdEncoding.DecodeString(in.StorageState.ContentBase64)
		if err != nil {
			return nil, fmt.Errorf("auth.storage_state.content_b64 is not valid base64: %w", err)
		}

		if len(decoded) == 0 {
			return nil, errors.New("auth.storage_state.content_b64 decodes to empty bytes")
		}

		if len(decoded) > maxAuthStateBytes {
			return nil, fmt.Errorf(
				"auth.storage_state.content_b64 decodes to %d bytes, exceeding the %d byte limit",
				len(decoded),
				maxAuthStateBytes,
			)
		}

		var probe map[string]any
		if err := json.Unmarshal(decoded, &probe); err != nil {
			return nil, fmt.Errorf("auth.storage_state.content_b64 is not valid JSON: %w", err)
		}

		// Re-marshal as a stable JSON shape; the orchestrator decodes this from
		// JobConfig.Auth before persistence and uploads.
		out := map[string]any{
			"mode":        authModeStorageState,
			"content_b64": in.StorageState.ContentBase64,
		}

		raw, err := json.Marshal(out)
		if err != nil {
			return nil, fmt.Errorf("auth.storage_state: failed to re-marshal: %w", err)
		}

		return raw, nil

	case authModeForm:
		if in.StorageState != nil {
			return nil, errors.New(`auth.storage_state is only valid with mode="storage_state"`)
		}

		if in.Form == nil {
			return nil, errors.New(`auth.form is required when mode="form"`)
		}

		if strings.TrimSpace(in.Form.LoginURL) == "" {
			return nil, errors.New("auth.form.login_url is required")
		}

		if len(in.Form.Steps) == 0 {
			return nil, errors.New("auth.form.steps must contain at least one step")
		}

		if in.Form.Success == nil {
			return nil, errors.New("auth.form.success is required")
		}

		if _, hasType := in.Form.Success["type"].(string); !hasType {
			return nil, errors.New("auth.form.success.type is required")
		}

		for i, step := range in.Form.Steps {
			if validateErr := validateAuthStep(step); validateErr != nil {
				return nil, fmt.Errorf("auth.form.steps[%d]: %w", i, validateErr)
			}
		}

		out := map[string]any{
			"mode":      authModeForm,
			"login_url": in.Form.LoginURL,
			"steps":     in.Form.Steps,
			"success":   in.Form.Success,
		}

		raw, err := json.Marshal(out)
		if err != nil {
			return nil, fmt.Errorf("auth.form: failed to re-marshal: %w", err)
		}

		return raw, nil

	case "":
		return nil, errors.New("auth.mode is required")

	default:
		return nil, fmt.Errorf("auth.mode %q is not supported (expected %q or %q)",
			in.Mode, authModeForm, authModeStorageState)
	}
}

// validateAuthStep mirrors the schema-level validation for PreScanAction
// without bringing in the full ajv pipeline. We only need to catch shapes the
// orchestrator can't safely process; the scanner-runner re-validates at
// hydration time.
func validateAuthStep(step map[string]any) error {
	t, _ := step["type"].(string)
	if t == "" {
		return errors.New("step.type is required")
	}

	switch t {
	case "click", "fill", "select", "hover":
		if _, hasSel := step["selector"].(string); !hasSel {
			return fmt.Errorf("step type %q requires a selector", t)
		}
	case "wait":
		if _, hasMS := step["ms"]; !hasMS {
			return errors.New(`step type "wait" requires "ms"`)
		}
	case "scroll":
		// selector optional
	case "keyboard":
		if _, hasKey := step["key"].(string); !hasKey {
			return errors.New(`step type "keyboard" requires "key"`)
		}
	default:
		return fmt.Errorf("unknown step.type %q", t)
	}

	if t == "fill" || t == "select" {
		raw, hasValue := step["value"]
		if !hasValue {
			return fmt.Errorf("step type %q requires a value", t)
		}

		if err := validateAuthStepValue(raw); err != nil {
			return err
		}
	}

	return nil
}

func validateAuthStepValue(raw any) error {
	switch v := raw.(type) {
	case string:
		return nil
	case map[string]any:
		name, ok := v["from_env"].(string)
		if !ok {
			return errors.New(`step.value object must be {from_env: NAME}`)
		}

		if !authFromEnvNamePattern.MatchString(name) {
			return fmt.Errorf("step.value.from_env %q must match %s", name, authFromEnvNamePattern.String())
		}

		for k := range v {
			if k != "from_env" {
				return fmt.Errorf(`step.value object has unexpected key %q (only "from_env" is allowed)`, k)
			}
		}

		return nil
	default:
		return fmt.Errorf("step.value must be a string or {from_env: NAME} object (got %T)", raw)
	}
}
