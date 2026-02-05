package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func httpClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
	}
}

func waitForAPI(t *testing.T) {
	t.Helper()

	timeout := time.NewTimer(30 * time.Second)
	defer timeout.Stop()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	client := httpClient()

	for {
		select {
		case <-timeout.C:
			t.Fatal("Timed out waiting for API to be ready")
		case <-ticker.C:
			if isAPIReady(t, client) {
				return
			}
		}
	}
}

func isAPIReady(t *testing.T, client *http.Client) bool {
	t.Helper()

	healthReq, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		apiBaseURL+"/healthz",
		http.NoBody,
	)
	if err != nil {
		return false
	}

	healthResp, healthErr := client.Do(healthReq)
	if healthErr == nil {
		_ = healthResp.Body.Close()

		if healthResp.StatusCode == http.StatusOK {
			return true
		}
	}

	zipReq, err := http.NewRequestWithContext(context.Background(), http.MethodGet, apiBaseURL+"/jobs/zip", http.NoBody)
	if err != nil {
		return false
	}

	resp, err := client.Do(zipReq)
	if err != nil {
		return false
	}

	_ = resp.Body.Close()

	// /jobs/zip may be POST-only; GET can yield 405 or 404 depending on router wiring.
	return resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode == http.StatusNotFound
}

func createTestZip(t *testing.T) string {
	t.Helper()

	fixturePath, err := filepath.Abs("../fixtures/test-site.zip")
	if err != nil {
		t.Fatalf("Failed to resolve fixture zip path: %v", err)
	}

	if _, statErr := os.Stat(fixturePath); statErr != nil {
		t.Fatalf("Test data test-site.zip not found at %s: %v", fixturePath, statErr)
	}

	tempPath := filepath.Join(t.TempDir(), "test-site.zip")

	src, err := os.Open(filepath.Clean(fixturePath))
	if err != nil {
		t.Fatalf("Failed to open fixture zip: %v", err)
	}

	defer func() { _ = src.Close() }()

	dst, err := os.Create(filepath.Clean(tempPath))
	if err != nil {
		t.Fatalf("Failed to create temp zip: %v", err)
	}

	defer func() { _ = dst.Close() }()

	if _, copyErr := io.Copy(dst, src); copyErr != nil {
		t.Fatalf("Failed to copy fixture zip: %v", copyErr)
	}

	return tempPath
}

func uploadZip(t *testing.T, zipPath string) string {
	t.Helper()

	file, err := os.Open(filepath.Clean(zipPath))
	if err != nil {
		t.Fatalf("Failed to open zip: %v", err)
	}

	defer func() { _ = file.Close() }()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filepath.Base(zipPath))
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}

	if _, copyErr := io.Copy(part, file); copyErr != nil {
		t.Fatalf("Failed to copy file content: %v", copyErr)
	}

	if closeErr := writer.Close(); closeErr != nil {
		t.Fatalf("Failed to close writer: %v", closeErr)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, apiBaseURL+"/jobs/zip", body)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("Failed to upload zip: %v", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK &&
		resp.StatusCode != http.StatusCreated {
		bodyBytes, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			t.Fatalf("Failed to read upload error body: %v", readErr)
		}

		t.Fatalf("Upload failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		JobID string `json:"job_id"`
	}
	if decodeErr := json.NewDecoder(resp.Body).Decode(&result); decodeErr != nil {
		t.Fatalf("Failed to decode response: %v", decodeErr)
	}

	return result.JobID
}

func waitForJobCompletion(t *testing.T, jobID string) {
	t.Helper()

	timeout := time.NewTimer(2 * time.Minute)
	defer timeout.Stop()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	client := httpClient()

	for {
		select {
		case <-timeout.C:
			t.Fatal("Timed out waiting for job completion")
		case <-ticker.C:
			status, err := fetchJobStatus(client, jobID)
			if err != nil {
				t.Logf("Failed to get job status: %v", err)

				continue
			}

			t.Logf("Job state: %s", status.State)

			if isDoneState(status.State) {
				return
			}

			if isFailedState(status.State) {
				t.Fatalf("Job failed: %s", status.Error)
			}
		}
	}
}

type jobStatusResponse struct {
	State string `json:"state"`
	Error string `json:"error,omitempty"`
}

func fetchJobStatus(client *http.Client, jobID string) (*jobStatusResponse, error) {
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		fmt.Sprintf("%s/jobs/%s", apiBaseURL, jobID),
		http.NoBody,
	)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() { _ = resp.Body.Close() }()

	var status jobStatusResponse
	if decodeErr := json.NewDecoder(resp.Body).Decode(&status); decodeErr != nil {
		return nil, decodeErr
	}

	return &status, nil
}

func isDoneState(state string) bool {
	switch strings.ToLower(state) {
	case "done", "complete":
		return true
	default:
		return false
	}
}

func isFailedState(state string) bool {
	return strings.EqualFold(state, "failed")
}

func verifyResults(t *testing.T, jobID string) {
	t.Helper()

	checkClient := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	checkEndpoint := func(name, endpoint string) {
		t.Helper()

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, endpoint, http.NoBody)
		if err != nil {
			t.Fatalf("Failed to create %s request: %v", name, err)
		}

		resp, err := checkClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to get %s: %v", name, err)
		}

		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusTemporaryRedirect && resp.StatusCode != http.StatusFound {
			if resp.StatusCode != http.StatusOK {
				t.Errorf("%s endpoint returned %d", name, resp.StatusCode)
			}

			return
		}

		location := resp.Header.Get("Location")
		if location == "" {
			t.Fatalf("%s returned redirect but no Location header", name)
		}

		// Rewrite minio:9000 to localhost:9000 for local testing.
		location = strings.Replace(location, "minio:9000", "localhost:9000", 1)
		t.Logf("Redirecting to: %s", location)

		redirectReq, err := http.NewRequestWithContext(context.Background(), http.MethodGet, location, http.NoBody)
		if err != nil {
			t.Fatalf("Failed to create request for %s: %v", name, err)
		}
		// Force Host header to match the signed URL (minio:9000).
		redirectReq.Host = "minio:9000"

		resp2, err := httpClient().Do(redirectReq)
		if err != nil {
			t.Fatalf("Failed to follow redirect for %s: %v", name, err)
		}

		defer func() { _ = resp2.Body.Close() }()

		if resp2.StatusCode != http.StatusOK {
			t.Errorf("%s redirect returned %d", name, resp2.StatusCode)
		}
	}

	checkEndpoint("Report", fmt.Sprintf("%s/jobs/%s/report", apiBaseURL, jobID))
	checkEndpoint("Results", fmt.Sprintf("%s/jobs/%s/results", apiBaseURL, jobID))
}
