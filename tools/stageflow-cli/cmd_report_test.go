package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestReportCommandValidationAndStateErrors(t *testing.T) {
	t.Run("missing job id", func(t *testing.T) {
		var (
			stdout bytes.Buffer
			stderr bytes.Buffer
		)

		exitCode := runReportCommand(context.Background(), nil, stubEnv, &stdout, &stderr)
		if exitCode != 2 {
			t.Fatalf("exitCode = %d, want 2", exitCode)
		}
	})

	t.Run("extra args", func(t *testing.T) {
		var (
			stdout bytes.Buffer
			stderr bytes.Buffer
		)

		exitCode := runReportCommand(
			context.Background(),
			[]string{"job-1", "job-2"},
			stubEnv,
			&stdout,
			&stderr,
		)
		if exitCode != 2 {
			t.Fatalf("exitCode = %d, want 2", exitCode)
		}
	})

	t.Run("running job", func(t *testing.T) {
		apiBaseURL := "http://stageflow.test"

		withDefaultHTTPClient(t, &http.Client{
			Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.Path != "/api/v1/jobs/job-1" {
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
				}

				return httpJSONResponse(t, http.StatusOK, sampleJobStatus("job-1", "SCANNING")), nil
			}),
		})

		var (
			stdout bytes.Buffer
			stderr bytes.Buffer
		)

		exitCode := runReportCommand(
			context.Background(),
			[]string{"--api", apiBaseURL, "job-1"},
			stubEnv,
			&stdout,
			&stderr,
		)
		if exitCode != 2 {
			t.Fatalf("exitCode = %d, want 2", exitCode)
		}
	})

	t.Run("failed job", func(t *testing.T) {
		apiBaseURL := "http://stageflow.test"

		withDefaultHTTPClient(t, &http.Client{
			Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.Path != "/api/v1/jobs/job-1" {
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
				}

				return httpJSONResponse(t, http.StatusOK, JobStatus{ID: "job-1", State: "FAILED", Error: "boom"}), nil
			}),
		})

		var (
			stdout bytes.Buffer
			stderr bytes.Buffer
		)

		exitCode := runReportCommand(
			context.Background(),
			[]string{"--api", apiBaseURL, "job-1"},
			stubEnv,
			&stdout,
			&stderr,
		)
		if exitCode != 2 {
			t.Fatalf("exitCode = %d, want 2", exitCode)
		}
	})
}

func TestReportCommandDoneJobJSONOutput(t *testing.T) {
	jobID := "job-report-json"
	reportDoc := sampleReport(jobID)

	apiBaseURL := "http://stageflow.test"

	withDefaultHTTPClient(t, &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			switch r.URL.Path {
			case "/api/v1/jobs/" + jobID:
				return httpJSONResponse(t, http.StatusOK, sampleJobStatus(jobID, "DONE")), nil
			case "/api/v1/jobs/" + jobID + "/results":
				return httpJSONResponse(t, http.StatusOK, reportDoc), nil
			default:
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
				return nil, errors.New("unreachable")
			}
		}),
	})

	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)

	exitCode := runReportCommand(context.Background(), []string{
		"--api", apiBaseURL,
		"--format", "json",
		"--severity", "info",
		jobID,
	}, stubEnv, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0; stderr=%s", exitCode, stderr.String())
	}

	var payload struct {
		JobID string `json:"job_id"`
	}

	unmarshalErr := json.Unmarshal(stdout.Bytes(), &payload)
	if unmarshalErr != nil {
		t.Fatalf("unmarshal stdout: %v\n%s", unmarshalErr, stdout.String())
	}

	if payload.JobID != jobID {
		t.Fatalf("payload.JobID = %q, want %q", payload.JobID, jobID)
	}

	if strings.Contains(stderr.String(), "Failed") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}

	t.Run("accepts job id before flags", func(t *testing.T) {
		stdout.Reset()
		stderr.Reset()

		exitCode2 := runReportCommand(context.Background(), []string{
			jobID,
			"--api", apiBaseURL,
			"--format", "json",
			"--severity", "info",
		}, stubEnv, &stdout, &stderr)

		if exitCode2 != 0 {
			t.Fatalf("exitCode = %d, want 0; stderr=%s", exitCode2, stderr.String())
		}

		var payload2 struct {
			JobID string `json:"job_id"`
		}

		unmarshalErr2 := json.Unmarshal(stdout.Bytes(), &payload2)
		if unmarshalErr2 != nil {
			t.Fatalf("unmarshal stdout: %v\n%s", unmarshalErr2, stdout.String())
		}

		if payload2.JobID != jobID {
			t.Fatalf("payload.JobID = %q, want %q", payload2.JobID, jobID)
		}
	})
}
