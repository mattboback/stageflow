package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func performGET(
	ctx context.Context,
	client *http.Client,
	requestURL *url.URL,
	apiToken string,
) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), http.NoBody)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(apiToken) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiToken))
	}

	// #nosec G107 -- URL is validated by buildAPIURL and limited to http/https schemes.
	return client.Do(req)
}

func buildAPIURL(base, apiPath string, query url.Values) (*url.URL, error) {
	parsedBase, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Errorf("invalid API URL %q: %w", base, err)
	}

	if parsedBase.Scheme != "http" && parsedBase.Scheme != "https" {
		return nil, fmt.Errorf("invalid API URL scheme %q", parsedBase.Scheme)
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

func decodeOKJSON(ctx context.Context, client *http.Client, requestURL *url.URL, apiToken string, dest any) error {
	resp, err := performGET(ctx, client, requestURL, apiToken)
	if err != nil {
		return fmt.Errorf("failed to connect to API: %w", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("API returned status %d (failed to read body: %w)", resp.StatusCode, readErr)
		}

		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if decodeErr := json.NewDecoder(resp.Body).Decode(dest); decodeErr != nil {
		return fmt.Errorf("failed to parse response: %w", decodeErr)
	}

	return nil
}
