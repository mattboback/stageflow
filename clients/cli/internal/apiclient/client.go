package apiclient

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

func (c *Client) BuildURL(apiPath string) (*url.URL, error) {
	parsedBase, err := url.Parse(c.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid API URL %q: %w", c.BaseURL, err)
	}

	pathURL, err := url.Parse(apiPath)
	if err != nil {
		return nil, fmt.Errorf("invalid path %q: %w", apiPath, err)
	}

	return parsedBase.ResolveReference(pathURL), nil
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if c.APIKey != "" {
		req.Header.Set("X-Api-Key", c.APIKey)
	}

	return c.HTTPClient.Do(req)
}

func (c *Client) GetJSON(ctx context.Context, apiPath string, dest any) error {
	reqURL, err := c.BuildURL(apiPath)
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

func marshalRequestBody(body any) (io.Reader, error) {
	if body == nil {
		return http.NoBody, nil
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	return bytes.NewReader(payload), nil
}

func decodeJSONResponse(resp *http.Response, dest any) error {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, err := readResponseBody(resp.Body)
		if err != nil {
			return err
		}

		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	if dest == nil {
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

func (c *Client) SendJSON(ctx context.Context, method, apiPath string, body any, dest any) error {
	reqURL, err := c.BuildURL(apiPath)
	if err != nil {
		return err
	}

	reqBody, err := marshalRequestBody(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL.String(), reqBody)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	return decodeJSONResponse(resp, dest)
}

func (c *Client) PostJSON(ctx context.Context, apiPath string, body any, dest any) error {
	return c.SendJSON(ctx, http.MethodPost, apiPath, body, dest)
}

func (c *Client) PatchJSON(ctx context.Context, apiPath string, body any, dest any) error {
	return c.SendJSON(ctx, http.MethodPatch, apiPath, body, dest)
}

func (c *Client) DeleteJSON(ctx context.Context, apiPath string) error {
	reqURL, err := c.BuildURL(apiPath)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, reqURL.String(), http.NoBody)
	if err != nil {
		return err
	}

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

	return nil
}

func readResponseBody(reader io.Reader) ([]byte, error) {
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, nil
}
