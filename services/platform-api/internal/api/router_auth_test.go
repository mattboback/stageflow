package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/httputil"
)

const literalBrowserAuthBody = `{
	"urls":["https://example.com/dashboard"],
	"auth":{
		"mode":"form",
		"form":{
			"login_url":"https://example.com/login",
			"steps":[
				{"type":"fill","selector":"#email","value":"demo@example.com"},
				{"type":"fill","selector":"#password","value":"throwaway-password"},
				{"type":"click","selector":"button[type=submit]"}
			],
			"success":{"type":"url_matches","pattern":"/dashboard"}
		}
	}
}`

const testAPIToken = "caller-token"

// routeCall sends one request through the full router, optionally with the API
// token, from a fixed client address.
func routeCall(t *testing.T, server *Server, method, path, body, apiKey string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	if apiKey != "" {
		req.Header.Set("X-Api-Key", apiKey)
	}

	rr := httptest.NewRecorder()
	server.Router().ServeHTTP(rr, req)

	return rr
}

func TestRouterPublicRoutesIgnoreMissingOrWrongKey(t *testing.T) {
	t.Setenv(publicSubmissionBurstEnv, "20")

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
	}{
		{
			name:       "URL intake",
			method:     http.MethodPost,
			path:       "/api/v1/jobs/urls",
			body:       `{"urls":["https://example.com"]}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "anonymous URL intake",
			method:     http.MethodPost,
			path:       "/api/v1/jobs/urls/anonymous",
			body:       `{"urls":["https://example.com"]}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "browser auth URL intake",
			method:     http.MethodPost,
			path:       "/api/v1/jobs/urls/browser-auth",
			body:       literalBrowserAuthBody,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "page discovery",
			method:     http.MethodPost,
			path:       "/api/v1/discover",
			body:       `{`,
			wantStatus: http.StatusBadRequest,
		},
		{name: "scanner catalog", method: http.MethodGet, path: "/api/v1/scanners", wantStatus: http.StatusOK},
		{name: "job status", method: http.MethodGet, path: "/api/v1/jobs/job-123", wantStatus: http.StatusNotFound},
		{name: "job delete", method: http.MethodDelete, path: "/api/v1/jobs/job-123", wantStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		for _, key := range []string{"", "wrong-token"} {
			t.Run(tt.name+" key="+key, func(t *testing.T) {
				server, _, _ := newTestServer(t)
				server.config.APIToken = testAPIToken

				rr := routeCall(t, server, tt.method, tt.path, tt.body, key)
				if rr.Code != tt.wantStatus {
					t.Fatalf("status = %d, want %d: %s", rr.Code, tt.wantStatus, rr.Body.String())
				}
			})
		}
	}
}

func TestPublicSubmissionLimitAppliesOnlyToCallersWithoutKey(t *testing.T) {
	t.Setenv(publicSubmissionRateEnv, "1")
	t.Setenv(publicSubmissionBurstEnv, "1")

	submit := func(router http.Handler, remoteAddr, apiKey string) int {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls", strings.NewReader(`{`))
		req.RemoteAddr = remoteAddr

		if apiKey != "" {
			req.Header.Set("X-Api-Key", apiKey)
		}

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		return rr.Code
	}

	server, _, _ := newTestServer(t)
	server.config.APIToken = testAPIToken
	router := server.Router()

	if code := submit(router, "198.51.100.7:1000", ""); code != http.StatusBadRequest {
		t.Fatalf("first keyless submission = %d, want 400 (malformed body, not limited)", code)
	}

	if code := submit(router, "198.51.100.7:1000", ""); code != http.StatusTooManyRequests {
		t.Fatalf("second keyless submission = %d, want 429", code)
	}

	for i := range 3 {
		if code := submit(router, "198.51.100.8:1000", testAPIToken); code != http.StatusBadRequest {
			t.Fatalf("token submission %d = %d, want 400 (never limited)", i, code)
		}
	}
}

type urlSubmissionPolicyCase struct {
	name           string
	body           string
	apiKey         string
	allowPrivate   bool
	wantStatus     int
	wantErrorField string
}

const (
	storageStateBody = `{"urls":["https://example.com"],"auth":{"mode":"storage_state","storage_state":{"content_b64":"e30="}}}`
	fromEnvFillBody  = `{
		"urls":["https://example.com/dashboard"],
		"auth":{"mode":"form","form":{
			"login_url":"https://example.com/login",
			"steps":[{"type":"fill","selector":"#password","value":{"from_env":"POSTGRES_PASSWORD"}}],
			"success":{"type":"url_matches","pattern":"/dashboard"}
		}}
	}`
	fromEnvClickBody = `{
		"urls":["https://example.com/dashboard"],
		"auth":{"mode":"form","form":{
			"login_url":"https://example.com/login",
			"steps":[{"type":"click","selector":"#go","value":{"from_env":"DATABASE_URL"}}],
			"success":{"type":"url_matches","pattern":"/dashboard"}
		}}
	}`
	privateTargetBody = `{"urls":["http://127.0.0.1"],"allow_private_targets":true}`
)

// TestURLSubmissionPolicy covers POST /api/v1/jobs/urls: auth recipes from any
// caller, literal values only, and private targets only for a token caller on an
// instance that allows them.
func TestURLSubmissionPolicy(t *testing.T) {
	t.Setenv(publicSubmissionBurstEnv, "20")

	tests := []urlSubmissionPolicyCase{
		{name: "no key, no auth", body: `{"urls":["https://example.com"]}`, wantStatus: http.StatusCreated},
		{name: "no key, literal form recipe", body: literalBrowserAuthBody, wantStatus: http.StatusCreated},
		{name: "no key, storage state", body: storageStateBody, wantStatus: http.StatusCreated},
		{
			name:           "no key, from_env reference",
			body:           fromEnvFillBody,
			wantStatus:     http.StatusBadRequest,
			wantErrorField: "auth.form.steps[0].value",
		},
		{
			name:           "key, from_env reference",
			body:           fromEnvFillBody,
			apiKey:         testAPIToken,
			wantStatus:     http.StatusBadRequest,
			wantErrorField: "auth.form.steps[0].value",
		},
		{
			name:           "no key, from_env on a click step",
			body:           fromEnvClickBody,
			wantStatus:     http.StatusBadRequest,
			wantErrorField: "auth.form.steps[0].value",
		},
		{
			name:           "no key, private targets on an instance that allows them",
			body:           privateTargetBody,
			allowPrivate:   true,
			wantStatus:     http.StatusBadRequest,
			wantErrorField: "allow_private_targets",
		},
		{
			name:           "key, private targets on an instance that forbids them",
			body:           privateTargetBody,
			apiKey:         testAPIToken,
			wantStatus:     http.StatusBadRequest,
			wantErrorField: "allow_private_targets",
		},
		{
			name:         "key, private targets on an instance that allows them",
			body:         privateTargetBody,
			apiKey:       testAPIToken,
			allowPrivate: true,
			wantStatus:   http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertURLSubmissionPolicy(t, tt)
		})
	}
}

func assertURLSubmissionPolicy(t *testing.T, tt urlSubmissionPolicyCase) {
	t.Helper()

	server, _, publisher := newTestServer(t)
	server.config.APIToken = testAPIToken
	server.config.AllowPrivateTargets = tt.allowPrivate

	rr := routeCall(t, server, http.MethodPost, "/api/v1/jobs/urls", tt.body, tt.apiKey)
	if rr.Code != tt.wantStatus {
		t.Fatalf("status = %d, want %d: %s", rr.Code, tt.wantStatus, rr.Body.String())
	}

	if tt.wantStatus == http.StatusCreated {
		if len(publisher.envelopes) != 1 {
			t.Fatalf("expected one published job, got %d", len(publisher.envelopes))
		}

		return
	}

	if len(publisher.envelopes) != 0 {
		t.Fatalf("rejected request published %d jobs", len(publisher.envelopes))
	}

	var response httputil.ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("decode error response: %v", err)
	}

	if response.Error.Field != tt.wantErrorField {
		t.Fatalf("error field = %q, want %q", response.Error.Field, tt.wantErrorField)
	}

	if strings.Contains(rr.Body.String(), "POSTGRES_PASSWORD") || strings.Contains(rr.Body.String(), "DATABASE_URL") {
		t.Fatalf("error response echoed the env reference: %s", rr.Body.String())
	}
}

func TestAnonymousURLSubmissionAcceptsOnlyPublicUnauthenticatedJobs(t *testing.T) {
	t.Setenv(publicSubmissionBurstEnv, "20")

	tests := []struct {
		name           string
		body           string
		allowPrivate   bool
		wantStatus     int
		wantErrorField string
	}{
		{
			name:       "public job",
			body:       `{"urls":["https://example.com"]}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:           "form auth",
			body:           literalBrowserAuthBody,
			wantStatus:     http.StatusBadRequest,
			wantErrorField: "auth",
		},
		{
			name:           "storage state auth",
			body:           `{"urls":["https://example.com"],"auth":{"mode":"storage_state","storage_state":{"content_b64":"e30="}}}`,
			wantStatus:     http.StatusBadRequest,
			wantErrorField: "auth",
		},
		{
			name:           "private target opt-in even when server allows it",
			body:           `{"urls":["http://127.0.0.1"],"allow_private_targets":true}`,
			allowPrivate:   true,
			wantStatus:     http.StatusBadRequest,
			wantErrorField: "allow_private_targets",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, _, publisher := newTestServer(t)
			server.config.AllowPrivateTargets = tt.allowPrivate

			req := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/jobs/urls/anonymous",
				strings.NewReader(tt.body),
			)
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()

			server.Router().ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Fatalf("expected %d, got %d: %s", tt.wantStatus, rr.Code, rr.Body.String())
			}

			if tt.wantStatus == http.StatusCreated {
				if len(publisher.envelopes) != 1 {
					t.Fatalf("expected one published job, got %d", len(publisher.envelopes))
				}

				return
			}

			if len(publisher.envelopes) != 0 {
				t.Fatalf("rejected request published %d jobs", len(publisher.envelopes))
			}

			var response httputil.ErrorResponse
			if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
				t.Fatalf("decode error response: %v", err)
			}

			if response.Error.Field != tt.wantErrorField {
				t.Fatalf("error field = %q, want %q", response.Error.Field, tt.wantErrorField)
			}
		})
	}
}

func TestURLSubmissionParseErrorsDoNotEchoEmbeddedCredentials(t *testing.T) {
	t.Setenv(publicSubmissionBurstEnv, "20")

	tests := []struct {
		name   string
		target string
		canary string
	}{
		{
			name:   "unsupported scheme",
			target: "ftp://demo:unsupported-password-canary@example.com/archive",
			canary: "unsupported-password-canary",
		},
		{
			name:   "malformed userinfo escape",
			target: "https://demo:malformed-password-canary%zz@example.com/",
			canary: "malformed-password-canary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, _, publisher := newTestServer(t)

			body, err := json.Marshal(map[string]any{"urls": []string{tt.target}})
			if err != nil {
				t.Fatalf("marshal request: %v", err)
			}

			req := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/jobs/urls/anonymous",
				strings.NewReader(string(body)),
			)
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			server.Router().ServeHTTP(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400: %s", rr.Code, rr.Body.String())
			}

			if strings.Contains(rr.Body.String(), tt.canary) {
				t.Fatalf("error response echoed embedded credentials: %s", rr.Body.String())
			}

			if len(publisher.envelopes) != 0 {
				t.Fatalf("rejected target published %d jobs", len(publisher.envelopes))
			}
		})
	}
}

type browserAuthSubmissionTestCase struct {
	name           string
	body           string
	allowPrivate   bool
	wantStatus     int
	wantErrorField string
	forbidResponse string
}

func TestBrowserAuthURLSubmissionAcceptsOnlyLiteralPublicFormRecipes(t *testing.T) {
	t.Setenv(publicSubmissionBurstEnv, "20")

	privateTargetBody := strings.Replace(
		literalBrowserAuthBody,
		`"urls":["https://example.com/dashboard"]`,
		`"urls":["http://127.0.0.1"],"allow_private_targets":true`,
		1,
	)

	tests := []browserAuthSubmissionTestCase{
		{name: "literal form recipe", body: literalBrowserAuthBody, wantStatus: http.StatusCreated},
		{
			name:           "missing auth",
			body:           `{"urls":["https://example.com"]}`,
			wantStatus:     http.StatusBadRequest,
			wantErrorField: "auth",
		},
		{
			name:           "storage state",
			body:           `{"urls":["https://example.com"],"auth":{"mode":"storage_state","storage_state":{"content_b64":"e30="}}}`,
			wantStatus:     http.StatusBadRequest,
			wantErrorField: "auth.mode",
		},
		{
			name: "environment reference",
			body: `{
				"urls":["https://example.com"],
				"auth":{"mode":"form","form":{
					"login_url":"https://example.com/login",
					"steps":[{"type":"fill","selector":"#password","value":{"from_env":"DEMO_PASSWORD"}}],
					"success":{"type":"url_matches","pattern":"/dashboard"}
				}}
			}`,
			wantStatus:     http.StatusBadRequest,
			wantErrorField: "auth.form.steps[0].value",
		},
		{
			name: "login URL embedded credentials",
			body: strings.Replace(
				literalBrowserAuthBody,
				"https://example.com/login",
				"https://demo:login-password-canary-828f@example.com/login",
				1,
			),
			wantStatus:     http.StatusBadRequest,
			wantErrorField: "auth",
			forbidResponse: "login-password-canary-828f",
		},
		{
			name:           "private target opt-in even when server allows it",
			body:           privateTargetBody,
			allowPrivate:   true,
			wantStatus:     http.StatusBadRequest,
			wantErrorField: "allow_private_targets",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertBrowserAuthSubmission(t, tt)
		})
	}
}

func assertBrowserAuthSubmission(t *testing.T, tt browserAuthSubmissionTestCase) {
	t.Helper()

	server, _, publisher := newTestServer(t)
	server.config.AllowPrivateTargets = tt.allowPrivate

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/jobs/urls/browser-auth?source=browser",
		strings.NewReader(tt.body),
	)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.Router().ServeHTTP(rr, req)

	if rr.Code != tt.wantStatus {
		t.Fatalf("expected %d, got %d: %s", tt.wantStatus, rr.Code, rr.Body.String())
	}

	if tt.forbidResponse != "" && strings.Contains(rr.Body.String(), tt.forbidResponse) {
		t.Fatalf("error response leaked rejected credentials: %s", rr.Body.String())
	}

	if tt.wantStatus == http.StatusCreated {
		assertLiteralBrowserAuthPublished(t, publisher)

		return
	}

	if len(publisher.envelopes) != 0 {
		t.Fatalf("rejected request published %d jobs", len(publisher.envelopes))
	}

	var response httputil.ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("decode error response: %v", err)
	}

	if response.Error.Field != tt.wantErrorField {
		t.Fatalf("error field = %q, want %q", response.Error.Field, tt.wantErrorField)
	}
}

func assertLiteralBrowserAuthPublished(t *testing.T, publisher *fakePublisher) {
	t.Helper()

	if len(publisher.envelopes) != 1 {
		t.Fatalf("expected one published job, got %d", len(publisher.envelopes))
	}

	created, ok := publisher.envelopes[0].Payload.(*events.JobCreatedPayload)
	if !ok || created == nil {
		t.Fatalf("published payload = %T, want *events.JobCreatedPayload", publisher.envelopes[0].Payload)
	}

	if !strings.Contains(string(created.Config.Auth), `"value":"throwaway-password"`) {
		t.Fatalf("literal form recipe was not preserved for execution: %s", created.Config.Auth)
	}
}

func TestRemovedProjectRoutesReturnNotFound(t *testing.T) {
	for _, path := range []string{"/api/v1/projects", "/api/v1/projects/demo", "/api/v1/jobs/job-123/diff"} {
		for _, key := range []string{"", "wrong-token", testAPIToken} {
			t.Run(path+" key="+key, func(t *testing.T) {
				server, _, _ := newTestServer(t)
				server.config.APIToken = testAPIToken

				rr := routeCall(t, server, http.MethodGet, path, "", key)
				if rr.Code != http.StatusNotFound {
					t.Fatalf("status = %d, want 404: %s", rr.Code, rr.Body.String())
				}
			})
		}
	}
}
