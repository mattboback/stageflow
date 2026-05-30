package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientBuildURLResolvesAPIPaths(t *testing.T) {
	client := NewClient("https://api.example.com/root/", "", nil)

	got, err := client.BuildURL("/api/v1/projects/a%2Fb")
	if err != nil {
		t.Fatalf("BuildURL() error = %v", err)
	}

	if got.String() != "https://api.example.com/api/v1/projects/a%2Fb" {
		t.Fatalf("BuildURL() = %q", got.String())
	}
}

func TestClientDoAddsAPIKeyHeader(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://api.example.com/test", http.NoBody)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	client := NewClient("https://api.example.com", "secret", &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if got := r.Header.Get("X-Api-Key"); got != "secret" {
				t.Fatalf("X-Api-Key = %q, want secret", got)
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       http.NoBody,
				Header:     make(http.Header),
			}, nil
		}),
	})

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
}

func TestClientJSONHelpers(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
		gotBody   map[string]string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()

		if r.Body != http.NoBody {
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "", server.Client())

	var resp struct {
		OK bool `json:"ok"`
	}
	if err := client.SendJSON(
		context.Background(),
		http.MethodPatch,
		"/api/v1/projects/demo",
		map[string]string{"name": "Demo"},
		&resp,
	); err != nil {
		t.Fatalf("SendJSON() error = %v", err)
	}

	if gotMethod != http.MethodPatch || gotPath != "/api/v1/projects/demo" {
		t.Fatalf("request = %s %s", gotMethod, gotPath)
	}

	if gotBody["name"] != "Demo" || !resp.OK {
		t.Fatalf("body/response mismatch: body=%v resp=%v", gotBody, resp)
	}
}

func TestClientErrorBodies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "bad thing", http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewClient(server.URL, "", server.Client())

	var resp struct{}

	err := client.GetJSON(context.Background(), "/bad", &resp)
	if err == nil {
		t.Fatal("GetJSON() err = nil, want non-nil")
	}

	if !strings.Contains(err.Error(), "status 400") || !strings.Contains(err.Error(), "bad thing") {
		t.Fatalf("GetJSON() err = %q, want status and body", err.Error())
	}
}

func TestProjectEndpointPathEscaping(t *testing.T) {
	var gotPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(
			`{"id":"1","slug":"a/b","name":"Demo","urls":["https://example.com"],` +
				`"created_at":"2026-05-30T00:00:00Z","updated_at":"2026-05-30T00:00:00Z"}`,
		))
	}))
	defer server.Close()

	client := NewClient(server.URL, "", server.Client())

	if _, err := client.GetProject(context.Background(), "a/b"); err != nil {
		t.Fatalf("GetProject() error = %v", err)
	}

	if gotPath != "/api/v1/projects/a%2Fb" {
		t.Fatalf("path = %q, want escaped slug", gotPath)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
