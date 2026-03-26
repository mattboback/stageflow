package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestCLIDocsSmoke(t *testing.T) {
	outDir := t.TempDir()

	_, stderr, exitCode := runCLI(t, "stageflow", "docs", "--out-dir", outDir)
	if exitCode != 0 {
		t.Fatalf("exitCode=%d stderr=%q", exitCode, stderr)
	}

	if _, statErr := os.Stat(filepath.Join(outDir, "stageflow.md")); statErr != nil {
		t.Fatalf("expected generated docs: %v", statErr)
	}
}

func TestCheckedInCLIDocsUseCurrentDefaultScannerList(t *testing.T) {
	repoRoot := mustFindRepoRoot(t)
	docPath := filepath.Join(repoRoot, "docs", "reference", "cli", "stageflow", "stageflow_scan.md")

	data, err := os.ReadFile(docPath)
	requireNoErr(t, err)

	if !strings.Contains(string(data), defaultScanScanners) {
		t.Fatalf("expected %s to contain current default scanner list %q", docPath, defaultScanScanners)
	}
}

func TestToolsReadmeUsesCurrentDefaultScannerList(t *testing.T) {
	repoRoot := mustFindRepoRoot(t)
	readmePath := filepath.Join(repoRoot, "clients", "cli", "README.md")

	data, err := os.ReadFile(readmePath)
	requireNoErr(t, err)

	if !strings.Contains(string(data), "scanners: "+defaultScanScanners) {
		t.Fatalf("expected %s to contain current default scanner list %q", readmePath, defaultScanScanners)
	}
}

func TestCheckedInProjectConfigUsesLocalSelfScanOverrides(t *testing.T) {
	repoRoot := mustFindRepoRoot(t)
	configPath := filepath.Join(repoRoot, ".stageflow", "config.yaml")

	data, err := os.ReadFile(configPath)
	requireNoErr(t, err)

	contents := string(data)
	for _, expected := range []string{
		`cmd: ["just", "run", "clients/web"]`,
		`VITE_API_URL: http://localhost:8080`,
		`VITE_SITE_URL: http://127.0.0.1:5173`,
		`url: http://127.0.0.1:5173`,
	} {
		if !strings.Contains(contents, expected) {
			t.Fatalf("expected %s to contain %q", configPath, expected)
		}
	}
}

func TestCLIProjectCommandsSmoke(t *testing.T) {
	root := t.TempDir()

	err := os.WriteFile(filepath.Join(root, "justfile"), []byte("run SERVICE:\n\t@true\n"), 0o600)
	requireNoErr(t, err)

	err = os.MkdirAll(filepath.Join(root, "clients", "web"), 0o750)
	requireNoErr(t, err)

	err = os.WriteFile(
		filepath.Join(root, "clients", "web", "package.json"),
		[]byte(`{"scripts":{"dev":"vite dev"}}`),
		0o600,
	)
	requireNoErr(t, err)

	err = os.WriteFile(filepath.Join(root, "clients", "web", "vite.config.ts"), []byte("export default {}\n"), 0o600)
	requireNoErr(t, err)

	stdout, stderr, exitCode := runCLI(t, "stageflow", "project", "init", root)
	if exitCode != 0 {
		t.Fatalf("exitCode=%d stderr=%q", exitCode, stderr)
	}

	if !strings.Contains(stdout, "Created StageFlow project bootstrap") {
		t.Fatalf("unexpected init output: %q", stdout)
	}

	data, err := os.ReadFile(filepath.Join(root, ".stageflow", "config.yaml"))
	requireNoErr(t, err)

	contents := string(data)
	if !strings.Contains(contents, `cmd: ["just", "run", "clients/web"]`) {
		t.Fatalf("expected inferred command in config: %q", contents)
	}

	if !strings.Contains(contents, "url: http://127.0.0.1:5173") {
		t.Fatalf("expected inferred URL in config: %q", contents)
	}
}

func TestCLIProjectDoctorRepoSmoke(t *testing.T) {
	repoDir := os.Getenv("STAGEFLOW_TEST_REPO_DIR")
	if repoDir == "" {
		t.Skip("skipping: set STAGEFLOW_TEST_REPO_DIR to the repo root")
	}

	if _, err := os.Stat(repoDir); err != nil {
		t.Skipf("skipping: %s not found", repoDir)
	}

	cleanup := withWorkingDir(t, repoDir)
	defer cleanup()

	stdout, stderr, exitCode := runCLI(t, "stageflow", "project", "doctor", "--skip-dev")
	if exitCode != 0 {
		t.Fatalf("exitCode=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
	}

	if !strings.Contains(stdout, "Doctor checks passed") {
		t.Fatalf("unexpected doctor output: %q", stdout)
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
		"--no-stream",
		"--summary-only",
	)
	if exitCode != 0 {
		t.Fatalf("exitCode=%d stderr=%q", exitCode, stderr)
	}

	if !strings.Contains(stderr, "Waiting for completion") {
		t.Fatalf("unexpected scan stderr: %q", stderr)
	}

	if !strings.Contains(stdout, "Job: "+jobID) {
		t.Fatalf("unexpected scan output: %q", stdout)
	}
}

func runCLI(t *testing.T, args ...string) (string, string, int) {
	t.Helper()

	var stdout bytes.Buffer

	var stderr bytes.Buffer

	exitCode := run(args, stubEnv, &stdout, &stderr)

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

func mustFindRepoRoot(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	requireNoErr(t, err)

	root, ok, err := findGitRoot(wd)
	requireNoErr(t, err)

	if !ok {
		t.Fatalf("expected to find git root from %s", wd)
	}

	return root
}

func newCLISmokeAPIServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()

	jobID := "job-smoke"
	reportDoc := sampleReport(jobID)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/jobs/urls":
			writeJSONResponse(t, w, http.StatusAccepted, SubmitJobResponse{JobID: jobID, Status: "accepted"})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/jobs/"+jobID:
			writeJSONResponse(t, w, http.StatusOK, JobStatus{
				ID:                jobID,
				State:             jobStateDone,
				ExpectedScanners:  []string{"axe", "lighthouse"},
				CompletedScanners: []string{"axe", "lighthouse"},
				CreatedAt:         time.Unix(1700000000, 0).UTC(),
				UpdatedAt:         time.Unix(1700000030, 0).UTC(),
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/jobs/"+jobID+"/results":
			writeJSONResponse(t, w, http.StatusOK, reportDoc)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/scanners":
			writeJSONResponse(t, w, http.StatusOK, ScannersResponse{
				Total:   2,
				Enabled: 1,
				Scanners: []ScannerInfo{
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
