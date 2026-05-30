package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/httputil"
	"github.com/mattboback/stageflow/libs/go/logging"
	"github.com/mattboback/stageflow/libs/go/models"
	"github.com/mattboback/stageflow/libs/go/storage"
	"github.com/mattboback/stageflow/services/platform-api/internal/jobstatus"
)

const maxUploadSize = 100 * 1024 * 1024

// Note: keep reverse proxies/LB client_max_body_size in sync with maxUploadSize.

type formFieldTooLargeError struct {
	limit int64
}

func (e *formFieldTooLargeError) Error() string {
	return fmt.Sprintf("multipart field exceeds %d byte limit", e.limit)
}

func readFormValue(part *multipart.Part, limit int64) (string, error) {
	data, err := io.ReadAll(io.LimitReader(part, limit+1))
	if err != nil {
		return "", err
	}

	if int64(len(data)) > limit {
		return "", &formFieldTooLargeError{limit: limit}
	}

	return string(data), nil
}

func formReadClientError(field string, err error) error {
	var tooLarge *formFieldTooLargeError
	if errors.As(err, &tooLarge) {
		return newClientDetailError(httputil.NewValidationError(
			field,
			fmt.Sprintf("%s field is too large", field),
			fmt.Sprintf("Keep %s at or below %d bytes.", field, tooLarge.limit),
		))
	}

	return newClientMessageError(fmt.Sprintf("Failed to read %s field", field))
}

func sanitizeFilename(name string) string {
	if name == "" {
		return ""
	}

	return filepath.Base(name)
}

type clientError struct {
	status  int
	detail  *httputil.ErrorDetail
	message string
}

func (e *clientError) Error() string {
	switch {
	case e.message != "":
		return e.message
	case e.detail != nil:
		return e.detail.Message
	default:
		return "client error"
	}
}

func newClientDetailError(detail httputil.ErrorDetail) *clientError {
	return &clientError{status: http.StatusBadRequest, detail: &detail}
}

func newClientMessageError(message string) *clientError {
	return &clientError{status: http.StatusBadRequest, message: message}
}

type zipJobRequest struct {
	jobID          string
	runID          string
	zipPath        string
	modules        []string
	scannerConfigs map[string]map[string]any
	screenshot     bool
	highlightStyle string
}

type zipUploadState struct {
	fileUploaded   bool
	modulesValues  []string
	highlightStyle string
	screenshot     *bool
	zipPath        string
	scannerConfigs map[string]map[string]any
}

func (s *Server) handleJobZipUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

	jobReq, parseErr := s.parseZipUpload(r.Context(), r)
	if parseErr != nil && s.handleZipParseError(r.Context(), w, parseErr) {
		return
	}

	jobCtx := logging.WithJobID(r.Context(), jobReq.jobID)
	if err := jobCtx.Err(); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			httputil.RespondError(w, http.StatusServiceUnavailable, "Request timeout")
		} else {
			httputil.RespondError(w, http.StatusRequestTimeout, "Request canceled")
		}

		return
	}

	if err := s.enqueueZipJob(jobCtx, jobReq); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			httputil.RespondError(w, http.StatusServiceUnavailable, "Request timeout")

			return
		}

		if errors.Is(err, context.Canceled) {
			httputil.RespondError(w, http.StatusRequestTimeout, "Request canceled")

			return
		}

		logging.Error(jobCtx, "Failed to queue job", "error", err)
		httputil.RespondError(w, http.StatusInternalServerError, "Failed to persist job")

		return
	}

	httputil.RespondCreated(w, map[string]any{
		"job_id":  jobReq.jobID,
		"status":  "pending",
		"message": "Job created successfully",
	})
}

func (s *Server) handleZipParseError(ctx context.Context, w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.DeadlineExceeded) {
		httputil.RespondError(w, http.StatusServiceUnavailable, "Request timeout")

		return true
	}

	if errors.Is(err, context.Canceled) {
		httputil.RespondError(w, http.StatusRequestTimeout, "Request canceled")

		return true
	}

	if isRequestTooLarge(err) {
		httputil.RespondError(w, http.StatusRequestEntityTooLarge,
			fmt.Sprintf("Request body too large (max %d bytes)", maxUploadSize))

		return true
	}

	var cErr *clientError
	if errors.As(err, &cErr) {
		if cErr.detail != nil {
			httputil.RespondStructuredError(w, cErr.status, *cErr.detail)
		} else {
			httputil.RespondError(w, cErr.status, cErr.message)
		}

		return true
	}

	logging.Error(ctx, "Zip upload failed", "error", err)
	httputil.RespondError(w, http.StatusInternalServerError, "Failed to handle upload")

	return true
}

func isRequestTooLarge(err error) bool {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		return true
	}

	msg := err.Error()

	return strings.Contains(msg, "http: request body too large") || strings.Contains(msg, "request body too large")
}

func (s *Server) parseZipUpload(ctx context.Context, r *http.Request) (*zipJobRequest, error) {
	reader, err := r.MultipartReader()
	if err != nil {
		return nil, newClientMessageError("Invalid multipart form: " + err.Error())
	}

	jobID := uuid.New().String()
	jobCtx := logging.WithJobID(ctx, jobID)
	runID := uuid.New().String()[:8]
	jobCtx = logging.WithRunID(jobCtx, runID)

	state := &zipUploadState{}

	for {
		part, partErr := reader.NextPart()
		if errors.Is(partErr, io.EOF) {
			break
		}

		if partErr != nil {
			return nil, newClientMessageError("Failed to read multipart form: " + partErr.Error())
		}

		if partProcessErr := s.processZipPart(jobCtx, part, jobID, state); partProcessErr != nil {
			part.Close()

			return nil, partProcessErr
		}

		part.Close()
	}

	if !state.fileUploaded {
		return nil, newClientDetailError(httputil.NewValidationError(
			"file",
			"Missing required field 'file'",
			"Please include a ZIP file in the 'file' field of your multipart form upload.",
		))
	}

	modules, err := s.normalizeModules(state.modulesValues)
	if err != nil {
		if strings.Contains(err.Error(), "unsupported scanner module") {
			supported := s.listSupportedModuleIDs()

			return nil, newClientDetailError(
				httputil.NewUnsupportedModuleError(extractModuleName(err.Error()), supported),
			)
		}

		return nil, newClientMessageError(err.Error())
	}

	style := normalizeHighlightStyle(state.highlightStyle)
	screenshot := defaultScreenshot

	if state.screenshot != nil {
		screenshot = *state.screenshot
	}

	if detail := validateScannerConfigs(modules, state.scannerConfigs); detail != nil {
		return nil, newClientDetailError(*detail)
	}

	return &zipJobRequest{
		jobID:          jobID,
		runID:          runID,
		zipPath:        state.zipPath,
		modules:        modules,
		scannerConfigs: state.scannerConfigs,
		screenshot:     screenshot,
		highlightStyle: style,
	}, nil
}

func (s *Server) processZipPart(ctx context.Context, part *multipart.Part, jobID string, state *zipUploadState) error {
	switch part.FormName() {
	case "file":
		return s.handleZipFilePart(ctx, part, jobID, state)
	case "modules":
		return handleModulesPart(ctx, part, state)
	case "scanner_configs":
		return handleScannerConfigsPart(ctx, part, state)
	case "highlight_style":
		return handleHighlightStylePart(ctx, part, state)
	case artifactTypeScreenshot:
		return handleScreenshotPart(ctx, part, state)
	default:
		drainUnknownPart(ctx, part)

		return nil
	}
}

func (s *Server) handleZipFilePart(
	ctx context.Context,
	part *multipart.Part,
	jobID string,
	state *zipUploadState,
) error {
	if state.fileUploaded {
		return newClientDetailError(
			httputil.NewValidationError(
				"file",
				"Multiple 'file' parts are not supported",
				"Only include one ZIP archive per request.",
			),
		)
	}

	filename := sanitizeFilename(part.FileName())
	if filename == "" || !strings.HasSuffix(strings.ToLower(filename), ".zip") {
		return newClientDetailError(
			httputil.NewValidationError(
				"file",
				"File must be a ZIP archive",
				"Upload a .zip file that contains your job data.",
			),
		)
	}

	path := fmt.Sprintf("staging/%s/%s", jobID, filename)

	if err := ctx.Err(); err != nil {
		return err
	}

	if err := s.config.Storage.UploadFile(ctx, storage.BucketStaging, path, part, -1); err != nil {
		logging.Error(ctx, "Failed to upload file to MinIO", "error", err)

		return fmt.Errorf("failed to store file: %w", err)
	}

	state.zipPath = path
	state.fileUploaded = true

	return nil
}

func handleModulesPart(ctx context.Context, part *multipart.Part, state *zipUploadState) error {
	value, err := readFormValue(part, 16*1024)
	if err != nil {
		logging.Error(ctx, "Failed to read modules value", "error", err)

		return formReadClientError("modules", err)
	}

	if strings.TrimSpace(value) != "" {
		state.modulesValues = splitCSV(value)
	}

	return nil
}

func handleScannerConfigsPart(ctx context.Context, part *multipart.Part, state *zipUploadState) error {
	value, err := readFormValue(part, 256*1024)
	if err != nil {
		logging.Error(ctx, "Failed to read scanner_configs value", "error", err)

		return formReadClientError("scanner_configs", err)
	}

	if strings.TrimSpace(value) == "" {
		return nil
	}

	var parsed map[string]map[string]any
	if unmarshalErr := json.Unmarshal([]byte(value), &parsed); unmarshalErr != nil {
		logging.Error(ctx, "Failed to parse scanner_configs", "error", unmarshalErr)

		return newClientDetailError(httputil.NewValidationError(
			"scanner_configs",
			"Invalid scanner_configs JSON",
			`Provide a JSON object mapping scanner IDs to option objects, e.g. {"ai-navigator":{"goal":{"objective":"Reach checkout"},"vision":{"model":"openai/gpt-4o-mini"}}}`,
		))
	}

	state.scannerConfigs = parsed

	return nil
}

func handleHighlightStylePart(ctx context.Context, part *multipart.Part, state *zipUploadState) error {
	value, err := readFormValue(part, 32)
	if err != nil {
		logging.Error(ctx, "Failed to read highlight_style value", "error", err)

		return formReadClientError("highlight_style", err)
	}

	state.highlightStyle = value

	return nil
}

func handleScreenshotPart(ctx context.Context, part *multipart.Part, state *zipUploadState) error {
	value, err := readFormValue(part, 32)
	if err != nil {
		logging.Error(ctx, "Failed to read screenshot value", "error", err)

		return formReadClientError("screenshot", err)
	}

	enabled := strings.EqualFold(strings.TrimSpace(value), "true")
	state.screenshot = &enabled

	return nil
}

func drainUnknownPart(ctx context.Context, part *multipart.Part) {
	if _, err := readFormValue(part, 8*1024); err != nil {
		logging.Warn(ctx, "Failed to drain unknown multipart field", "field", part.FormName(), "error", err)
	}
}

func (s *Server) enqueueZipJob(ctx context.Context, req *zipJobRequest) error {
	if req.runID != "" {
		ctx = logging.WithRunID(ctx, req.runID)
	}

	payload := &events.JobCreatedPayload{
		JobID:     req.jobID,
		InputType: "zip",
		InputPath: req.zipPath,
		Config: models.JobConfig{
			Modules:        req.modules,
			ScannerConfigs: req.scannerConfigs,
			Screenshot:     req.screenshot,
			HighlightStyle: req.highlightStyle,
		},
	}

	envelope := events.NewEnvelope(events.EventJobCreated, req.jobID, "platform-api", payload)
	envelope.RequestID = logging.RequestID(ctx)
	envelope.RunID = logging.RunID(ctx)

	if err := ctx.Err(); err != nil {
		s.cleanupStagedZip(req.zipPath)

		return err
	}

	if err := s.config.Publisher.PublishJobCreated(ctx, envelope); err != nil {
		s.cleanupStagedZip(req.zipPath)

		return fmt.Errorf("failed to publish job.created event: %w", err)
	}

	if _, beginErr := s.jobStatus.Begin(ctx, jobstatus.BeginJob{
		Payload:    payload,
		ObservedAt: envelope.Timestamp,
	}); beginErr != nil {
		logging.Warn(ctx, "Failed to seed provisional job status", "error", beginErr)
	}

	logging.Info(ctx, "Job created", "filename", filepath.Base(req.zipPath), "input_type", "zip")

	return nil
}

func (s *Server) cleanupStagedZip(key string) {
	if key == "" || s == nil || s.config == nil || s.config.Storage == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.config.Storage.DeleteFile(ctx, storage.BucketStaging, key); err != nil {
		logging.Warn(ctx, "Failed to clean up staged ZIP", "key", key, "error", err)
	}
}
