// Package podman provides a thin client for the Podman HTTP API.
package podman

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

const maxPodmanDiagnosticBytes = 256 * 1024

// Client talks to Podman over a Unix socket via its HTTP API.
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// Config configures the Podman Unix socket path.
type Config struct {
	SocketPath string
	// RequestTimeout sets a hard timeout for each Podman API call.
	RequestTimeout time.Duration
	// DialTimeout controls how long the Unix socket dial can block.
	DialTimeout time.Duration
	// ResponseHeaderTimeout limits time to the first response header byte.
	ResponseHeaderTimeout time.Duration
	// IdleConnTimeout controls keepalive idle lifetime.
	IdleConnTimeout time.Duration
}

// Default control-plane timeouts.
//
// Creating a pod is not one operation: Podman writes the pod record, creates and
// possibly pulls the infra container, and programs the network namespace through
// netavark. On a loaded host that legitimately runs past ten seconds, and when it
// did in production the lost response left a pod behind that every retry then
// collided with. Kubelet's equivalent runtime-request-timeout defaults to two
// minutes for the same reason.
//
// RequestTimeout caps the whole request, so it has to move with the header
// timeout -- raising only the latter changes nothing but the error text. It also
// bounds how long one hung socket can hold up a reconcile tick.
const (
	defaultRequestTimeout        = 120 * time.Second
	defaultDialTimeout           = 5 * time.Second
	defaultResponseHeaderTimeout = 60 * time.Second
	defaultIdleConnTimeout       = 90 * time.Second
)

// DefaultConfig returns the standard rootless Podman socket path.
func DefaultConfig() *Config {
	return &Config{
		SocketPath:            "/run/podman/podman.sock",
		RequestTimeout:        defaultRequestTimeout,
		DialTimeout:           defaultDialTimeout,
		ResponseHeaderTimeout: defaultResponseHeaderTimeout,
		IdleConnTimeout:       defaultIdleConnTimeout,
	}
}

// libpodPrefix pins the Libpod API version. The host runs Podman 5.
const libpodPrefix = "/v5.0.0/libpod"

// NewClient builds a Podman HTTP client over a Unix socket.
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if config.RequestTimeout <= 0 {
		config.RequestTimeout = defaultRequestTimeout
	}

	if config.DialTimeout <= 0 {
		config.DialTimeout = defaultDialTimeout
	}

	if config.ResponseHeaderTimeout <= 0 {
		config.ResponseHeaderTimeout = defaultResponseHeaderTimeout
	}

	if config.IdleConnTimeout <= 0 {
		config.IdleConnTimeout = defaultIdleConnTimeout
	}

	dialer := &net.Dialer{
		Timeout:   config.DialTimeout,
		KeepAlive: 30 * time.Second,
	}

	defaultTransport := &http.Transport{
		ResponseHeaderTimeout: config.ResponseHeaderTimeout,
		IdleConnTimeout:       config.IdleConnTimeout,
		MaxIdleConns:          20,
		MaxIdleConnsPerHost:   10,
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return dialer.DialContext(ctx, "unix", config.SocketPath)
		},
	}

	return &Client{
		httpClient: &http.Client{Timeout: config.RequestTimeout, Transport: defaultTransport},
		baseURL:    "http://unix",
	}, nil
}

// APIError preserves the Podman HTTP status so idempotent runtime operations
// can distinguish a missing resource or create race from an infrastructure
// failure without parsing error strings.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error (status %d): %s", e.StatusCode, e.Body)
}

func isAPIStatus(err error, status int) bool {
	var apiErr *APIError

	return errors.As(err, &apiErr) && apiErr.StatusCode == status
}

// IsNotFound reports whether err is Podman's 404. Exported because callers
// outside this package delete by name and need "it was already gone" to count as
// success rather than as a failure that aborts the rest of a cleanup.
func IsNotFound(err error) bool {
	return isAPIStatus(err, http.StatusNotFound)
}

// doLibpodRequest sends a JSON request to a Libpod endpoint.
func (c *Client) doLibpodRequest(ctx context.Context, method, suffix string, body any) (*http.Response, error) {
	if suffix == "" {
		return nil, errors.New("libpod request suffix is required")
	}

	if !strings.HasPrefix(suffix, "/") {
		suffix = "/" + suffix
	}

	var reqBody io.Reader

	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}

		reqBody = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+libpodPrefix+suffix, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

func parseResponse(resp *http.Response, target any) error {
	if resp.StatusCode >= 400 {
		body, truncated, readErr := readBoundedPodmanBody(resp.Body)
		if readErr != nil {
			return fmt.Errorf("API error (status %d): failed to read response body: %w", resp.StatusCode, readErr)
		}

		if truncated {
			body = append([]byte("[truncated] "), body...)
		}

		return &APIError{StatusCode: resp.StatusCode, Body: string(body)}
	}

	if target != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

func readBoundedPodmanBody(r io.Reader) ([]byte, bool, error) {
	const limit = int64(maxPodmanDiagnosticBytes)

	body, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, false, err
	}

	if int64(len(body)) <= limit {
		return body, false, nil
	}

	return body[:limit], true, nil
}
