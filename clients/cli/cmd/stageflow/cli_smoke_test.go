package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
	"github.com/mattboback/stageflow/clients/cli/internal/testsupport"
)

func TestCLIHelpAndVersionSmoke(t *testing.T) {
	stdout, stderr, exitCode := runCLI(t, "stageflow", "--help")
	if exitCode != 0 {
		t.Fatalf("exitCode=%d stderr=%q", exitCode, stderr)
	}

	if !strings.Contains(stdout, "StageFlow CLI") {
		t.Fatalf("unexpected help output: %q", stdout)
	}

	stdout, stderr, exitCode = runCLI(t, "stageflow", "version")
	if exitCode != 0 {
		t.Fatalf("exitCode=%d stderr=%q", exitCode, stderr)
	}

	if strings.TrimSpace(stdout) == "" {
		t.Fatalf("expected version output")
	}
}

func TestCLICompletionSmoke(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		t.Run(shell, func(t *testing.T) {
			stdout, stderr, exitCode := runCLI(t, "stageflow", "completion", shell)
			if exitCode != 0 {
				t.Fatalf("exitCode=%d stderr=%q", exitCode, stderr)
			}

			if strings.TrimSpace(stdout) == "" {
				t.Fatalf("expected completion output for %s", shell)
			}
		})
	}
}

func TestCLIRemovedCommandsAreRejected(t *testing.T) {
	for _, command := range []string{"dev", "stack", "docs"} {
		t.Run(command, func(t *testing.T) {
			_, stderr, code := runCLI(t, "stageflow", command)
			if code != 2 || !strings.Contains(stderr, "unknown command") {
				t.Fatalf("removed command returned exitCode=%d stderr=%q", code, stderr)
			}
		})
	}
}

func TestCLIRemovedNoStreamFlagIsRejected(t *testing.T) {
	for _, args := range [][]string{
		{"stageflow", "scan", "https://example.com", "--no-stream"},
		{"stageflow", "diff", "baseline.json", "current.json", "--no-stream"},
	} {
		t.Run(args[1], func(t *testing.T) {
			_, stderr, code := runCLI(t, args...)
			if code != 2 || !strings.Contains(stderr, "unknown flag: --no-stream") {
				t.Fatalf("removed flag returned exitCode=%d stderr=%q", code, stderr)
			}
		})
	}
}

func TestCLIApiCommandsSmoke(t *testing.T) {
	server, jobID := newCLISmokeAPIServer(t)
	defer server.Close()

	stdout, stderr, exitCode := runCLI(t, "stageflow", "--api", server.URL, "scanners")
	if exitCode != 0 {
		t.Fatalf("exitCode=%d stderr=%q", exitCode, stderr)
	}

	if !strings.Contains(stdout, "Scanners (enabled 1/2)") {
		t.Fatalf("unexpected scanners output: %q", stdout)
	}

	stdout, stderr, exitCode = runCLI(t, "stageflow", "--api", server.URL, "report", jobID, "--summary-only")
	if exitCode != 0 {
		t.Fatalf("exitCode=%d stderr=%q", exitCode, stderr)
	}

	if !strings.Contains(stdout, "Job: "+jobID) {
		t.Fatalf("unexpected report output: %q", stdout)
	}

	stdout, stderr, exitCode = runCLI(
		t,
		"stageflow",
		"--api",
		server.URL,
		"scan",
		"https://example.com",
		"--summary-only",
	)
	if exitCode != 0 {
		t.Fatalf("exitCode=%d stderr=%q", exitCode, stderr)
	}

	if !strings.Contains(stderr, "Waiting for completion") {
		t.Fatalf("unexpected scan stderr: %q", stderr)
	}

	if !strings.Contains(stderr, "stream lost, falling back to polling...") {
		t.Fatalf("expected automatic polling fallback after rejected SSE request: %q", stderr)
	}

	if !strings.Contains(stdout, "Job: "+jobID) {
		t.Fatalf("unexpected scan output: %q", stdout)
	}
}

func runCLI(t *testing.T, args ...string) (string, string, int) {
	t.Helper()

	var stdout bytes.Buffer

	var stderr bytes.Buffer

	exitCode := run(args, testsupport.StubEnv, &stdout, &stderr)

	return stdout.String(), stderr.String(), exitCode
}

func withWorkingDir(t *testing.T, dir string) func() {
	t.Helper()

	wd, getwdErr := os.Getwd()
	if getwdErr != nil {
		t.Fatalf("getwd: %v", getwdErr)
	}

	chdirErr := os.Chdir(dir)
	if chdirErr != nil {
		t.Fatalf("chdir %s: %v", dir, chdirErr)
	}

	return func() {
		restoreErr := os.Chdir(wd)
		if restoreErr != nil {
			t.Fatalf("restore wd %s: %v", wd, restoreErr)
		}
	}
}

func newCLISmokeAPIServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()

	jobID := "job-smoke"
	reportDoc := testsupport.SampleReport(jobID)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/jobs/urls":
			writeJSONResponse(t, w, http.StatusAccepted, apiclient.SubmitJobResponse{JobID: jobID, Status: "accepted"})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/jobs/"+jobID+"/stream":
			http.NotFound(w, r)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/jobs/"+jobID:
			writeJSONResponse(t, w, http.StatusOK, apiclient.JobStatus{
				ID:                jobID,
				State:             apiclient.JobStateDone,
				ExpectedScanners:  []string{"axe", "lighthouse"},
				CompletedScanners: []string{"axe", "lighthouse"},
				CreatedAt:         time.Unix(1700000000, 0).UTC(),
				UpdatedAt:         time.Unix(1700000030, 0).UTC(),
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/jobs/"+jobID+"/results":
			writeJSONResponse(t, w, http.StatusOK, reportDoc)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/scanners":
			writeJSONResponse(t, w, http.StatusOK, apiclient.ScannersResponse{
				Total:   2,
				Enabled: 1,
				Scanners: []apiclient.ScannerInfo{
					{
						ID:         "axe",
						Name:       "Axe",
						Enabled:    true,
						Categories: []string{"accessibility"},
						Version:    "1.0.0",
						BuiltIn:    true,
					},
					{
						ID:         "seo",
						Name:       "SEO",
						Enabled:    false,
						Categories: []string{"quality"},
						Version:    "1.0.0",
						BuiltIn:    true,
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	})

	return httptest.NewServer(handler), jobID
}

func writeJSONResponse(t *testing.T, w http.ResponseWriter, status int, payload any) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	encodeErr := json.NewEncoder(w).Encode(payload)
	if encodeErr != nil {
		panic(fmt.Sprintf("encode response: %v", encodeErr))
	}
}
