package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

var defaultHTTPClient = http.DefaultClient

func NewClient(baseURL, apiKey string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = defaultHTTPClient
	}

	return &Client{
		BaseURL:    baseURL,
		APIKey:     apiKey,
		HTTPClient: httpClient,
	}
}

func (c *Client) buildURL(apiPath string, query url.Values) (*url.URL, error) {
	parsedBase, err := url.Parse(c.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid API URL %q: %w", c.BaseURL, err)
	}

	pathURL, err := url.Parse(apiPath)
	if err != nil {
		return nil, fmt.Errorf("invalid path %q: %w", apiPath, err)
	}

	resolved := parsedBase.ResolveReference(pathURL)
	if query != nil {
		resolved.RawQuery = query.Encode()
	}

	return resolved, nil
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if c.APIKey != "" {
		req.Header.Set("X-Api-Key", c.APIKey)
	}

	return c.HTTPClient.Do(req)
}

func (c *Client) getJSON(ctx context.Context, apiPath string, dest any) error {
	reqURL, err := c.buildURL(apiPath, nil)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), http.NoBody)
	if err != nil {
		return err
	}

	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, readErr := readResponseBody(resp.Body)
		if readErr != nil {
			return readErr
		}

		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	decodeErr := json.NewDecoder(resp.Body).Decode(dest)
	if decodeErr != nil {
		return fmt.Errorf("failed to decode response: %w", decodeErr)
	}

	return nil
}

func (c *Client) postJSON(ctx context.Context, apiPath string, body any, dest any) error {
	reqURL, err := c.buildURL(apiPath, nil)
	if err != nil {
		return err
	}

	var reqBody io.Reader

	if body != nil {
		payload, marshalErr := json.Marshal(body)
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal request body: %w", marshalErr)
		}

		reqBody = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL.String(), reqBody)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, readErr := readResponseBody(resp.Body)
		if readErr != nil {
			return readErr
		}

		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	if dest != nil {
		decodeErr := json.NewDecoder(resp.Body).Decode(dest)
		if decodeErr != nil {
			return fmt.Errorf("failed to decode response: %w", decodeErr)
		}
	}

	return nil
}

func readResponseBody(reader io.Reader) ([]byte, error) {
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, nil
}
