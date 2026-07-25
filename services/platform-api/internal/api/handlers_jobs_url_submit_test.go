// URL submission request handling -- validation, normalization, and defaulting
// of the submit payload. Exercises handlers_jobs_url_submit.go.

package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/httputil"
)

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

func TestHandleJobURLSubmitNormalizesBrowserEngine(t *testing.T) {
	server, _, publisher := newTestServer(t)

	body := bytes.NewBufferString(`{"urls":["https://example.com"],"browser":"  FIREFOX  "}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls", body)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.handleJobURLSubmit(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	created, ok := publisher.envelopes[0].Payload.(*events.JobCreatedPayload)
	if !ok || created == nil {
		t.Fatalf("expected payload type *events.JobCreatedPayload")
	}

	if created.Config.Browser != "firefox" {
		t.Fatalf("Browser = %q, want %q", created.Config.Browser, "firefox")
	}
}

func TestHandleJobURLSubmitInvalidBrowserEngineFallsBackToDefault(t *testing.T) {
	server, _, publisher := newTestServer(t)

	body := bytes.NewBufferString(`{"urls":["https://example.com"],"browser":"opera"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls", body)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.handleJobURLSubmit(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	created, ok := publisher.envelopes[0].Payload.(*events.JobCreatedPayload)
	if !ok || created == nil {
		t.Fatalf("expected payload type *events.JobCreatedPayload")
	}

	if created.Config.Browser != defaultBrowserEngine {
		t.Fatalf("Browser = %q, want %q", created.Config.Browser, defaultBrowserEngine)
	}
}

func TestHandleJobURLSubmitDefaultsBrowserEngineWhenOmitted(t *testing.T) {
	server, _, publisher := newTestServer(t)

	body := bytes.NewBufferString(`{"urls":["https://example.com"]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls", body)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.handleJobURLSubmit(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	created, ok := publisher.envelopes[0].Payload.(*events.JobCreatedPayload)
	if !ok || created == nil {
		t.Fatalf("expected payload type *events.JobCreatedPayload")
	}

	if created.Config.Browser != defaultBrowserEngine {
		t.Fatalf("Browser = %q, want %q (omitted browser should default)", created.Config.Browser, defaultBrowserEngine)
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

	published, ok := publisher.envelopes[0].Payload.(*events.JobCreatedPayload)
	if !ok {
		t.Fatalf("published payload = %T, want *events.JobCreatedPayload", publisher.envelopes[0].Payload)
	}

	if !published.Config.Screenshot {
		t.Fatal("expected omitted screenshot field to default to true")
	}
}

func TestHandleJobURLSubmitExplicitScreenshotFalse(t *testing.T) {
	server, _, publisher := newTestServer(t)

	payload := map[string]any{
		"urls":       []string{"https://example.com"},
		"modules":    []string{"axe"},
		"screenshot": false,
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

	if len(publisher.envelopes) != 1 {
		t.Fatalf("expected 1 published event, got %d", len(publisher.envelopes))
	}

	published, ok := publisher.envelopes[0].Payload.(*events.JobCreatedPayload)
	if !ok {
		t.Fatalf("published payload = %T, want *events.JobCreatedPayload", publisher.envelopes[0].Payload)
	}

	if published.Config.Screenshot {
		t.Fatal("expected explicit screenshot=false to be honored")
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

func TestHandleJobURLSubmitUnknownField(t *testing.T) {
	server, _, _ := newTestServer(t)

	body := `{"urls":["https://example.com"],"scanners":["axe"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.handleJobURLSubmit(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}

	if !strings.Contains(rr.Body.String(), "scanners") {
		t.Fatalf("expected error to name the unknown field, got: %s", rr.Body.String())
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
