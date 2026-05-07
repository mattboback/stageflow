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
	"sync"
	"time"
)

const maxPodmanDiagnosticBytes = 256 * 1024

// Client talks to Podman over a Unix socket via its HTTP API.
type Client struct {
	httpClient         *http.Client
	longPollHTTPClient *http.Client
	baseURL            string
	apiPrefix          string
	apiMu              sync.RWMutex
}

// Config configures the Podman Unix socket path.
type Config struct {
	SocketPath string
	// APIPrefix overrides the API prefix used for Libpod endpoints, e.g. "/v5.0.0/libpod".
	// When empty, the client defaults to "/v4.0.0/libpod" and falls back between v4 and v5 on 404s.
	APIPrefix string
	// RequestTimeout sets a hard timeout for standard Podman API calls.
	// Long-poll endpoints (for example container wait) intentionally bypass this timeout.
	RequestTimeout time.Duration
	// DialTimeout controls how long the Unix socket dial can block.
	DialTimeout time.Duration
	// ResponseHeaderTimeout limits time to first response header byte for standard requests.
	// Long-poll endpoints intentionally bypass this timeout.
	ResponseHeaderTimeout time.Duration
	// IdleConnTimeout controls keepalive idle lifetime.
	IdleConnTimeout time.Duration
}

// DefaultConfig returns the standard rootless Podman socket path.
func DefaultConfig() *Config {
	return &Config{
		SocketPath:            "/run/podman/podman.sock",
		RequestTimeout:        30 * time.Second,
		DialTimeout:           5 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		IdleConnTimeout:       90 * time.Second,
	}
}

const (
	libpodV4Prefix = "/v4.0.0/libpod"
	libpodV5Prefix = "/v5.0.0/libpod"
)

// NewClient builds a Podman HTTP client over a Unix socket.
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if config.RequestTimeout <= 0 {
		config.RequestTimeout = 30 * time.Second
	}

	if config.DialTimeout <= 0 {
		config.DialTimeout = 5 * time.Second
	}

	if config.ResponseHeaderTimeout <= 0 {
		config.ResponseHeaderTimeout = 10 * time.Second
	}

	if config.IdleConnTimeout <= 0 {
		config.IdleConnTimeout = 90 * time.Second
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

	// Create HTTP clients that connect to the Unix socket.
	// Wait/long-poll endpoints must not use hard HTTP-level timeouts.
	httpClient := &http.Client{
		Timeout:   config.RequestTimeout,
		Transport: defaultTransport,
	}

	longPollTransport := defaultTransport.Clone()
	longPollTransport.ResponseHeaderTimeout = 0

	longPollHTTPClient := &http.Client{
		Transport: longPollTransport,
	}

	apiPrefix := libpodV4Prefix
	if config.APIPrefix != "" {
		apiPrefix = config.APIPrefix
	}

	return &Client{
		httpClient:         httpClient,
		longPollHTTPClient: longPollHTTPClient,
		baseURL:            "http://unix",
		apiPrefix:          apiPrefix,
	}, nil
}

func (c *Client) currentAPIPrefix() string {
	c.apiMu.RLock()
	defer c.apiMu.RUnlock()

	if c.apiPrefix == "" {
		return libpodV4Prefix
	}

	return c.apiPrefix
}

func (c *Client) setAPIPrefix(prefix string) {
	c.apiMu.Lock()
	c.apiPrefix = prefix
	c.apiMu.Unlock()
}

// doLibpodRequest sends a request to a Libpod endpoint, automatically falling back between v4 and v5
// when the server returns a 404 for the chosen API prefix.
func (c *Client) doLibpodRequest(ctx context.Context, method, suffix string, body any) (*http.Response, error) {
	return c.doLibpodRequestWithOptions(ctx, method, suffix, body, requestOptions{})
}

// doLibpodLongPollRequest sends a request for long-poll endpoints (e.g. container wait) without
// HTTP-level request/response-header timeouts that would prematurely terminate healthy waits.
func (c *Client) doLibpodLongPollRequest(ctx context.Context, method, suffix string, body any) (*http.Response, error) {
	return c.doLibpodRequestWithOptions(ctx, method, suffix, body, requestOptions{longPoll: true})
}

type requestOptions struct {
	longPoll bool
}

func (c *Client) doLibpodRequestWithOptions(
	ctx context.Context,
	method, suffix string,
	body any,
	options requestOptions,
) (*http.Response, error) {
	if suffix == "" {
		return nil, errors.New("libpod request suffix is required")
	}

	if !strings.HasPrefix(suffix, "/") {
		suffix = "/" + suffix
	}

	prefix := c.currentAPIPrefix()

	resp, err := c.doRequest(ctx, method, prefix+suffix, body, options)
	if err != nil {
		return nil, err
	}

	// If the chosen prefix yields a 404, retry with the alternate prefix and remember it if it works.
	//nolint:nestif // API version fallback logic requires nested conditional handling
	if resp.StatusCode == http.StatusNotFound {
		var alt string

		switch prefix {
		case libpodV4Prefix:
			alt = libpodV5Prefix
		case libpodV5Prefix:
			alt = libpodV4Prefix
		}

		if alt != "" {
			_ = resp.Body.Close()

			resp2, err2 := c.doRequest(ctx, method, alt+suffix, body, options)
			if err2 != nil {
				return nil, err2
			}

			if resp2.StatusCode != http.StatusNotFound {
				c.setAPIPrefix(alt)
			}

			return resp2, nil
		}
	}

	return resp, nil
}

func (c *Client) doRequest(
	ctx context.Context,
	method, path string,
	body any,
	options requestOptions,
) (*http.Response, error) {
	var reqBody io.Reader

	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}

		reqBody = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	httpClient := c.httpClient
	if options.longPoll && c.longPollHTTPClient != nil {
		httpClient = c.longPollHTTPClient
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

func parseResponse(resp *http.Response, target any) error {
	if resp.StatusCode >= 400 {
		body, truncated, readErr := readBoundedPodmanBody(resp.Body, maxPodmanDiagnosticBytes)
		if readErr != nil {
			return fmt.Errorf("API error (status %d): failed to read response body: %w", resp.StatusCode, readErr)
		}

		if truncated {
			body = append([]byte("[truncated] "), body...)
		}

		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	if target != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

func readBoundedPodmanBody(r io.Reader, limit int64) ([]byte, bool, error) {
	if limit < 1 {
		return nil, false, errors.New("podman response limit must be positive")
	}

	body, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, false, err
	}

	if int64(len(body)) <= limit {
		return body, false, nil
	}

	return body[:limit], true, nil
}

type tailBuffer struct {
	limit int
	buf   []byte
}

func newTailBuffer(limit int64) (*tailBuffer, error) {
	if limit < 1 {
		return nil, errors.New("tail buffer limit must be positive")
	}

	if int64(int(limit)) != limit {
		return nil, errors.New("tail buffer limit is too large")
	}

	return &tailBuffer{limit: int(limit)}, nil
}

func (b *tailBuffer) Write(p []byte) (int, error) {
	written := len(p)
	if len(p) >= b.limit {
		b.buf = append(b.buf[:0], p[len(p)-b.limit:]...)
		return written, nil
	}

	b.buf = append(b.buf, p...)
	if overflow := len(b.buf) - b.limit; overflow > 0 {
		copy(b.buf, b.buf[overflow:])
		b.buf = b.buf[:b.limit]
	}

	return written, nil
}

func readPodmanTail(r io.Reader, limit int64) (string, bool, error) {
	buf, err := newTailBuffer(limit)
	if err != nil {
		return "", false, err
	}

	n, err := io.Copy(buf, r)
	if err != nil {
		return "", false, err
	}

	return string(buf.buf), n > limit, nil
}
