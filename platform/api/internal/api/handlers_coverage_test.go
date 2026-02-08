package api

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattboback/stageflow/packages/shared-go/events"
	"github.com/mattboback/stageflow/packages/shared-go/models"
	"github.com/mattboback/stageflow/platform/api/internal/status"
)

// --- handleListScanners ---

func TestHandleListScanners_WithRegistry(t *testing.T) {
	server, _, _, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scanners", http.NoBody)
	rr := httptest.NewRecorder()

	server.handleListScanners(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp struct {
		Total   int `json:"total"`
		Enabled int `json:"enabled"`
	}

	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Total == 0 {
		t.Fatal("expected at least one scanner, got total=0")
	}

	if resp.Enabled == 0 {
		t.Fatal("expected at least one enabled scanner")
	}
}

func TestHandleListScanners_NilRegistry(t *testing.T) {
	server := &Server{
		config:          &ServerConfig{},
		scannerRegistry: nil,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scanners", http.NoBody)
	rr := httptest.NewRecorder()

	server.handleListScanners(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp struct {
		Scanners []struct {
			ID string `json:"id"`
		} `json:"scanners"`
		Total int `json:"total"`
	}

	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Total != 1 {
		t.Fatalf("expected fallback total=1, got %d", resp.Total)
	}

	if resp.Scanners[0].ID != "axe" {
		t.Fatalf("expected fallback scanner id=axe, got %q", resp.Scanners[0].ID)
	}
}

func TestHandleListScanners_MethodNotAllowed(t *testing.T) {
	server, _, _, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/scanners", http.NoBody)
	rr := httptest.NewRecorder()

	server.handleListScanners(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

// --- normalizeHighlightStyle ---

func TestNormalizeHighlightStyle(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"dashed", "dashed"},
		{"solid", "solid"},
		{"DASHED", "dashed"},
		{"SOLID", "solid"},
		{"  dashed  ", "dashed"},
		{"", "dashed"},
		{"invalid", "dashed"},
		{"dotted", "dashed"},
	}

	for _, tc := range tests {
		got := normalizeHighlightStyle(tc.input)
		if got != tc.want {
			t.Errorf("normalizeHighlightStyle(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// --- validateScannerConfigs ---

func TestValidateScannerConfigs_NoAiNavigator(t *testing.T) {
	result := validateScannerConfigs([]string{"axe", "lighthouse"}, nil)
	if result != nil {
		t.Fatalf("expected nil for non-ai-navigator modules, got %+v", result)
	}
}

func TestValidateScannerConfigs_AiNavigatorMissingConfig(t *testing.T) {
	result := validateScannerConfigs([]string{"ai-navigator"}, nil)
	if result == nil {
		t.Fatal("expected error for missing ai-navigator config")
	}

	if result.Field != "scanner_configs.ai-navigator" {
		t.Fatalf("expected field scanner_configs.ai-navigator, got %q", result.Field)
	}
}

func TestValidateScannerConfigs_AiNavigatorMissingGoal(t *testing.T) {
	configs := map[string]map[string]any{
		"ai-navigator": {"vision": map[string]any{"model": "openai/gpt-4o"}},
	}

	result := validateScannerConfigs([]string{"ai-navigator"}, configs)
	if result == nil {
		t.Fatal("expected error for missing goal")
	}

	if result.Field != "scanner_configs.ai-navigator.goal" {
		t.Fatalf("expected field scanner_configs.ai-navigator.goal, got %q", result.Field)
	}
}

func TestValidateScannerConfigs_AiNavigatorEmptyObjective(t *testing.T) {
	configs := map[string]map[string]any{
		"ai-navigator": {
			"goal":   map[string]any{"objective": "  "},
			"vision": map[string]any{"model": "openai/gpt-4o"},
		},
	}

	result := validateScannerConfigs([]string{"ai-navigator"}, configs)
	if result == nil {
		t.Fatal("expected error for empty objective")
	}

	if result.Field != "scanner_configs.ai-navigator.goal.objective" {
		t.Fatalf("expected field scanner_configs.ai-navigator.goal.objective, got %q", result.Field)
	}
}

func TestValidateScannerConfigs_AiNavigatorMissingModelNoEnv(t *testing.T) {
	t.Setenv("AI_NAVIGATOR_DEFAULT_MODEL", "")

	configs := map[string]map[string]any{
		"ai-navigator": {
			"goal": map[string]any{"objective": "Reach checkout"},
		},
	}

	result := validateScannerConfigs([]string{"ai-navigator"}, configs)
	if result == nil {
		t.Fatal("expected error for missing model without env default")
	}

	if result.Field != "scanner_configs.ai-navigator.vision.model" {
		t.Fatalf("expected field scanner_configs.ai-navigator.vision.model, got %q", result.Field)
	}
}

func TestValidateScannerConfigs_AiNavigatorFallsBackToEnvModel(t *testing.T) {
	t.Setenv("AI_NAVIGATOR_DEFAULT_MODEL", "openai/gpt-4o-mini")

	configs := map[string]map[string]any{
		"ai-navigator": {
			"goal": map[string]any{"objective": "Reach checkout"},
		},
	}

	result := validateScannerConfigs([]string{"ai-navigator"}, configs)
	if result != nil {
		t.Fatalf("expected nil, got error: %+v", result)
	}

	vision := configs["ai-navigator"]["vision"].(map[string]any)
	if vision["model"] != "openai/gpt-4o-mini" {
		t.Fatalf("expected model to be set from env, got %v", vision["model"])
	}
}

func TestValidateScannerConfigs_AiNavigatorInvalidProvider(t *testing.T) {
	configs := map[string]map[string]any{
		"ai-navigator": {
			"goal":   map[string]any{"objective": "Reach checkout"},
			"vision": map[string]any{"model": "openai/gpt-4o", "provider": "azure"},
		},
	}

	result := validateScannerConfigs([]string{"ai-navigator"}, configs)
	if result == nil {
		t.Fatal("expected error for invalid provider")
	}

	if result.Field != "scanner_configs.ai-navigator.vision.provider" {
		t.Fatalf("expected field scanner_configs.ai-navigator.vision.provider, got %q", result.Field)
	}
}

func TestValidateScannerConfigs_AiNavigatorOpenrouterProviderOK(t *testing.T) {
	configs := map[string]map[string]any{
		"ai-navigator": {
			"goal":   map[string]any{"objective": "Reach checkout"},
			"vision": map[string]any{"model": "openai/gpt-4o", "provider": "openrouter"},
		},
	}

	result := validateScannerConfigs([]string{"ai-navigator"}, configs)
	if result != nil {
		t.Fatalf("expected nil for openrouter provider, got: %+v", result)
	}
}

// --- handleJobZipUpload additional paths ---

func writeField(t *testing.T, w *multipart.Writer, key, value string) {
	t.Helper()

	if err := w.WriteField(key, value); err != nil {
		t.Fatalf("write field %s: %v", key, err)
	}
}

func addZipFile(t *testing.T, w *multipart.Writer, filename string, data []byte) {
	t.Helper()

	part, err := w.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}

	if _, writeErr := part.Write(data); writeErr != nil {
		t.Fatalf("write file: %v", writeErr)
	}
}

func TestZipUploadWithScannerConfigsField(t *testing.T) {
	server, store, _, publisher := newTestServer(t)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	addZipFile(t, writer, "site.zip", buildTestZip(t))
	writeField(t, writer, "modules", "axe")
	writeField(t, writer, "scanner_configs", `{"axe":{"rules":["color-contrast"]}}`)
	writeField(t, writer, "highlight_style", "solid")
	writeField(t, writer, "screenshot", "true")
	writeField(t, writer, "unknown_field", "ignored")
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/zip", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	server.handleJobZipUpload(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	if len(store.uploads) != 1 {
		t.Fatalf("expected 1 upload, got %d", len(store.uploads))
	}

	if len(publisher.envelopes) != 1 {
		t.Fatalf("expected 1 envelope, got %d", len(publisher.envelopes))
	}

	payload, ok := publisher.envelopes[0].Payload.(*events.JobCreatedPayload)
	if !ok {
		t.Fatal("expected JobCreatedPayload")
	}

	if payload.Config.HighlightStyle != "solid" {
		t.Fatalf("expected highlight_style=solid, got %q", payload.Config.HighlightStyle)
	}

	if !payload.Config.Screenshot {
		t.Fatal("expected screenshot=true")
	}

	if payload.Config.ScannerConfigs["axe"] == nil {
		t.Fatal("expected scanner_configs.axe to be set")
	}
}

func TestZipUploadInvalidScannerConfigsJSON(t *testing.T) {
	server, _, _, _ := newTestServer(t)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	addZipFile(t, writer, "site.zip", buildTestZip(t))
	writeField(t, writer, "modules", "axe")
	writeField(t, writer, "scanner_configs", "not-valid-json")
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/zip", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	server.handleJobZipUpload(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestZipUploadMissingFile(t *testing.T) {
	server, _, _, _ := newTestServer(t)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writeField(t, writer, "modules", "axe")
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/zip", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	server.handleJobZipUpload(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestZipUploadNonZipFile(t *testing.T) {
	server, _, _, _ := newTestServer(t)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	addZipFile(t, writer, "site.tar.gz", []byte("not a zip"))
	writeField(t, writer, "modules", "axe")
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/zip", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	server.handleJobZipUpload(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleJobZipUploadMethodNotAllowed(t *testing.T) {
	server, _, _, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/zip", http.NoBody)
	rr := httptest.NewRecorder()

	server.handleJobZipUpload(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

// --- handleJobStatus success + method not allowed ---

func TestHandleJobStatusFound(t *testing.T) {
	server, _, store, _ := newTestServer(t)

	jobID := "job-status-found"
	if err := store.HandleJobCreated(context.Background(), &events.JobCreatedPayload{
		JobID:     jobID,
		InputType: models.JobInputTypeURLs,
		URLs:      []string{"https://example.com"},
	}); err != nil {
		t.Fatalf("create job: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/"+jobID, http.NoBody)
	rr := httptest.NewRecorder()

	server.handleJobStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var job models.JobStatus
	if err := json.NewDecoder(rr.Body).Decode(&job); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if job.State != models.JobStateScanning {
		t.Fatalf("expected SCANNING, got %s", job.State)
	}
}

func TestHandleJobStatusMethodNotAllowed(t *testing.T) {
	server, _, _, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/test", http.NoBody)
	rr := httptest.NewRecorder()

	server.handleJobStatus(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

// --- handleJobReport ---

func TestHandleJobReportNotCompleted(t *testing.T) {
	server, _, store, _ := newTestServer(t)

	jobID := "job-report-pending"
	if err := store.HandleJobCreated(context.Background(), &events.JobCreatedPayload{
		JobID:     jobID,
		InputType: models.JobInputTypeURLs,
		URLs:      []string{"https://example.com"},
	}); err != nil {
		t.Fatalf("create job: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/"+jobID+"/report", http.NoBody)
	rr := httptest.NewRecorder()

	server.handleJobReport(rr, req, jobID)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleJobReportNotFound(t *testing.T) {
	server, _, _, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/missing/report", http.NoBody)
	rr := httptest.NewRecorder()

	server.handleJobReport(rr, req, "missing")

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestHandleJobReportRedirect(t *testing.T) {
	server, _, store, _ := newTestServer(t)

	jobID := "job-report-redirect"
	if err := store.HandleJobCreated(context.Background(), &events.JobCreatedPayload{
		JobID:     jobID,
		InputType: models.JobInputTypeURLs,
		URLs:      []string{"https://example.com"},
	}); err != nil {
		t.Fatalf("create job: %v", err)
	}

	if err := store.HandleJobCompleted(context.Background(), &events.JobCompletedPayload{
		JobID: jobID,
		Artifacts: events.ArtifactLocations{
			ReportHTML: jobID + "/report.html",
			ReportJSON: jobID + "/report.json",
		},
	}); err != nil {
		t.Fatalf("complete job: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/"+jobID+"/report", http.NoBody)
	rr := httptest.NewRecorder()

	server.handleJobReport(rr, req, jobID)

	if rr.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected 307, got %d: %s", rr.Code, rr.Body.String())
	}

	loc := rr.Header().Get("Location")
	if loc == "" {
		t.Fatal("expected Location header for redirect")
	}
}

// --- handleJobResults ---

func TestHandleJobResultsNotCompleted(t *testing.T) {
	server, _, store, _ := newTestServer(t)

	jobID := "job-results-pending"
	if err := store.HandleJobCreated(context.Background(), &events.JobCreatedPayload{
		JobID:     jobID,
		InputType: models.JobInputTypeURLs,
		URLs:      []string{"https://example.com"},
	}); err != nil {
		t.Fatalf("create job: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/"+jobID+"/results", http.NoBody)
	rr := httptest.NewRecorder()

	server.handleJobResults(rr, req, jobID)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleJobResultsNotFound(t *testing.T) {
	server, _, _, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/missing/results", http.NoBody)
	rr := httptest.NewRecorder()

	server.handleJobResults(rr, req, "missing")

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestHandleJobResultsCompletedNoReport(t *testing.T) {
	server, _, store, _ := newTestServer(t)

	jobID := "job-results-no-report"
	if err := store.HandleJobCreated(context.Background(), &events.JobCreatedPayload{
		JobID:     jobID,
		InputType: models.JobInputTypeURLs,
		URLs:      []string{"https://example.com"},
	}); err != nil {
		t.Fatalf("create job: %v", err)
	}

	if err := store.HandleJobCompleted(context.Background(), &events.JobCompletedPayload{
		JobID:     jobID,
		Artifacts: events.ArtifactLocations{},
	}); err != nil {
		t.Fatalf("complete job: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/"+jobID+"/results", http.NoBody)
	rr := httptest.NewRecorder()

	server.handleJobResults(rr, req, jobID)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestHandleJobResultsRedirect(t *testing.T) {
	server, _, store, _ := newTestServer(t)

	jobID := "job-results-redirect"
	if err := store.HandleJobCreated(context.Background(), &events.JobCreatedPayload{
		JobID:     jobID,
		InputType: models.JobInputTypeURLs,
		URLs:      []string{"https://example.com"},
	}); err != nil {
		t.Fatalf("create job: %v", err)
	}

	if err := store.HandleJobCompleted(context.Background(), &events.JobCompletedPayload{
		JobID: jobID,
		Artifacts: events.ArtifactLocations{
			ReportJSON: jobID + "/report.json",
		},
	}); err != nil {
		t.Fatalf("complete job: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/"+jobID+"/results", http.NoBody)
	rr := httptest.NewRecorder()

	server.handleJobResults(rr, req, jobID)

	if rr.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected 307, got %d: %s", rr.Code, rr.Body.String())
	}

	loc := rr.Header().Get("Location")
	if loc == "" {
		t.Fatal("expected Location header for redirect")
	}
}

// --- buildJobStatusResponse ---

func TestBuildJobStatusResponse_DoneWithArtifacts(t *testing.T) {
	server, _, store, _ := newTestServer(t)

	jobID := "job-response-done"
	if err := store.HandleJobCreated(context.Background(), &events.JobCreatedPayload{
		JobID:     jobID,
		InputType: models.JobInputTypeURLs,
		URLs:      []string{"https://example.com"},
	}); err != nil {
		t.Fatalf("create job: %v", err)
	}

	if err := store.HandleJobCompleted(context.Background(), &events.JobCompletedPayload{
		JobID: jobID,
		Artifacts: events.ArtifactLocations{
			ReportJSON: jobID + "/report.json",
			ReportHTML: jobID + "/report.html",
		},
	}); err != nil {
		t.Fatalf("complete job: %v", err)
	}

	rec, err := store.GetJob(context.Background(), jobID)
	if err != nil {
		t.Fatalf("get job: %v", err)
	}

	job, err := server.buildJobStatusResponse(context.Background(), rec)
	if err != nil {
		t.Fatalf("build response: %v", err)
	}

	if job.Artifacts == nil {
		t.Fatal("expected artifacts to be set")
	}

	if job.Artifacts.ReportJSON == "" {
		t.Fatal("expected report JSON URL")
	}

	if job.Artifacts.ReportHTML == "" {
		t.Fatal("expected report HTML URL")
	}
}

func TestBuildJobStatusResponse_Pending(t *testing.T) {
	server, _, store, _ := newTestServer(t)

	jobID := "job-response-pending"
	if err := store.HandleJobCreated(context.Background(), &events.JobCreatedPayload{
		JobID:     jobID,
		InputType: models.JobInputTypeURLs,
		URLs:      []string{"https://example.com"},
	}); err != nil {
		t.Fatalf("create job: %v", err)
	}

	rec, err := store.GetJob(context.Background(), jobID)
	if err != nil {
		t.Fatalf("get job: %v", err)
	}

	job, err := server.buildJobStatusResponse(context.Background(), rec)
	if err != nil {
		t.Fatalf("build response: %v", err)
	}

	if job.Artifacts != nil {
		t.Fatal("expected no artifacts for pending job")
	}
}

func TestBuildPerScannerArtifacts(t *testing.T) {
	server, _, _, _ := newTestServer(t)

	rec := &status.JobRecord{
		JobID: "test-per-scanner",
		State: models.JobStateDone,
		ScannerArtifacts: map[string]*status.ScannerArtifactRecord{
			"axe": {
				ScannerType: "axe",
				ResultsKey:  "test-per-scanner/axe/results.json",
				ReportKey:   "test-per-scanner/axe/report.html",
				StageLogKey: "test-per-scanner/axe/scan-stage-log.json",
				RecipeKey:   "test-per-scanner/axe/scan-recipe.json",
			},
		},
	}

	perScanner, ok := server.buildPerScannerArtifacts(context.Background(), rec)
	if !ok {
		t.Fatal("expected buildPerScannerArtifacts to return ok=true")
	}

	axeArtifacts, exists := perScanner["axe"]
	if !exists {
		t.Fatal("expected axe artifacts")
	}

	if axeArtifacts.ResultsJSON == "" {
		t.Fatal("expected results JSON URL for axe")
	}

	if axeArtifacts.ScannerType != "axe" {
		t.Fatalf("expected scanner_type=axe, got %q", axeArtifacts.ScannerType)
	}
}

// --- handleJobURLSubmit success + edge cases ---

func TestHandleJobURLSubmitSuccess(t *testing.T) {
	server, _, _, publisher := newTestServer(t)

	payload := map[string]any{
		"urls":    []string{"https://example.com"},
		"modules": []string{"axe"},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.handleJobURLSubmit(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if decErr := json.NewDecoder(rr.Body).Decode(&resp); decErr != nil {
		t.Fatalf("decode: %v", decErr)
	}

	if resp["job_id"] == nil || resp["job_id"] == "" {
		t.Fatal("expected job_id in response")
	}

	if resp["status"] != "pending" {
		t.Fatalf("expected status=pending, got %v", resp["status"])
	}

	if len(publisher.envelopes) != 1 {
		t.Fatalf("expected 1 published event, got %d", len(publisher.envelopes))
	}
}

func TestHandleJobURLSubmitInvalidJSON(t *testing.T) {
	server, _, _, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.handleJobURLSubmit(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleJobURLSubmitMethodNotAllowed(t *testing.T) {
	server, _, _, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/urls", http.NoBody)
	rr := httptest.NewRecorder()

	server.handleJobURLSubmit(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

// --- clientError ---

func TestClientErrorMessage(t *testing.T) {
	msgErr := newClientMessageError("test message")
	if msgErr.Error() != "test message" {
		t.Fatalf("expected 'test message', got %q", msgErr.Error())
	}

	emptyErr := &clientError{}
	if emptyErr.Error() != "client error" {
		t.Fatalf("expected 'client error', got %q", emptyErr.Error())
	}
}

// --- sanitizeFilename ---

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"file.zip", "file.zip"},
		{"/path/to/file.zip", "file.zip"},
		{"../../../etc/passwd", "passwd"},
	}

	for _, tc := range tests {
		got := sanitizeFilename(tc.input)
		if got != tc.want {
			t.Errorf("sanitizeFilename(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
