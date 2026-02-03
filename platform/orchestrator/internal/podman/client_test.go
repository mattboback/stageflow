package podman

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type mockPodmanServer struct {
	handlers map[string]http.HandlerFunc
	URL      string
}

func newMockPodmanServer() *mockPodmanServer {
	return &mockPodmanServer{
		handlers: make(map[string]http.HandlerFunc),
		URL:      "http://podman",
	}
}

func (m *mockPodmanServer) handle(method, path string, handler http.HandlerFunc) {
	m.handlers[method+" "+path] = handler
}

func (m *mockPodmanServer) Close() {}

func (m *mockPodmanServer) RoundTrip(req *http.Request) (*http.Response, error) {
	key := req.Method + " " + req.URL.Path

	handler, ok := m.handlers[key]
	if !ok {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(strings.NewReader("not found")),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	}

	rec := httptest.NewRecorder()
	handler(rec, req)
	resp := rec.Result()
	resp.Request = req

	return resp, nil
}

func (m *mockPodmanServer) newClient() *Client {
	return &Client{
		httpClient: &http.Client{Transport: m},
		baseURL:    m.URL,
	}
}

func TestNewClient(t *testing.T) {
	config := &Config{
		SocketPath: "/run/podman/podman.sock",
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client == nil {
		t.Fatal("Expected client to be non-nil")
	}
	if client.httpClient == nil {
		t.Error("Expected httpClient to be non-nil")
	}
	if client.baseURL != "http://unix" {
		t.Errorf("Expected baseURL http://unix, got %s", client.baseURL)
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.SocketPath != "/run/podman/podman.sock" {
		t.Errorf("Expected default socket path /run/podman/podman.sock, got %s", config.SocketPath)
	}
}

func TestParseResponse_Success(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       http.NoBody,
	}
	defer func() { _ = resp.Body.Close() }()

	err := parseResponse(resp, nil)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestParseResponse_Error(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader("internal server error")),
		Header:     make(http.Header),
	}
	defer func() { _ = resp.Body.Close() }()
	if err := parseResponse(resp, nil); err == nil {
		t.Error("Expected error for status 500")
	}
}

func TestParseResponse_JSON(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"id":"test-123"}`)),
		Header:     make(http.Header),
	}
	defer func() { _ = resp.Body.Close() }()

	var result map[string]string
	if err := parseResponse(resp, &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if result["id"] != "test-123" {
		t.Errorf("Expected id test-123, got %s", result["id"])
	}
}
