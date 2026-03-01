package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestScannersCommandSummaryAndJSON(t *testing.T) {
	response := sampleScannersResponse()

	apiBaseURL := "http://stageflow.test"

	withDefaultHTTPClient(t, &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/api/v1/scanners" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}

			return httpJSONResponse(t, http.StatusOK, response), nil
		}),
	})

	t.Run("summary", func(t *testing.T) {
		var (
			stdout bytes.Buffer
			stderr bytes.Buffer
		)

		exitCode := runScannersCommand(
			context.Background(),
			[]string{"--api", apiBaseURL},
			stubEnv,
			&stdout,
			&stderr,
		)
		if exitCode != 0 {
			t.Fatalf("exitCode = %d, want 0; stderr=%s", exitCode, stderr.String())
		}

		got := stdout.String()
		if !strings.Contains(got, "Scanners (enabled 1/2)") || !strings.Contains(got, "axe") {
			t.Fatalf("unexpected summary output: %s", got)
		}
	})

	t.Run("json", func(t *testing.T) {
		var (
			stdout bytes.Buffer
			stderr bytes.Buffer
		)

		exitCode := runScannersCommand(
			context.Background(),
			[]string{"--api", apiBaseURL, "--format", "json"},
			stubEnv,
			&stdout,
			&stderr,
		)
		if exitCode != 0 {
			t.Fatalf("exitCode = %d, want 0; stderr=%s", exitCode, stderr.String())
		}

		var payload ScannersResponse

		unmarshalErr := json.Unmarshal(stdout.Bytes(), &payload)
		if unmarshalErr != nil {
			t.Fatalf("unmarshal stdout: %v\n%s", unmarshalErr, stdout.String())
		}

		if payload.Total != 2 {
			t.Fatalf("payload.Total = %d, want 2", payload.Total)
		}
	})
}

func TestScannersCommandInvalidFormat(t *testing.T) {
	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)

	exitCode := runScannersCommand(context.Background(), []string{"--format", "yaml"}, stubEnv, &stdout, &stderr)
	if exitCode != 2 {
		t.Fatalf("exitCode = %d, want 2", exitCode)
	}
}
