package podman

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type mockPodmanServer struct {
	handlers map[string]http.HandlerFunc
	URL      string
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
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

func TestNewClient_LongPollClientDisablesHTTPTimeouts(t *testing.T) {
	config := &Config{
		SocketPath:            "/run/podman/podman.sock",
		RequestTimeout:        30 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client.httpClient.Timeout != config.RequestTimeout {
		t.Errorf("Expected regular client timeout %v, got %v", config.RequestTimeout, client.httpClient.Timeout)
	}

	if client.longPollHTTPClient == nil {
		t.Fatal("Expected longPollHTTPClient to be non-nil")
	}

	if client.longPollHTTPClient.Timeout != 0 {
		t.Errorf("Expected long-poll client timeout 0, got %v", client.longPollHTTPClient.Timeout)
	}

	regularTransport, ok := client.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("Expected regular transport to be *http.Transport, got %T", client.httpClient.Transport)
	}

	if regularTransport.ResponseHeaderTimeout != config.ResponseHeaderTimeout {
		t.Errorf(
			"Expected regular response header timeout %v, got %v",
			config.ResponseHeaderTimeout,
			regularTransport.ResponseHeaderTimeout,
		)
	}

	longPollTransport, ok := client.longPollHTTPClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("Expected long-poll transport to be *http.Transport, got %T", client.longPollHTTPClient.Transport)
	}

	if longPollTransport.ResponseHeaderTimeout != 0 {
		t.Errorf("Expected long-poll response header timeout 0, got %v", longPollTransport.ResponseHeaderTimeout)
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

func TestReadBoundedPodmanBodyTruncatesOversizedBody(t *testing.T) {
	t.Parallel()

	body := strings.NewReader(strings.Repeat("x", maxPodmanDiagnosticBytes) + "overflow")

	got, truncated, err := readBoundedPodmanBody(body)
	if err != nil {
		t.Fatalf("readBoundedPodmanBody: %v", err)
	}

	if !truncated {
		t.Fatal("expected oversized body to be marked truncated")
	}

	if len(got) != maxPodmanDiagnosticBytes {
		t.Fatalf("body length = %d, want %d", len(got), maxPodmanDiagnosticBytes)
	}

	if string(got) != strings.Repeat("x", maxPodmanDiagnosticBytes) {
		t.Fatal("expected bounded body to contain the leading diagnostic bytes")
	}
}

func TestReadPodmanTailKeepsOnlyFinalLinesWhenOversized(t *testing.T) {
	t.Parallel()

	want := "keep-1\nkeep-2\n"
	body := strings.NewReader("drop-1\ndrop-2\n" + want)

	got, truncated, err := readPodmanTail(body, int64(len(want)))
	if err != nil {
		t.Fatalf("readPodmanTail: %v", err)
	}

	if !truncated {
		t.Fatal("expected oversized log body to be marked truncated")
	}

	if got != want {
		t.Fatalf("tail = %q, want %q", got, want)
	}
}
