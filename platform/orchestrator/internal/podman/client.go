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
)

// Client talks to Podman over a Unix socket via its HTTP API.
type Client struct {
	httpClient *http.Client
	baseURL    string
	apiPrefix  string
	apiMu      sync.RWMutex
}

// Config configures the Podman Unix socket path.
type Config struct {
	SocketPath string
	// APIPrefix overrides the API prefix used for Libpod endpoints, e.g. "/v5.0.0/libpod".
	// When empty, the client defaults to "/v4.0.0/libpod" and falls back between v4 and v5 on 404s.
	APIPrefix string
}

// DefaultConfig returns the standard rootless Podman socket path.
func DefaultConfig() *Config {
	return &Config{
		SocketPath: "/run/podman/podman.sock",
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

	// Create HTTP client that connects to Unix socket
	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "unix", config.SocketPath)
			},
		},
	}

	apiPrefix := libpodV4Prefix
	if config.APIPrefix != "" {
		apiPrefix = config.APIPrefix
	}

	return &Client{
		httpClient: httpClient,
		baseURL:    "http://unix",
		apiPrefix:  apiPrefix,
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
	if suffix == "" {
		return nil, errors.New("libpod request suffix is required")
	}

	if !strings.HasPrefix(suffix, "/") {
		suffix = "/" + suffix
	}

	prefix := c.currentAPIPrefix()

	resp, err := c.doRequest(ctx, method, prefix+suffix, body)
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

			resp2, err2 := c.doRequest(ctx, method, alt+suffix, body)
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

func (c *Client) doRequest(ctx context.Context, method, path string, body any) (*http.Response, error) {
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

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

func parseResponse(resp *http.Response, target any) error {
	if resp.StatusCode >= 400 {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("API error (status %d): failed to read response body: %w", resp.StatusCode, readErr)
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
