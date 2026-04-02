package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/httputil"
	"github.com/mattboback/stageflow/libs/go/models"
	storagepkg "github.com/mattboback/stageflow/libs/go/storage"
	"github.com/mattboback/stageflow/services/platform-api/internal/jobstatus"
	"github.com/mattboback/stageflow/services/platform-api/internal/status"
)

func applyTestSignal(t *testing.T, server *Server, signal jobstatus.Signal) {
	t.Helper()

	if _, err := server.jobStatus.Apply(context.Background(), signal); err != nil {
		t.Fatalf("apply signal: %v", err)
	}
}

// --- handleListScanners ---

func TestHandleListScanners_WithRegistry(t *testing.T) {
	server, _, _ := newTestServer(t)

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
	server, _, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/scanners", http.NoBody)
	rr := httptest.NewRecorder()

	server.handleListScanners(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

// --- URL submit behavior ---

func TestHandleJobURLSubmitNormalizesHighlightStyle(t *testing.T) {
	server, _, publisher := newTestServer(t)

	body := bytes.NewBufferString(`{"urls":["https://example.com"],"highlight_style":"  SOLID  "}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls", body)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.handleJobURLSubmit(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	if len(publisher.envelopes) != 1 {
		t.Fatalf("expected 1 published event, got %d", len(publisher.envelopes))
	}

	created, ok := publisher.envelopes[0].Payload.(*events.JobCreatedPayload)
	if !ok || created == nil {
		t.Fatalf("expected payload type *events.JobCreatedPayload")
	}

	if created.Config.HighlightStyle != "solid" {
		t.Fatalf("HighlightStyle = %q, want %q", created.Config.HighlightStyle, "solid")
	}
}

func TestHandleJobURLSubmitInvalidHighlightStyleFallsBackToDefault(t *testing.T) {
	server, _, publisher := newTestServer(t)

	body := bytes.NewBufferString(`{"urls":["https://example.com"],"highlight_style":"dotted"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls", body)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.handleJobURLSubmit(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	if len(publisher.envelopes) != 1 {
		t.Fatalf("expected 1 published event, got %d", len(publisher.envelopes))
	}

	created, ok := publisher.envelopes[0].Payload.(*events.JobCreatedPayload)
	if !ok || created == nil {
		t.Fatalf("expected payload type *events.JobCreatedPayload")
	}

	if created.Config.HighlightStyle != defaultHighlightStyle {
		t.Fatalf("HighlightStyle = %q, want %q", created.Config.HighlightStyle, defaultHighlightStyle)
	}
}

func TestHandleJobURLSubmitAiNavigatorInvalidProviderReturnsValidationError(t *testing.T) {
	server, _, _ := newTestServer(t)

	payload := map[string]any{
		"urls":    []string{"https://example.com"},
		"modules": []string{"ai-navigator"},
		"scanner_configs": map[string]any{
			"ai-navigator": map[string]any{
				"goal": map[string]any{"objective": "Reach checkout"},
				"vision": map[string]any{
					"model":    "openai/gpt-4o",
					"provider": "azure",
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.handleJobURLSubmit(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}

	var parsed httputil.ErrorResponse
	if decodeErr := json.NewDecoder(rr.Body).Decode(&parsed); decodeErr != nil {
		t.Fatalf("decode error response: %v", decodeErr)
	}

	if parsed.Error.Field != "scanner_configs.ai-navigator.vision.provider" {
		t.Fatalf("Error.Field = %q, want %q", parsed.Error.Field, "scanner_configs.ai-navigator.vision.provider")
	}
}

func TestHandleJobURLSubmitAiNavigatorMissingGoalReturnsValidationError(t *testing.T) {
	server, _, _ := newTestServer(t)

	payload := map[string]any{
		"urls":    []string{"https://example.com"},
		"modules": []string{"ai-navigator"},
		"scanner_configs": map[string]any{
			"ai-navigator": map[string]any{
				"vision": map[string]any{
					"model": "openai/gpt-4o",
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.handleJobURLSubmit(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}

	var parsed httputil.ErrorResponse
	if decodeErr := json.NewDecoder(rr.Body).Decode(&parsed); decodeErr != nil {
		t.Fatalf("decode error response: %v", decodeErr)
	}

	if parsed.Error.Field != "scanner_configs.ai-navigator.goal" {
		t.Fatalf("Error.Field = %q, want %q", parsed.Error.Field, "scanner_configs.ai-navigator.goal")
	}
}

func TestHandleJobURLSubmitAiNavigatorMissingModelWithoutEnvReturnsValidationError(t *testing.T) {
	t.Setenv("AI_NAVIGATOR_DEFAULT_MODEL", "")

	server, _, _ := newTestServer(t)

	payload := map[string]any{
		"urls":    []string{"https://example.com"},
		"modules": []string{"ai-navigator"},
		"scanner_configs": map[string]any{
			"ai-navigator": map[string]any{
				"goal": map[string]any{"objective": "Reach checkout"},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.handleJobURLSubmit(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}

	var parsed httputil.ErrorResponse
	if decodeErr := json.NewDecoder(rr.Body).Decode(&parsed); decodeErr != nil {
		t.Fatalf("decode error response: %v", decodeErr)
	}

	if parsed.Error.Field != "scanner_configs.ai-navigator.vision.model" {
		t.Fatalf("Error.Field = %q, want %q", parsed.Error.Field, "scanner_configs.ai-navigator.vision.model")
	}
}

func TestHandleJobURLSubmitAiNavigatorOpenrouterProviderAccepted(t *testing.T) {
	server, _, publisher := newTestServer(t)

	payload := map[string]any{
		"urls":    []string{"https://example.com"},
		"modules": []string{"ai-navigator"},
		"scanner_configs": map[string]any{
			"ai-navigator": map[string]any{
				"goal": map[string]any{"objective": "Reach checkout"},
				"vision": map[string]any{
					"model":    "openai/gpt-4o",
					"provider": "openrouter",
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.handleJobURLSubmit(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	if len(publisher.envelopes) != 1 {
		t.Fatalf("expected 1 published event, got %d", len(publisher.envelopes))
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
	server, store, publisher := newTestServer(t)

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
	server, _, _ := newTestServer(t)

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
	server, _, _ := newTestServer(t)

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
	server, _, _ := newTestServer(t)

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
	server, _, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/zip", http.NoBody)
	rr := httptest.NewRecorder()

	server.handleJobZipUpload(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

// --- handleJobStatus success + method not allowed ---

func TestHandleJobStatusFound(t *testing.T) {
	server, _, _ := newTestServer(t)

	jobID := "job-status-found"
	applyTestSignal(t, server, jobstatus.Signal{
		Kind: jobstatus.SignalJobCreated,
		JobCreated: &events.JobCreatedPayload{
			JobID:     jobID,
			InputType: models.JobInputTypeURLs,
			URLs:      []string{"https://example.com"},
		},
	})

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
	server, _, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/test", http.NoBody)
	rr := httptest.NewRecorder()

	server.handleJobStatus(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

// --- handleJobReport ---

func TestHandleJobReportNotCompleted(t *testing.T) {
	server, _, _ := newTestServer(t)

	jobID := "job-report-pending"
	applyTestSignal(t, server, jobstatus.Signal{
		Kind: jobstatus.SignalJobCreated,
		JobCreated: &events.JobCreatedPayload{
			JobID:     jobID,
			InputType: models.JobInputTypeURLs,
			URLs:      []string{"https://example.com"},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/"+jobID+"/report", http.NoBody)
	rr := httptest.NewRecorder()

	server.handleJobReport(rr, req, jobID)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleJobReportNotFound(t *testing.T) {
	server, _, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/missing/report", http.NoBody)
	rr := httptest.NewRecorder()

	server.handleJobReport(rr, req, "missing")

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestHandleJobReportRedirect(t *testing.T) {
	server, _, _ := newTestServer(t)

	jobID := "job-report-redirect"
	applyTestSignal(t, server, jobstatus.Signal{
		Kind: jobstatus.SignalJobCreated,
		JobCreated: &events.JobCreatedPayload{
			JobID:     jobID,
			InputType: models.JobInputTypeURLs,
			URLs:      []string{"https://example.com"},
		},
	})
	applyTestSignal(t, server, jobstatus.Signal{
		Kind: jobstatus.SignalJobCompleted,
		JobCompleted: &events.JobCompletedPayload{
			JobID: jobID,
			Artifacts: events.ArtifactLocations{
				ReportHTML: jobID + "/report.html",
				ReportJSON: jobID + "/report.json",
			},
		},
	})

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
	server, _, _ := newTestServer(t)

	jobID := "job-results-pending"
	applyTestSignal(t, server, jobstatus.Signal{
		Kind: jobstatus.SignalJobCreated,
		JobCreated: &events.JobCreatedPayload{
			JobID:     jobID,
			InputType: models.JobInputTypeURLs,
			URLs:      []string{"https://example.com"},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/"+jobID+"/results", http.NoBody)
	rr := httptest.NewRecorder()

	server.handleJobResults(rr, req, jobID)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleJobResultsNotFound(t *testing.T) {
	server, _, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/missing/results", http.NoBody)
	rr := httptest.NewRecorder()

	server.handleJobResults(rr, req, "missing")

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestHandleJobResultsCompletedNoReport(t *testing.T) {
	server, _, _ := newTestServer(t)

	jobID := "job-results-no-report"
	applyTestSignal(t, server, jobstatus.Signal{
		Kind: jobstatus.SignalJobCreated,
		JobCreated: &events.JobCreatedPayload{
			JobID:     jobID,
			InputType: models.JobInputTypeURLs,
			URLs:      []string{"https://example.com"},
		},
	})
	applyTestSignal(t, server, jobstatus.Signal{
		Kind:         jobstatus.SignalJobCompleted,
		JobCompleted: &events.JobCompletedPayload{JobID: jobID, Artifacts: events.ArtifactLocations{}},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/"+jobID+"/results", http.NoBody)
	rr := httptest.NewRecorder()

	server.handleJobResults(rr, req, jobID)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestHandleJobResultsRedirect(t *testing.T) {
	server, _, _ := newTestServer(t)

	jobID := "job-results-redirect"
	applyTestSignal(t, server, jobstatus.Signal{
		Kind: jobstatus.SignalJobCreated,
		JobCreated: &events.JobCreatedPayload{
			JobID:     jobID,
			InputType: models.JobInputTypeURLs,
			URLs:      []string{"https://example.com"},
		},
	})
	applyTestSignal(t, server, jobstatus.Signal{
		Kind: jobstatus.SignalJobCompleted,
		JobCompleted: &events.JobCompletedPayload{
			JobID:     jobID,
			Artifacts: events.ArtifactLocations{ReportJSON: jobID + "/report.json"},
		},
	})

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
	server, _, _ := newTestServer(t)

	jobID := "job-response-done"
	applyTestSignal(t, server, jobstatus.Signal{
		Kind: jobstatus.SignalJobCreated,
		JobCreated: &events.JobCreatedPayload{
			JobID:     jobID,
			InputType: models.JobInputTypeURLs,
			URLs:      []string{"https://example.com"},
		},
	})
	applyTestSignal(t, server, jobstatus.Signal{
		Kind: jobstatus.SignalJobCompleted,
		JobCompleted: &events.JobCompletedPayload{
			JobID: jobID,
			Artifacts: events.ArtifactLocations{
				ReportJSON: jobID + "/report.json",
				ReportHTML: jobID + "/report.html",
			},
		},
	})

	rec, err := server.jobStatus.Current(context.Background(), jobID)
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

func TestBuildJobStatusResponse_DoneUsesAggregatedReportIssueCount(t *testing.T) {
	server, storage, _ := newTestServer(t)

	jobID := "job-response-report-count"
	reportKey := jobID + "/report.json"
	now := time.Now().UTC()
	infoCount := 2
	axeCount := 1
	seoCount := 4
	reportDoc := report.UnifiedReportV2{
		Version: "2.0.0",
		Meta: report.ReportMeta{
			JobId:       jobID,
			ScannedAt:   &now,
			CompletedAt: &now,
		},
		Summary: report.ReportSummary{
			TotalIssues: 5,
			BySeverity: report.SeverityCounts{
				Critical: 0,
				Serious:  1,
				Moderate: 2,
				Minor:    0,
				Info:     &infoCount,
			},
			ByScanner:       map[string]int{"axe": 1, "seo": 4},
			PagesScanned:    1,
			PagesWithIssues: 1,
		},
		Scanners: []report.ScannerSummary{
			{Id: "axe", Status: report.ScannerStatusSuccess, IssueCount: &axeCount},
			{Id: "seo", Status: report.ScannerStatusSuccess, IssueCount: &seoCount},
		},
		Pages: []report.PageSummary{{
			Id:         "page-1",
			Url:        "https://example.com",
			IssueCount: 5,
			StartedAt:  &now,
			FinishedAt: &now,
		}},
		Issues:    []report.IssueDetail{},
		Artifacts: []report.ReportArtifact{},
		Errors:    []report.ReportError{},
	}

	reportBytes, err := json.Marshal(reportDoc)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	storage.uploads[fmt.Sprintf("%s::%s", storagepkg.BucketArtifacts, reportKey)] = reportBytes

	rec := &status.JobRecord{
		JobID:           jobID,
		State:           models.JobStateDone,
		TotalViolations: 6,
		ReportJSONKey:   reportKey,
		ReportKey:       jobID + "/report.html",
	}

	job, err := server.buildJobStatusResponse(context.Background(), rec)
	if err != nil {
		t.Fatalf("build response: %v", err)
	}

	if job.TotalViolations != 5 {
		t.Fatalf("expected normalized violations count 5, got %d", job.TotalViolations)
	}
}

func TestBuildJobStatusResponse_Pending(t *testing.T) {
	server, _, _ := newTestServer(t)

	jobID := "job-response-pending"
	applyTestSignal(t, server, jobstatus.Signal{
		Kind: jobstatus.SignalJobCreated,
		JobCreated: &events.JobCreatedPayload{
			JobID:     jobID,
			InputType: models.JobInputTypeURLs,
			URLs:      []string{"https://example.com"},
		},
	})

	rec, err := server.jobStatus.Current(context.Background(), jobID)
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
	server, _, _ := newTestServer(t)

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
	server, _, publisher := newTestServer(t)

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
	server, _, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.handleJobURLSubmit(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleJobURLSubmitMethodNotAllowed(t *testing.T) {
	server, _, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/urls", http.NoBody)
	rr := httptest.NewRecorder()

	server.handleJobURLSubmit(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

func TestZipUploadSanitizesUploadedFilename(t *testing.T) {
	server, objectStore, _ := newTestServer(t)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	addZipFile(t, writer, "../../../etc/passwd.zip", buildTestZip(t))
	writeField(t, writer, "modules", "axe")

	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/zip", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	server.handleJobZipUpload(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	if len(objectStore.uploads) != 1 {
		t.Fatalf("expected 1 upload, got %d", len(objectStore.uploads))
	}

	var uploadedPath string
	for key := range objectStore.uploads {
		uploadedPath = key
	}

	if strings.Contains(uploadedPath, "../") {
		t.Fatalf("uploaded object path must not contain path traversal segments: %q", uploadedPath)
	}

	if !strings.HasSuffix(uploadedPath, "/passwd.zip") {
		t.Fatalf("uploaded object path = %q, want suffix %q", uploadedPath, "/passwd.zip")
	}
}
