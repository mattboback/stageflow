package main

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunCommandSuccessViaSSE(t *testing.T) {
	jobID := "job-run-success"
	reportDoc := sampleReport(jobID)

	apiBaseURL := "http://stageflow.test"

	withDefaultHTTPClient(t, &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			switch {
			case r.Method == http.MethodPost && r.URL.Path == "/api/v1/jobs/urls":
				return httpJSONResponse(t, http.StatusCreated, SubmitJobResponse{JobID: jobID}), nil
			case r.Method == http.MethodGet && r.URL.Path == "/api/v1/jobs/"+jobID+"/stream":
				return httpTextResponse(
					http.StatusOK,
					"text/event-stream",
					"event: done\ndata: {\"state\":\"DONE\"}\n\n",
				), nil
			case r.Method == http.MethodGet && r.URL.Path == "/api/v1/jobs/"+jobID:
				return httpJSONResponse(t, http.StatusOK, sampleJobStatus(jobID, "DONE")), nil
			case r.Method == http.MethodGet && r.URL.Path == "/api/v1/jobs/"+jobID+"/results":
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

	exitCode := runRunCommand(context.Background(), []string{
		"--api",
		apiBaseURL,
		"--url",
		"https://example.com",
		"--severity",
		"info",
	}, stubEnv, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0; stderr=%s", exitCode, stderr.String())
	}

	if got := stdout.String(); !strings.Contains(got, "Job ID: "+jobID) {
		t.Fatalf("stdout missing job id: %s", got)
	}

	if got := stderr.String(); !strings.Contains(got, "Job submitted: "+jobID) {
		t.Fatalf("stderr missing submission message: %s", got)
	}
}

func TestRunCommandThresholdFailure(t *testing.T) {
	jobID := "job-run-threshold"
	reportDoc := sampleReport(jobID)

	apiBaseURL := "http://stageflow.test"

	withDefaultHTTPClient(t, &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			switch {
			case r.Method == http.MethodPost && r.URL.Path == "/api/v1/jobs/urls":
				return httpJSONResponse(t, http.StatusCreated, SubmitJobResponse{JobID: jobID}), nil
			case r.Method == http.MethodGet && r.URL.Path == "/api/v1/jobs/"+jobID+"/stream":
				return httpTextResponse(
					http.StatusOK,
					"text/event-stream",
					"event: done\ndata: {\"state\":\"DONE\"}\n\n",
				), nil
			case r.Method == http.MethodGet && r.URL.Path == "/api/v1/jobs/"+jobID:
				return httpJSONResponse(t, http.StatusOK, sampleJobStatus(jobID, "DONE")), nil
			case r.Method == http.MethodGet && r.URL.Path == "/api/v1/jobs/"+jobID+"/results":
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

	exitCode := runRunCommand(context.Background(), []string{
		"--api",
		apiBaseURL,
		"--url",
		"https://example.com",
		"--severity",
		"info",
		"--threshold-critical",
		"0",
	}, stubEnv, &stdout, &stderr)

	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1; stderr=%s", exitCode, stderr.String())
	}

	if got := stdout.String(); !strings.Contains(got, "Thresholds: FAIL") {
		t.Fatalf("stdout missing threshold failure: %s", got)
	}
}

func TestRunCommandNoStreamPolling(t *testing.T) {
	jobID := "job-run-poll"
	reportDoc := sampleReport(jobID)

	var statusCalls atomic.Int32

	apiBaseURL := "http://stageflow.test"

	withDefaultHTTPClient(t, &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			switch {
			case r.Method == http.MethodPost && r.URL.Path == "/api/v1/jobs/urls":
				return httpJSONResponse(t, http.StatusCreated, SubmitJobResponse{JobID: jobID}), nil
			case r.Method == http.MethodGet && r.URL.Path == "/api/v1/jobs/"+jobID:
				call := statusCalls.Add(1)
				if call == 1 {
					return httpJSONResponse(t, http.StatusOK, sampleJobStatus(jobID, "SCANNING")), nil
				}

				return httpJSONResponse(t, http.StatusOK, sampleJobStatus(jobID, "DONE")), nil
			case r.Method == http.MethodGet && r.URL.Path == "/api/v1/jobs/"+jobID+"/results":
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

	exitCode := runRunCommand(context.Background(), []string{
		"--api",
		apiBaseURL,
		"--url",
		"https://example.com",
		"--severity",
		"info",
		"--no-stream",
	}, stubEnv, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0; stderr=%s", exitCode, stderr.String())
	}

	if statusCalls.Load() < 2 {
		t.Fatalf("expected at least 2 status calls, got %d", statusCalls.Load())
	}
}

//nolint:gocyclo
func TestRunCommandTimeoutAppliesToFinalFetches(t *testing.T) {
	jobID := "job-run-timeout-final-fetch"

	statusStarted := make(chan struct{}, 1)
	statusCanceled := make(chan struct{}, 1)

	apiBaseURL := "http://stageflow.test"

	withDefaultHTTPClient(t, &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			switch {
			case r.Method == http.MethodPost && r.URL.Path == "/api/v1/jobs/urls":
				return httpJSONResponse(t, http.StatusCreated, SubmitJobResponse{JobID: jobID}), nil
			case r.Method == http.MethodGet && r.URL.Path == "/api/v1/jobs/"+jobID+"/stream":
				return httpTextResponse(
					http.StatusOK,
					"text/event-stream",
					"event: done\ndata: {\"state\":\"DONE\"}\n\n",
				), nil
			case r.Method == http.MethodGet && r.URL.Path == "/api/v1/jobs/"+jobID:
				select {
				case statusStarted <- struct{}{}:
				default:
				}

				<-r.Context().Done()

				select {
				case statusCanceled <- struct{}{}:
				default:
				}

				return nil, r.Context().Err()
			case r.Method == http.MethodGet && r.URL.Path == "/api/v1/jobs/"+jobID+"/results":
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
				return nil, errors.New("unreachable")
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

	exitCodeCh := make(chan int, 1)

	go func() {
		exitCodeCh <- runRunCommand(context.Background(), []string{
			"--api",
			apiBaseURL,
			"--url",
			"https://example.com",
			"--severity",
			"info",
			"--timeout",
			"100ms",
		}, stubEnv, &stdout, &stderr)
	}()

	select {
	case <-statusStarted:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for final status request to start")
	}

	var exitCode int
	select {
	case exitCode = <-exitCodeCh:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for run command to exit")
	}

	if exitCode != 2 {
		t.Fatalf("exitCode = %d, want 2; stderr=%s", exitCode, stderr.String())
	}

	if got := stderr.String(); !strings.Contains(got, "context deadline exceeded") {
		t.Fatalf("stderr missing timeout error: %s", got)
	}

	select {
	case <-statusCanceled:
	default:
		t.Fatalf("expected final status request to be canceled by --timeout")
	}
}

func TestRunCommandNoStreamReturnsPollingErrors(t *testing.T) {
	jobID := "job-run-poll-error"

	apiBaseURL := "http://stageflow.test"

	withDefaultHTTPClient(t, &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			switch {
			case r.Method == http.MethodPost && r.URL.Path == "/api/v1/jobs/urls":
				return httpJSONResponse(t, http.StatusCreated, SubmitJobResponse{JobID: jobID}), nil
			case r.Method == http.MethodGet && r.URL.Path == "/api/v1/jobs/"+jobID:
				return httpTextResponse(http.StatusInternalServerError, "text/plain; charset=utf-8", "boom\n"), nil
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

	exitCode := runRunCommand(context.Background(), []string{
		"--api",
		apiBaseURL,
		"--url",
		"https://example.com",
		"--severity",
		"info",
		"--no-stream",
		"--timeout",
		"50ms",
	}, stubEnv, &stdout, &stderr)

	if exitCode != 2 {
		t.Fatalf("exitCode = %d, want 2; stderr=%s", exitCode, stderr.String())
	}

	if got := stderr.String(); !strings.Contains(got, "API request failed with status 500") {
		t.Fatalf("stderr missing underlying polling error: %s", got)
	}
}

func TestRunCommandValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "missing url",
			args: []string{"--format", "summary"},
		},
		{
			name: "invalid format",
			args: []string{"--url", "https://example.com", "--format", "xml"},
		},
		{
			name: "invalid severity",
			args: []string{"--url", "https://example.com", "--severity", "blocker"},
		},
		{
			name: "negative max",
			args: []string{"--url", "https://example.com", "--max", "-1"},
		},
		{
			name: "empty scanner module",
			args: []string{"--url", "https://example.com", "--scanners", "axe,"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				stdout bytes.Buffer
				stderr bytes.Buffer
			)

			exitCode := runRunCommand(context.Background(), tt.args, stubEnv, &stdout, &stderr)
			if exitCode != 2 {
				t.Fatalf("exitCode = %d, want 2; stderr=%s", exitCode, stderr.String())
			}
		})
	}
}
