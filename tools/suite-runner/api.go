package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

func submitJob(
	ctx context.Context,
	client *http.Client,
	apiURL string,
	domain string,
	modules []string,
	screenshot bool,
) (string, error) {
	payload := map[string]interface{}{
		"urls":       []string{domain},
		"modules":    modules,
		"screenshot": screenshot,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		apiURL+"/api/v1/jobs/urls",
		bytes.NewReader(body),
	)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		b, readErr := readBody(resp)
		if readErr != nil {
			return "", readErr
		}

		return "", fmt.Errorf("submit failed (%d): %s", resp.StatusCode, string(b))
	}

	var sr submitResponse
	if decodeErr := json.NewDecoder(resp.Body).Decode(&sr); decodeErr != nil {
		return "", decodeErr
	}

	return sr.JobID, nil
}

func fetchStatus(ctx context.Context, client *http.Client, apiURL, jobID string) (jobStatus, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		apiURL+"/api/v1/jobs/"+jobID,
		http.NoBody,
	)
	if err != nil {
		return jobStatus{}, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return jobStatus{}, err
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		b, readErr := readBody(resp)
		if readErr != nil {
			return jobStatus{}, readErr
		}

		return jobStatus{}, fmt.Errorf("status failed (%d): %s", resp.StatusCode, string(b))
	}

	var st jobStatus
	if decodeErr := json.NewDecoder(resp.Body).Decode(&st); decodeErr != nil {
		return jobStatus{}, decodeErr
	}

	return st, nil
}

func fetchResults(
	ctx context.Context,
	client *http.Client,
	artifacts *artifactPayload,
) (resultsSummary, error) {
	if artifacts == nil || artifacts.ResultsJSON == "" {
		return resultsSummary{}, errors.New("results_json missing")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, artifacts.ResultsJSON, http.NoBody)
	if err != nil {
		return resultsSummary{}, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return resultsSummary{}, err
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		b, readErr := readBody(resp)
		if readErr != nil {
			return resultsSummary{}, readErr
		}

		return resultsSummary{}, fmt.Errorf("results fetch failed (%d): %s", resp.StatusCode, string(b))
	}

	var rs resultsSummary
	if decodeErr := json.NewDecoder(resp.Body).Decode(&rs); decodeErr != nil {
		return resultsSummary{}, decodeErr
	}

	if rs.Summary.ByImpact == nil {
		rs.Summary.ByImpact = map[string]int{}
	}

	return rs, nil
}

func readBody(resp *http.Response) ([]byte, error) {
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	return data, nil
}
