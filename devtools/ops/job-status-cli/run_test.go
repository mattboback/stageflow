package main

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestRunEventsCommand(t *testing.T) {
	now := time.Now().UTC()

	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodGet {
				t.Fatalf("unexpected method: %s", req.Method)
			}

			if req.URL.Path != "/api/v1/jobs/job-123/events" {
				t.Fatalf("unexpected path: %s", req.URL.Path)
			}

			body := `{
  "job_id": "job-123",
  "events": [
    {
      "ID": 1,
      "JobID": "job-123",
      "Event": "job.created",
      "Timestamp": "` + now.Format(time.RFC3339Nano) + `",
      "Payload": "{\"input_type\":\"zip\"}",
      "Producer": "platform-api",
      "HandlerStatus": "ok",
      "NATSDeliveries": 1,
      "NATSStreamSeq": 42
    }
  ],
  "limit": 500,
  "offset": 0
}`

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exit := run(
		[]string{"job-status-cli", "events", "job-123"},
		func(key string) string {
			if key == "ORCHESTRATOR_ADMIN_URL" {
				return "http://example.test"
			}

			return ""
		},
		client,
		stdout,
		stderr,
	)

	if exit != 0 {
		t.Fatalf("expected exit 0, got %d (stderr=%q)", exit, stderr.String())
	}

	if !strings.Contains(stdout.String(), "job.created") {
		t.Fatalf("expected output to include event name, got:\n%s", stdout.String())
	}
}

func TestRunEventsCommand_MissingJobID(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exit := run(
		[]string{"job-status-cli", "events"},
		func(string) string { return "" },
		http.DefaultClient,
		stdout,
		stderr,
	)

	if exit == 0 {
		t.Fatalf("expected non-zero exit, got %d", exit)
	}
}
