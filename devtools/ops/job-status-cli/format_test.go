package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestRunJobsCommand_FormatsDurationsAndTruncatesErrors(t *testing.T) {
	now := time.Now().UTC()
	createdAt := now.Add(-(2*time.Hour + 15*time.Minute))
	completedAt := now.Add(-(25 * time.Minute))
	longError := "This error message is intentionally very long and should be truncated in output"

	body := fmt.Sprintf(
		`{"jobs":[{"id":"job-1","state":"DONE","input_type":"urls","input_path":"","pod_id":"","created_at":"%s","updated_at":"%s","completed_at":"%s","error":"%s"}],"total":1,"limit":20,"offset":0}`,
		createdAt.Format(time.RFC3339Nano),
		now.Format(time.RFC3339Nano),
		completedAt.Format(time.RFC3339Nano),
		longError,
	)

	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodGet {
				t.Fatalf("request method = %s, want %s", req.Method, http.MethodGet)
			}

			if req.URL.Path != "/api/v1/jobs" {
				t.Fatalf("request path = %s, want %s", req.URL.Path, "/api/v1/jobs")
			}

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
		[]string{"job-status-cli", "jobs"},
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
		t.Fatalf("run exit = %d, want %d (stderr=%q)", exit, 0, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "Jobs (showing 1 of 1 total)") {
		t.Fatalf("output missing jobs heading:\n%s", output)
	}

	if !strings.Contains(output, "2h ago") {
		t.Fatalf("output missing created duration:\n%s", output)
	}

	if !strings.Contains(output, "25m ago") {
		t.Fatalf("output missing completed duration:\n%s", output)
	}

	if strings.Contains(output, longError) {
		t.Fatalf("output should not contain full error string:\n%s", output)
	}

	if !strings.Contains(output, "This error message is") || !strings.Contains(output, "...") {
		t.Fatalf("output missing truncated error text:\n%s", output)
	}
}

func TestRunPodsCommand_FormatsMissingJobFieldsAsDash(t *testing.T) {
	body := `{"pods":[{"id":"0123456789abcdef","name":"scanner-pod","status":"running","job_id":null,"job_state":null}],"total":1}`

	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodGet {
				t.Fatalf("request method = %s, want %s", req.Method, http.MethodGet)
			}

			if req.URL.Path != "/api/v1/pods" {
				t.Fatalf("request path = %s, want %s", req.URL.Path, "/api/v1/pods")
			}

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
		[]string{"job-status-cli", "pods"},
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
		t.Fatalf("run exit = %d, want %d (stderr=%q)", exit, 0, stderr.String())
	}

	output := stdout.String()

	podLine := func() string {
		for _, line := range strings.Split(stdout.String(), "\n") {
			if strings.Contains(line, "scanner-pod") {
				return line
			}
		}

		return ""
	}()

	if podLine == "" {
		t.Fatalf("output missing pod row:\n%s", output)
	}

	fields := strings.Fields(podLine)
	if len(fields) != 5 {
		t.Fatalf("pod row field count = %d, want %d (row=%q)", len(fields), 5, podLine)
	}

	if fields[0] != "0123456789ab" {
		t.Fatalf("pod id column = %q, want %q", fields[0], "0123456789ab")
	}

	if fields[3] != "-" || fields[4] != "-" {
		t.Fatalf("job columns = %q/%q, want -/-", fields[3], fields[4])
	}
}
