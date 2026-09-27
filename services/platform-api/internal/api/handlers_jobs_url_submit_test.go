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
)

func TestHandleJobURLSubmitAcceptsOmittedOrChromiumBrowser(t *testing.T) {
	for _, body := range []string{
		`{"urls":["https://example.com"]}`,
		`{"urls":["https://example.com"],"browser":"chromium"}`,
		`{"urls":["https://example.com"],"browser":"  CHROMIUM  "}`,
	} {
		server, _, publisher := newTestServer(t)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		server.handleJobURLSubmit(rr, req)

		if rr.Code != http.StatusCreated {
			t.Fatalf("%s: expected 201, got %d: %s", body, rr.Code, rr.Body.String())
		}

		if len(publisher.envelopes) != 1 {
			t.Fatalf("%s: expected 1 envelope, got %d", body, len(publisher.envelopes))
		}
	}
}

func TestHandleJobURLSubmitRejectsOtherBrowserEngines(t *testing.T) {
	for _, engine := range []string{"firefox", "webkit", "opera"} {
		server, _, publisher := newTestServer(t)

		body := bytes.NewBufferString(`{"urls":["https://example.com"],"browser":"` + engine + `"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls", body)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		server.handleJobURLSubmit(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("%s: expected 400, got %d: %s", engine, rr.Code, rr.Body.String())
		}

		if !strings.Contains(rr.Body.String(), "browser") {
			t.Fatalf("%s: expected a browser validation error, got %s", engine, rr.Body.String())
		}

		if len(publisher.envelopes) != 0 {
			t.Fatalf("%s: expected no job to be published, got %d", engine, len(publisher.envelopes))
		}
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
