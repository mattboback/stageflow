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
	"sync/atomic"
	"testing"
	"time"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
	"github.com/mattboback/stageflow/clients/cli/internal/testsupport"
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
	doc := testsupport.SampleReport(jobID)

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
				State:             apiclient.JobStateDone,
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
	body := `{"cookies":[{"value":"private-storage-state-canary"}],"origins":[]}`

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
		"--summary-only",
		"--auth-state",
		statePath,
		"--scanner",
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

	if strings.Contains(stdout+stderr, "private-storage-state-canary") ||
		strings.Contains(stdout+stderr, captured.body.Auth.StorageState.ContentBase64) {
		t.Fatal("CLI output disclosed storage-state contents")
	}

	// The form recipe path should not be set when --auth-state is used.
	if captured.body.Auth.Form != nil {
		t.Errorf("Form should be nil for storage_state mode")
	}
}

func TestScanAuthRecipe_AttachesLocallyResolvedLiterals(t *testing.T) {
	const (
		user     = "cli-canary-user@example.com"
		password = "  cli-canary-password &+雪\n  "
	)

	t.Setenv("STAGEFLOW_AUTH_USER", user)
	t.Setenv("STAGEFLOW_AUTH_PASSWORD", password)

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
		"--summary-only",
		"--auth-recipe",
		recipePath,
		"--scanner",
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

	if captured.hits != 1 {
		t.Fatalf("expected one submission, got %d", captured.hits)
	}

	for i, want := range []string{user, password} {
		if captured.body.Auth.Form.Steps[i]["value"] != want {
			t.Fatalf("step %d did not contain the exact local literal", i)
		}

		if strings.Contains(stdout+stderr, want) {
			t.Fatal("CLI output disclosed resolved credentials")
		}
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

func TestScanAuthRecipe_UnsetVariablePreventsHTTP(t *testing.T) {
	const (
		name   = "STAGEFLOW_AUTH_MISSING_CLI_TEST"
		canary = "resolved-secret-before-missing-variable"
	)

	t.Setenv(name, "")
	testsupport.RequireNoErr(t, os.Unsetenv(name))
	t.Setenv("STAGEFLOW_AUTH_PRESENT", canary)

	var hits atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	path := filepath.Join(t.TempDir(), "recipe.json")
	body := `{"mode":"form","login_url":"https://example.com/login","steps":[
{"type":"fill","selector":"#user","value":{"from_env":"STAGEFLOW_AUTH_PRESENT"}},
{"type":"fill","selector":"#password","value":{"from_env":"` + name + `"}}],"success":{"type":"load"}}`
	testsupport.RequireNoErr(t, os.WriteFile(path, []byte(body), 0o600))

	stdout, stderr, code := runCLI(
		t,
		"stageflow",
		"--api",
		server.URL,
		"scan",
		"https://example.com",
		"--auth-recipe",
		path,
	)
	if code != 2 || !strings.Contains(stderr, "from_env variable is unset") {
		t.Fatalf("expected unset-variable failure; code=%d", code)
	}

	if hits.Load() != 0 {
		t.Fatal("unset environment reference reached HTTP")
	}

	if strings.Contains(stdout+stderr, canary) {
		t.Fatal("failed scan disclosed an already resolved credential")
	}
}

func TestScanAuthRecipe_InvalidModeIsNotLogged(t *testing.T) {
	const canary = "private-mode-value"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req apiclient.SubmitJobRequest
		testsupport.RequireNoErr(t, json.NewDecoder(r.Body).Decode(&req))

		if req.Auth == nil || req.Auth.Mode != canary {
			t.Error("CLI did not preserve mode for server validation")
		}

		http.Error(w, `{"error":"Authentication configuration is invalid"}`, http.StatusBadRequest)
	}))
	defer server.Close()

	path := filepath.Join(t.TempDir(), "recipe.json")
	testsupport.RequireNoErr(t, os.WriteFile(path, []byte(`{"mode":"`+canary+`","steps":[]}`), 0o600))

	stdout, stderr, code := runCLI(
		t,
		"stageflow",
		"--api",
		server.URL,
		"scan",
		"https://example.com",
		"--auth-recipe",
		path,
	)
	if code == 0 {
		t.Fatal("expected API rejection")
	}

	if strings.Contains(stdout+stderr, canary) {
		t.Fatal("CLI logged untrusted recipe mode")
	}
}
