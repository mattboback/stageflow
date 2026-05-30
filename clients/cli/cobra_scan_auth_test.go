package main

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
)

// authCaptureScanAPIServer captures the body of the first POST /api/v1/jobs/urls
// it receives and serves a passing job/report afterwards so the CLI exits cleanly.
type capturedScanSubmission struct {
	mu   sync.Mutex
	body apiclient.SubmitJobRequest
	hits int
}

func newCapturingScanAPI(t *testing.T) (*httptest.Server, *capturedScanSubmission) {
	t.Helper()

	jobID := "job-auth-test"
	captured := &capturedScanSubmission{}
	doc := sampleReport(jobID)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/jobs/urls":
			captured.mu.Lock()
			defer captured.mu.Unlock()

			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("failed to read submitted body: %v", err)
			}

			if unmarshalErr := json.Unmarshal(body, &captured.body); unmarshalErr != nil {
				t.Fatalf("failed to decode submitted body: %v", unmarshalErr)
			}

			captured.hits++

			writeJSONResponse(t, w, http.StatusAccepted, apiclient.SubmitJobResponse{JobID: jobID, Status: "accepted"})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/jobs/"+jobID:
			writeJSONResponse(t, w, http.StatusOK, apiclient.JobStatus{
				ID:                jobID,
				State:             jobStateDone,
				ExpectedScanners:  []string{"axe"},
				CompletedScanners: []string{"axe"},
				CreatedAt:         time.Unix(1700000000, 0).UTC(),
				UpdatedAt:         time.Unix(1700000030, 0).UTC(),
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/jobs/"+jobID+"/results":
			writeJSONResponse(t, w, http.StatusOK, doc)
		default:
			http.NotFound(w, r)
		}
	})

	return httptest.NewServer(handler), captured
}

func TestScanAuthState_AttachesBase64StorageState(t *testing.T) {
	server, captured := newCapturingScanAPI(t)
	defer server.Close()

	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")
	body := `{"cookies":[],"origins":[]}`

	if err := os.WriteFile(statePath, []byte(body), 0o600); err != nil {
		t.Fatalf("write state: %v", err)
	}

	stdout, stderr, exitCode := runCLI(
		t,
		"stageflow",
		"--api",
		server.URL,
		"scan",
		"https://example.com",
		"--no-stream",
		"--summary-only",
		"--auth-state",
		statePath,
		"--scanners",
		"axe",
	)
	if exitCode != 0 {
		t.Fatalf("exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
	}

	if !strings.Contains(stderr, "Attaching auth block (mode=storage_state)") {
		t.Errorf("expected auth-attach notice in stderr; got %q", stderr)
	}

	captured.mu.Lock()
	defer captured.mu.Unlock()

	if captured.hits != 1 {
		t.Fatalf("expected 1 POST, got %d", captured.hits)
	}

	if captured.body.Auth == nil {
		t.Fatalf("expected Auth in submitted body, got nil")
	}

	if captured.body.Auth.Mode != "storage_state" {
		t.Errorf("Auth.Mode = %q, want storage_state", captured.body.Auth.Mode)
	}

	if captured.body.Auth.StorageState == nil {
		t.Fatalf("expected StorageState payload")
	}

	decoded, err := base64.StdEncoding.DecodeString(captured.body.Auth.StorageState.ContentBase64)
	if err != nil {
		t.Fatalf("decode content_b64: %v", err)
	}

	if string(decoded) != body {
		t.Errorf("submitted body mismatch:\n got=%q\nwant=%q", string(decoded), body)
	}

	// The form recipe path should not be set when --auth-state is used.
	if captured.body.Auth.Form != nil {
		t.Errorf("Form should be nil for storage_state mode")
	}
}

func TestScanAuthRecipe_AttachesFormRecipeWithFromEnv(t *testing.T) {
	server, captured := newCapturingScanAPI(t)
	defer server.Close()

	tmp := t.TempDir()
	recipePath := filepath.Join(tmp, "recipe.yaml")

	yaml := `mode: form
login_url: https://example.com/login
steps:
  - type: fill
    selector: input[name=email]
    value:
      from_env: STAGEFLOW_AUTH_USER
  - type: fill
    selector: input[name=password]
    value:
      from_env: STAGEFLOW_AUTH_PASSWORD
  - type: click
    selector: button[type=submit]
success:
  type: selector
  selector: '[data-test=signed-in]'
  timeout: 15000
`
	if err := os.WriteFile(recipePath, []byte(yaml), 0o600); err != nil {
		t.Fatalf("write recipe: %v", err)
	}

	stdout, stderr, exitCode := runCLI(
		t,
		"stageflow",
		"--api",
		server.URL,
		"scan",
		"https://example.com",
		"--no-stream",
		"--summary-only",
		"--auth-recipe",
		recipePath,
		"--scanners",
		"axe",
	)
	if exitCode != 0 {
		t.Fatalf("exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
	}

	if !strings.Contains(stderr, "Attaching auth block (mode=form)") {
		t.Errorf("expected auth-attach notice in stderr; got %q", stderr)
	}

	captured.mu.Lock()
	defer captured.mu.Unlock()

	if captured.body.Auth == nil || captured.body.Auth.Form == nil {
		t.Fatalf("expected form Auth, got %#v", captured.body.Auth)
	}

	if captured.body.Auth.Form.LoginURL != "https://example.com/login" {
		t.Errorf("LoginURL = %q", captured.body.Auth.Form.LoginURL)
	}

	if len(captured.body.Auth.Form.Steps) != 3 {
		t.Fatalf("steps len = %d", len(captured.body.Auth.Form.Steps))
	}

	// Re-marshal to JSON and assert the from_env references survive the wire
	// shape unchanged. Concretely: we must NOT see resolved values, only refs.
	wire, err := json.Marshal(captured.body.Auth)
	if err != nil {
		t.Fatalf("marshal Auth: %v", err)
	}

	wireStr := string(wire)
	if !strings.Contains(wireStr, `"from_env":"STAGEFLOW_AUTH_USER"`) {
		t.Errorf("USER from_env reference missing: %s", wireStr)
	}

	if !strings.Contains(wireStr, `"from_env":"STAGEFLOW_AUTH_PASSWORD"`) {
		t.Errorf("PASSWORD from_env reference missing: %s", wireStr)
	}

	// Defensive: prove that no resolved literal credential appears, even if
	// the host environment defines those vars.
	t.Setenv("STAGEFLOW_AUTH_USER", "demo@example.com")
	t.Setenv("STAGEFLOW_AUTH_PASSWORD", "hunter2-not-on-the-wire")

	if strings.Contains(wireStr, "demo@example.com") || strings.Contains(wireStr, "hunter2-not-on-the-wire") {
		t.Errorf("resolved credentials leaked into submitted Auth: %s", wireStr)
	}
}

func TestScanAuth_MutuallyExclusiveFlags(t *testing.T) {
	tmp := t.TempDir()
	recipePath := filepath.Join(tmp, "r.yaml")
	statePath := filepath.Join(tmp, "s.json")

	recipe := []byte(
		"mode: form\n" +
			"login_url: https://x\n" +
			"steps: [{type: click, selector: a}]\n" +
			"success: {type: load}\n",
	)
	if err := os.WriteFile(recipePath, recipe, 0o600); err != nil {
		t.Fatalf("write recipe: %v", err)
	}

	if err := os.WriteFile(statePath, []byte(`{"cookies":[]}`), 0o600); err != nil {
		t.Fatalf("write state: %v", err)
	}

	stdout, stderr, exitCode := runCLI(
		t,
		"stageflow",
		"scan",
		"https://example.com",
		"--auth-state",
		statePath,
		"--auth-recipe",
		recipePath,
	)
	if exitCode == 0 {
		t.Fatalf("expected non-zero exit; stdout=%q stderr=%q", stdout, stderr)
	}

	if !strings.Contains(stderr, "mutually exclusive") &&
		!strings.Contains(stderr, "if any flags in the group") {
		t.Errorf("expected mutually-exclusive error in stderr; got %q", stderr)
	}
}

func TestScanAuth_ProjectModeRejectsAuthFlags(t *testing.T) {
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "s.json")

	if err := os.WriteFile(statePath, []byte(`{"cookies":[]}`), 0o600); err != nil {
		t.Fatalf("write state: %v", err)
	}

	stdout, stderr, exitCode := runCLI(
		t,
		"stageflow",
		"scan",
		"--project",
		"demo-site",
		"--auth-state",
		statePath,
	)
	if exitCode == 0 {
		t.Fatalf("expected non-zero exit; stdout=%q stderr=%q", stdout, stderr)
	}

	if !strings.Contains(stderr, "not supported with --project") {
		t.Errorf("expected project-mode rejection in stderr; got %q", stderr)
	}
}
