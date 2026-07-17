package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/httputil"
	"github.com/mattboback/stageflow/libs/go/logging"
	"github.com/mattboback/stageflow/libs/go/models"
	sharedstorage "github.com/mattboback/stageflow/libs/go/storage"
	"github.com/mattboback/stageflow/services/platform-api/internal/jobstatus"
)

const (
	maxURLSubmitBodySize = 2 * 1024 * 1024 // 2 MB
	maxURLCount          = 100
	maxURLLength         = 2048
	defaultScreenshot    = true
	maxAuthStateBytes    = 1 << 20 // 1 MiB cap on storage-state payload
	authModeForm         = "form"
	authModeStorageState = "storage_state"
)

var authFromEnvNamePattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

var errAuthStorageUnavailable = errors.New("auth storage backend unavailable")

type jobURLSubmitRequest struct {
	URLs                []string                  `json:"urls"`
	Modules             []string                  `json:"modules"`
	ScannerConfigs      map[string]map[string]any `json:"scanner_configs,omitempty"`
	Screenshot          *bool                     `json:"screenshot"`
	HighlightStyle      string                    `json:"highlight_style"`
	Browser             string                    `json:"browser"`
	AllowPrivateTargets bool                      `json:"allow_private_targets"`
	Auth                *jobURLAuthInput          `json:"auth,omitempty"`
}

// jobURLAuthInput mirrors the wire shape the CLI ships in
// SubmitJobRequest.Auth. Discriminated by mode.
//
// For storage_state the CLI base64-encodes the captured Playwright JSON
// inline; the platform-api validates it parses + enforces a 1 MiB cap, uploads
// it to MinIO under the job's prefix, then forwards only an artifact_key in
// JobConfig.Auth. Raw storage-state bytes must not cross the NATS boundary.
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

type jobURLSubmitPolicy uint8

const (
	jobURLSubmitPolicyAuthenticated jobURLSubmitPolicy = iota
	jobURLSubmitPolicyAnonymous
	jobURLSubmitPolicyBrowserAuth
)

func (s *Server) handleJobURLSubmit(w http.ResponseWriter, r *http.Request) {
	s.handleJobURLSubmitWithPolicy(w, r, jobURLSubmitPolicyAuthenticated)
}

// handleAnonymousJobURLSubmit serves public browser scans that do not need a
// login. The trusted edge authenticates this exact route with its server-side
// token, so it must reject every auth recipe and private-network opt-in.
func (s *Server) handleAnonymousJobURLSubmit(w http.ResponseWriter, r *http.Request) {
	s.handleJobURLSubmitWithPolicy(w, r, jobURLSubmitPolicyAnonymous)
}

// handleBrowserAuthJobURLSubmit is the deliberately narrow authenticated-page
// browser flow. It accepts only literal form-fill values; storage state and
// from_env references remain available solely through the caller-authenticated
// URL endpoint.
func (s *Server) handleBrowserAuthJobURLSubmit(w http.ResponseWriter, r *http.Request) {
	s.handleJobURLSubmitWithPolicy(w, r, jobURLSubmitPolicyBrowserAuth)
}

func (s *Server) handleJobURLSubmitWithPolicy(
	w http.ResponseWriter,
	r *http.Request,
	policy jobURLSubmitPolicy,
) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

		return
	}

	req, ok := decodeJobURLSubmitRequest(w, r)
	if !ok {
		return
	}

	if detail := validateJobURLSubmitPolicy(req, policy); detail != nil {
		httputil.RespondStructuredError(w, http.StatusBadRequest, *detail)

		return
	}

	validationMode, ok := s.validateJobURLSubmitRequest(w, r, req)
	if !ok {
		return
	}

	modules, ok := s.normalizeURLSubmitModules(w, req)
	if !ok {
		return
	}

	highlightStyle := normalizeHighlightStyle(req.HighlightStyle)
	browserEngine := normalizeBrowserEngine(req.Browser)
	screenshot := defaultScreenshot

	if req.Screenshot != nil {
		screenshot = *req.Screenshot
	}

	jobID := uuid.New().String()
	ctx := logging.WithJobID(r.Context(), jobID)
	runID := uuid.New().String()[:8]
	ctx = logging.WithRunID(ctx, runID)

	authRaw, authStorageKey, ok := s.normalizeJobURLSubmitAuth(ctx, w, jobID, req.Auth, validationMode)
	if !ok {
		return
	}

	payload := &events.JobCreatedPayload{
		JobID:     jobID,
		InputType: "urls",
		URLs:      req.URLs,
		Config: models.JobConfig{
			Modules:             modules,
			ScannerConfigs:      req.ScannerConfigs,
			Screenshot:          screenshot,
			HighlightStyle:      highlightStyle,
			Browser:             browserEngine,
			AllowPrivateTargets: req.AllowPrivateTargets,
			Auth:                authRaw,
		},
	}

	envelope := events.NewEnvelope(events.EventJobCreated, jobID, "platform-api", payload)
	envelope.RequestID = logging.RequestID(ctx)
	envelope.RunID = logging.RunID(ctx)

	if !s.publishJobURLCreated(ctx, w, envelope, authStorageKey) {
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

func validateJobURLSubmitPolicy(
	req jobURLSubmitRequest,
	policy jobURLSubmitPolicy,
) *httputil.ErrorDetail {
	if policy == jobURLSubmitPolicyAuthenticated {
		return nil
	}

	if req.AllowPrivateTargets {
		detail := httputil.NewValidationError(
			"allow_private_targets",
			"Public browser scan endpoints do not permit private target scans",
			"Use a public http/https target or submit through the caller-authenticated URL endpoint.",
		)

		return &detail
	}

	if policy == jobURLSubmitPolicyAnonymous {
		if req.Auth == nil {
			return nil
		}

		detail := httputil.NewValidationError(
			"auth",
			"Authentication recipes are not accepted by the anonymous scan endpoint",
			"Remove auth or use the browser-auth endpoint for a literal form login.",
		)

		return &detail
	}

	if req.Auth == nil {
		detail := httputil.NewValidationError(
			"auth",
			"A form authentication recipe is required by the browser-auth endpoint",
			"Provide auth.mode=form and a form recipe, or use the anonymous endpoint.",
		)

		return &detail
	}

	if req.Auth.Mode != authModeForm {
		detail := httputil.NewValidationError(
			"auth.mode",
			`The browser-auth endpoint accepts only auth.mode="form"`,
			"Use the caller-authenticated URL endpoint for storage state or environment references.",
		)

		return &detail
	}

	if req.Auth.StorageState != nil {
		detail := httputil.NewValidationError(
			"auth.storage_state",
			"Storage state is not accepted by the browser-auth endpoint",
			"Use the caller-authenticated URL endpoint to submit storage state.",
		)

		return &detail
	}

	if req.Auth.Form == nil {
		detail := httputil.NewValidationError(
			"auth.form",
			`auth.form is required when auth.mode="form"`,
			"Provide a login URL, form steps, and success condition.",
		)

		return &detail
	}

	for i, step := range req.Auth.Form.Steps {
		value, hasValue := step["value"]
		if !hasValue {
			continue
		}

		if _, literal := value.(string); literal {
			continue
		}

		detail := httputil.NewValidationError(
			fmt.Sprintf("auth.form.steps[%d].value", i),
			"The browser-auth endpoint accepts only literal form values",
			"Use a literal string here or submit environment references through the caller-authenticated URL endpoint.",
		)

		return &detail
	}

	return nil
}

func decodeJobURLSubmitRequest(w http.ResponseWriter, r *http.Request) (jobURLSubmitRequest, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxURLSubmitBodySize)

	var req jobURLSubmitRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			httputil.RespondError(w, http.StatusRequestEntityTooLarge,
				fmt.Sprintf("Request body too large (max %d bytes)", maxURLSubmitBodySize))

			return jobURLSubmitRequest{}, false
		}

		httputil.RespondError(w, http.StatusBadRequest, "Invalid JSON body")

		return jobURLSubmitRequest{}, false
	}

	return req, true
}

func (s *Server) validateJobURLSubmitRequest(
	w http.ResponseWriter,
	r *http.Request,
	req jobURLSubmitRequest,
) (targetValidationMode, bool) {
	if detail := validateURLSubmitRequest(req.URLs); detail != nil {
		httputil.RespondStructuredError(w, http.StatusBadRequest, *detail)

		return targetValidationModePublic, false
	}

	validationMode, detail := s.resolveTargetValidationMode(req.AllowPrivateTargets)
	if detail != nil {
		httputil.RespondStructuredError(w, http.StatusBadRequest, *detail)

		return targetValidationModePublic, false
	}

	if err := validateTargetURLsWithResolver(r.Context(), s.ipResolver, req.URLs, validationMode); err != nil {
		detail := httputil.NewValidationError(
			"urls",
			"One or more target URLs are invalid or not allowed",
			"Use public http/https URLs without embedded credentials.",
		)
		httputil.RespondStructuredError(w, http.StatusBadRequest, detail)

		return targetValidationModePublic, false
	}

	return validationMode, true
}

func (s *Server) normalizeURLSubmitModules(w http.ResponseWriter, req jobURLSubmitRequest) ([]string, bool) {
	modules, err := s.normalizeModules(req.Modules)
	if err != nil {
		if strings.Contains(err.Error(), "unsupported scanner module") {
			supported := s.listSupportedModuleIDs()

			httputil.RespondStructuredError(w, http.StatusBadRequest,
				httputil.NewUnsupportedModuleError(extractModuleName(err.Error()), supported))
		} else {
			httputil.RespondError(w, http.StatusBadRequest, err.Error())
		}

		return nil, false
	}

	if scannerConfigDetail := validateScannerConfigs(modules, req.ScannerConfigs); scannerConfigDetail != nil {
		httputil.RespondStructuredError(w, http.StatusBadRequest, *scannerConfigDetail)

		return nil, false
	}

	return modules, true
}

func (s *Server) normalizeJobURLSubmitAuth(
	ctx context.Context,
	w http.ResponseWriter,
	jobID string,
	in *jobURLAuthInput,
	validationMode targetValidationMode,
) (json.RawMessage, string, bool) {
	authRaw, authStorageKey, authErr := s.normalizeAndStoreJobURLAuth(ctx, jobID, in, validationMode)
	if authErr != nil {
		if errors.Is(authErr, context.DeadlineExceeded) {
			httputil.RespondError(w, http.StatusServiceUnavailable, "Request timeout")

			return nil, "", false
		}

		if errors.Is(authErr, context.Canceled) {
			httputil.RespondError(w, http.StatusRequestTimeout, "Request canceled")

			return nil, "", false
		}

		if errors.Is(authErr, errAuthStorageUnavailable) {
			logging.Error(ctx, "Failed to store auth state", "error", authErr)
			httputil.RespondError(w, http.StatusInternalServerError, "Failed to store auth state")

			return nil, "", false
		}

		// Auth validation errors can contain browser URLs or values supplied in a
		// recipe. Keep the response structured and actionable without reflecting
		// execution-only data back to clients or access logs.
		detail := httputil.NewValidationError(
			"auth",
			"Authentication configuration is invalid",
			"Review the form login URL, steps, and success condition, then retry.",
		)
		httputil.RespondStructuredError(w, http.StatusBadRequest, detail)

		return nil, "", false
	}

	return authRaw, authStorageKey, true
}

func (s *Server) publishJobURLCreated(
	ctx context.Context,
	w http.ResponseWriter,
	envelope *events.Envelope,
	authStorageKey string,
) bool {
	if ctxErr := ctx.Err(); ctxErr != nil {
		s.cleanupUploadedAuthState(authStorageKey)

		if errors.Is(ctxErr, context.DeadlineExceeded) {
			httputil.RespondError(w, http.StatusServiceUnavailable, "Request timeout")
		} else {
			httputil.RespondError(w, http.StatusRequestTimeout, "Request canceled")
		}

		return false
	}

	if publishErr := s.config.Publisher.PublishJobCreated(ctx, envelope); publishErr != nil {
		s.cleanupUploadedAuthState(authStorageKey)
		logging.Error(ctx, "Failed to publish job.created event", "error", publishErr)
		httputil.RespondError(w, http.StatusInternalServerError, "Failed to queue job")

		return false
	}

	return true
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

func (s *Server) normalizeAndStoreJobURLAuth(
	ctx context.Context,
	jobID string,
	in *jobURLAuthInput,
	validationMode targetValidationMode,
) (json.RawMessage, string, error) {
	raw, decodedStorageState, err := normalizeJobURLAuth(ctx, s.ipResolver, validationMode, in)
	if err != nil || decodedStorageState == nil {
		return raw, "", err
	}

	if s == nil || s.config == nil || s.config.Storage == nil {
		return nil, "", errAuthStorageUnavailable
	}

	key := jobID + "/auth/storage-state.json"
	if uploadErr := s.config.Storage.UploadFile(
		ctx,
		sharedstorage.BucketArtifacts,
		key,
		bytes.NewReader(decodedStorageState),
		int64(len(decodedStorageState)),
	); uploadErr != nil {
		return nil, "", fmt.Errorf("%w: %w", errAuthStorageUnavailable, uploadErr)
	}

	out := map[string]any{
		"mode":         authModeStorageState,
		"artifact_key": key,
	}

	normalized, marshalErr := json.Marshal(out)
	if marshalErr != nil {
		s.cleanupUploadedAuthState(key)

		return nil, "", fmt.Errorf("auth.storage_state: failed to re-marshal: %w", marshalErr)
	}

	return normalized, key, nil
}

func (s *Server) cleanupUploadedAuthState(key string) {
	if key == "" || s == nil || s.config == nil || s.config.Storage == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.config.Storage.DeleteFile(ctx, sharedstorage.BucketArtifacts, key); err != nil {
		logging.Warn(ctx, "Failed to clean up uploaded auth state", "key", key, "error", err)
	}
}

// normalizeJobURLAuth validates the optional auth block carried in the URL
// submit request and returns its canonical JSON form (Provenance.auth shape)
// for storage in JobConfig.Auth.
//
// The platform-api enforces structure but never resolves credentials: form
// recipes round-trip with from_env references intact, and storage_state returns
// decoded bytes to the handler so it can upload them before publishing the
// job.created event.
func normalizeJobURLAuth(
	ctx context.Context,
	resolver ipAddrResolver,
	validationMode targetValidationMode,
	in *jobURLAuthInput,
) (json.RawMessage, []byte, error) {
	if in == nil {
		return nil, nil, nil
	}

	switch in.Mode {
	case authModeStorageState:
		return normalizeJobURLStorageStateAuth(in)

	case authModeForm:
		return normalizeJobURLFormAuth(ctx, resolver, validationMode, in)

	case "":
		return nil, nil, errors.New("auth.mode is required")

	default:
		return nil, nil, fmt.Errorf("auth.mode %q is not supported (expected %q or %q)",
			in.Mode, authModeForm, authModeStorageState)
	}
}

func normalizeJobURLStorageStateAuth(in *jobURLAuthInput) (json.RawMessage, []byte, error) {
	if in.Form != nil {
		return nil, nil, errors.New(`auth.form is only valid with mode="form"`)
	}

	if in.StorageState == nil || strings.TrimSpace(in.StorageState.ContentBase64) == "" {
		return nil, nil, errors.New(`auth.storage_state.content_b64 is required when mode="storage_state"`)
	}

	decoded, err := base64.StdEncoding.DecodeString(in.StorageState.ContentBase64)
	if err != nil {
		return nil, nil, fmt.Errorf("auth.storage_state.content_b64 is not valid base64: %w", err)
	}

	if validateErr := validateDecodedStorageState(decoded); validateErr != nil {
		return nil, nil, validateErr
	}

	return nil, decoded, nil
}

func validateDecodedStorageState(decoded []byte) error {
	if len(decoded) == 0 {
		return errors.New("auth.storage_state.content_b64 decodes to empty bytes")
	}

	if len(decoded) > maxAuthStateBytes {
		return fmt.Errorf(
			"auth.storage_state.content_b64 decodes to %d bytes, exceeding the %d byte limit",
			len(decoded),
			maxAuthStateBytes,
		)
	}

	var probe map[string]any
	if unmarshalErr := json.Unmarshal(decoded, &probe); unmarshalErr != nil {
		return fmt.Errorf("auth.storage_state.content_b64 is not valid JSON: %w", unmarshalErr)
	}

	return nil
}

func normalizeJobURLFormAuth(
	ctx context.Context,
	resolver ipAddrResolver,
	validationMode targetValidationMode,
	in *jobURLAuthInput,
) (json.RawMessage, []byte, error) {
	if err := validateJobURLFormAuth(ctx, resolver, validationMode, in); err != nil {
		return nil, nil, err
	}

	out := map[string]any{
		"mode":      authModeForm,
		"login_url": in.Form.LoginURL,
		"steps":     in.Form.Steps,
		"success":   in.Form.Success,
	}

	raw, err := json.Marshal(out)
	if err != nil {
		return nil, nil, fmt.Errorf("auth.form: failed to re-marshal: %w", err)
	}

	return raw, nil, nil
}

func validateJobURLFormAuth(
	ctx context.Context,
	resolver ipAddrResolver,
	validationMode targetValidationMode,
	in *jobURLAuthInput,
) error {
	if in.StorageState != nil {
		return errors.New(`auth.storage_state is only valid with mode="storage_state"`)
	}

	if in.Form == nil {
		return errors.New(`auth.form is required when mode="form"`)
	}

	if strings.TrimSpace(in.Form.LoginURL) == "" {
		return errors.New("auth.form.login_url is required")
	}

	if err := validateTargetURLsWithResolver(ctx, resolver, []string{in.Form.LoginURL}, validationMode); err != nil {
		return fmt.Errorf("auth.form.login_url: %w", err)
	}

	if err := validateJobURLFormRecipe(in.Form); err != nil {
		return err
	}

	return nil
}

func validateJobURLFormRecipe(form *jobURLAuthFormRecipe) error {
	if len(form.Steps) == 0 {
		return errors.New("auth.form.steps must contain at least one step")
	}

	if form.Success == nil {
		return errors.New("auth.form.success is required")
	}

	if _, hasType := form.Success["type"].(string); !hasType {
		return errors.New("auth.form.success.type is required")
	}

	for i, step := range form.Steps {
		if validateErr := validateAuthStep(step); validateErr != nil {
			return fmt.Errorf("auth.form.steps[%d]: %w", i, validateErr)
		}
	}

	return nil
}

// validateAuthStep mirrors the schema-level validation for PreScanAction
// without bringing in the full ajv pipeline. We only need to catch shapes the
// orchestrator can't safely process; the scanner-runner re-validates at
// hydration time.
func validateAuthStep(step map[string]any) error {
	t, ok := step["type"].(string)
	if !ok {
		return errors.New("step.type is required")
	}

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
