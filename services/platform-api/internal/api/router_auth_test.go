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

func TestRouterRequiresAPIKeyAcrossProtectedAndEdgeAuthenticatedRoutes(t *testing.T) {
	t.Setenv("PLATFORM_API_TOKEN", "edge-token")
	t.Setenv("PLATFORM_API_AUTH_DISABLED", "false")

	server, _, _ := newTestServer(t)
	router := server.Router()

	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{
			name:   "caller-authenticated URL intake",
			method: http.MethodPost,
			path:   "/api/v1/jobs/urls",
			body:   `{"urls":["https://example.com"]}`,
		},
		{
			name:   "anonymous URL intake",
			method: http.MethodPost,
			path:   "/api/v1/jobs/urls/anonymous",
			body:   `{"urls":["https://example.com"]}`,
		},
		{
			name:   "browser auth URL intake",
			method: http.MethodPost,
			path:   "/api/v1/jobs/urls/browser-auth",
			body:   literalBrowserAuthBody,
		},
		{
			name:   "anonymous trailing slash alias",
			method: http.MethodPost,
			path:   "/api/v1/jobs/urls/anonymous/",
			body:   `{}`,
		},
		{
			name:   "browser auth trailing slash alias",
			method: http.MethodPost,
			path:   "/api/v1/jobs/urls/browser-auth/",
			body:   `{}`,
		},
		{name: "projects", method: http.MethodGet, path: "/api/v1/projects"},
		{name: "job delete", method: http.MethodDelete, path: "/api/v1/jobs/job-123"},
		{
			name:   "project baseline diff",
			method: http.MethodGet,
			path:   "/api/v1/jobs/job-123/diff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			if tt.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("expected missing API key to return 401, got %d: %s", rr.Code, rr.Body.String())
			}
		})
	}
}

func TestAnonymousURLSubmissionAcceptsOnlyPublicUnauthenticatedJobs(t *testing.T) {
	t.Setenv("PLATFORM_API_TOKEN", "edge-token")
	t.Setenv("PLATFORM_API_AUTH_DISABLED", "false")
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
			req.Header.Set("X-Api-Key", "edge-token")

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
	t.Setenv("PLATFORM_API_TOKEN", "edge-token")
	t.Setenv("PLATFORM_API_AUTH_DISABLED", "false")
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
			req.Header.Set("X-Api-Key", "edge-token")

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
	t.Setenv("PLATFORM_API_TOKEN", "edge-token")
	t.Setenv("PLATFORM_API_AUTH_DISABLED", "false")
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
	req.Header.Set("X-Api-Key", "edge-token")

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

func TestCallerAuthenticatedURLSubmissionRetainsEnvironmentReferences(t *testing.T) {
	t.Setenv("PLATFORM_API_TOKEN", "caller-token")
	t.Setenv("PLATFORM_API_AUTH_DISABLED", "false")

	server, _, publisher := newTestServer(t)
	body := `{
		"urls":["https://example.com/dashboard"],
		"auth":{"mode":"form","form":{
			"login_url":"https://example.com/login",
			"steps":[{"type":"fill","selector":"#password","value":{"from_env":"DEMO_PASSWORD"}}],
			"success":{"type":"url_matches","pattern":"/dashboard"}
		}}
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer caller-token")

	rr := httptest.NewRecorder()

	server.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	if len(publisher.envelopes) != 1 {
		t.Fatalf("expected one published job, got %d", len(publisher.envelopes))
	}

	created := publisher.envelopes[0].Payload.(*events.JobCreatedPayload)
	if !strings.Contains(string(created.Config.Auth), `"from_env":"DEMO_PASSWORD"`) {
		t.Fatalf("caller-authenticated recipe lost environment reference: %s", created.Config.Auth)
	}
}
