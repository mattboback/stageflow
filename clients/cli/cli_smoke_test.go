package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
	"github.com/mattboback/stageflow/clients/cli/internal/projectmode"
	"github.com/mattboback/stageflow/clients/cli/internal/projectscan"
	"github.com/mattboback/stageflow/clients/cli/internal/testsupport"
	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
	"github.com/mattboback/stageflow/libs/go/diff"
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
	testsupport.RequireNoErr(t, err)

	if !strings.Contains(string(data), projectmode.DefaultScanScanners) {
		t.Fatalf("expected %s to contain current default scanner list %q", docPath, projectmode.DefaultScanScanners)
	}
}

func TestToolsReadmeUsesCurrentDefaultScannerList(t *testing.T) {
	repoRoot := mustFindRepoRoot(t)
	readmePath := filepath.Join(repoRoot, "clients", "cli", "README.md")

	data, err := os.ReadFile(readmePath)
	testsupport.RequireNoErr(t, err)

	scannerListYAML := "scanners: [" + strings.Join(projectmode.DefaultScanScannerList(), ", ") + "]"
	if !strings.Contains(string(data), scannerListYAML) {
		t.Fatalf("expected %s to contain current default scanner list %q", readmePath, scannerListYAML)
	}
}

func TestRepoProjectBootstrapSuggestionUsesLocalSelfScanDefaults(t *testing.T) {
	repoRoot := mustFindRepoRoot(t)
	suggestion, err := projectmode.DetectBootstrapSuggestion(repoRoot)
	testsupport.RequireNoErr(t, err)

	if suggestion.IsPlaceholder {
		t.Fatalf("expected repo bootstrap suggestion, got placeholder")
	}

	template := projectmode.DefaultConfigTemplate("http://localhost:8080", suggestion)
	for _, expected := range []string{
		`cmd: ["just", "run", "clients/web"]`,
		`url: http://127.0.0.1:5173`,
	} {
		if !strings.Contains(template, expected) {
			t.Fatalf("expected template to contain %q", expected)
		}
	}
}

func TestCLIDevCommandsSmoke(t *testing.T) {
	root := t.TempDir()

	err := os.WriteFile(filepath.Join(root, "justfile"), []byte("run SERVICE:\n\t@true\n"), 0o600)
	testsupport.RequireNoErr(t, err)

	err = os.MkdirAll(filepath.Join(root, "clients", "web"), 0o750)
	testsupport.RequireNoErr(t, err)

	err = os.WriteFile(
		filepath.Join(root, "clients", "web", "package.json"),
		[]byte(`{"scripts":{"dev":"vite dev"}}`),
		0o600,
	)
	testsupport.RequireNoErr(t, err)

	err = os.WriteFile(filepath.Join(root, "clients", "web", "vite.config.ts"), []byte("export default {}\n"), 0o600)
	testsupport.RequireNoErr(t, err)

	stdout, stderr, exitCode := runCLI(t, "stageflow", "dev", "init", root)
	if exitCode != 0 {
		t.Fatalf("exitCode=%d stderr=%q", exitCode, stderr)
	}

	if !strings.Contains(stdout, "Created StageFlow dev-loop bootstrap") {
		t.Fatalf("unexpected init output: %q", stdout)
	}

	data, err := os.ReadFile(filepath.Join(root, ".stageflow", "config.yaml"))
	testsupport.RequireNoErr(t, err)

	contents := string(data)
	if !strings.Contains(contents, `cmd: ["just", "run", "clients/web"]`) {
		t.Fatalf("expected inferred command in config: %q", contents)
	}

	if !strings.Contains(contents, "url: http://127.0.0.1:5173") {
		t.Fatalf("expected inferred URL in config: %q", contents)
	}
}

func TestCLIDevDoctorRepoSmoke(t *testing.T) {
	repoDir := os.Getenv("STAGEFLOW_TEST_REPO_DIR")
	if repoDir == "" {
		t.Skip("skipping: set STAGEFLOW_TEST_REPO_DIR to the repo root")
	}

	if _, err := os.Stat(repoDir); err != nil {
		t.Skipf("skipping: %s not found", repoDir)
	}

	cleanup := withWorkingDir(t, repoDir)
	defer cleanup()

	stdout, stderr, exitCode := runCLI(t, "stageflow", "dev", "doctor", "--skip-dev")
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

func TestCLIRemoteProjectScanJSONEnvelope(t *testing.T) {
	server, jobID := newCLIProjectScanAPIServer(t)
	defer server.Close()

	stdout, stderr, exitCode := runCLI(
		t,
		"stageflow",
		"--api",
		server.URL,
		"--format",
		"json",
		"project",
		"scan",
		"demo-site",
		"--no-stream",
		"--summary-only",
	)
	if exitCode != 1 {
		t.Fatalf("exitCode=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
	}

	if !strings.Contains(stderr, "Waiting for completion") {
		t.Fatalf("unexpected scan stderr: %q", stderr)
	}

	decoder := json.NewDecoder(strings.NewReader(stdout))

	var payload projectscan.Envelope
	testsupport.RequireNoErr(t, decoder.Decode(&payload))

	testsupport.RequireEqual(t, payload.Schema, "stageflow-cli/project-scan@v1", "payload.Schema")
	testsupport.RequireEqual(t, payload.Project.Slug, "demo-site", "payload.Project.Slug")
	testsupport.RequireEqual(
		t,
		payload.Project.Baseline.Status,
		projectscan.BaselineStatusAvailable,
		"payload.Project.Baseline.Status",
	)
	testsupport.RequireEqual(t, payload.Decision.Regressed, true, "payload.Decision.Regressed")
	testsupport.RequireEqual(t, payload.Decision.Passed, false, "payload.Decision.Passed")
	testsupport.RequireEqual(t, payload.Report.Schema, "stageflow-cli/report@v1", "payload.Report.Schema")

	if payload.Diff == nil {
		t.Fatalf("expected diff payload")
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		t.Fatalf("expected a single JSON document, got err=%v", err)
	}

	testsupport.RequireEqual(t, payload.Report.Job.ID, jobID, "payload.Report.Job.ID")
}

func TestCLIProjectScanFromConfigJSONEnvelope(t *testing.T) {
	server, jobID := newCLIProjectScanAPIServer(t)
	defer server.Close()

	root := t.TempDir()
	testsupport.WriteProjectConfig(t, root, `version: 2
stageflow:
  project: demo-site
`)

	cleanup := withWorkingDir(t, root)
	defer cleanup()

	stdout, stderr, exitCode := runCLI(
		t,
		"stageflow",
		"--api",
		server.URL,
		"--format",
		"json",
		"project",
		"scan",
		"--no-stream",
		"--summary-only",
	)
	if exitCode != 1 {
		t.Fatalf("exitCode=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
	}

	if !strings.Contains(stderr, `Using project "demo-site" from`) {
		t.Fatalf("unexpected project scan stderr: %q", stderr)
	}

	if !strings.Contains(stderr, "Waiting for completion") {
		t.Fatalf("unexpected project scan stderr: %q", stderr)
	}

	decoder := json.NewDecoder(strings.NewReader(stdout))

	var payload projectscan.Envelope
	testsupport.RequireNoErr(t, decoder.Decode(&payload))

	testsupport.RequireEqual(t, payload.Schema, "stageflow-cli/project-scan@v1", "payload.Schema")
	testsupport.RequireEqual(t, payload.Project.Slug, "demo-site", "payload.Project.Slug")
	testsupport.RequireEqual(t, payload.Report.Job.ID, jobID, "payload.Report.Job.ID")
	testsupport.RequireEqual(t, payload.Decision.Regressed, true, "payload.Decision.Regressed")

	if payload.Diff == nil {
		t.Fatalf("expected diff payload")
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

func mustFindRepoRoot(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	testsupport.RequireNoErr(t, err)

	root, ok, err := projectmode.FindGitRoot(wd)
	testsupport.RequireNoErr(t, err)

	if !ok {
		t.Fatalf("expected to find git root from %s", wd)
	}

	return root
}

func newCLISmokeAPIServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()

	jobID := "job-smoke"
	reportDoc := testsupport.SampleReport(jobID)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/jobs/urls":
			writeJSONResponse(t, w, http.StatusAccepted, apiclient.SubmitJobResponse{JobID: jobID, Status: "accepted"})
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

func newCLIProjectScanAPIServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()

	jobID := "job-project"
	currentDoc := testsupport.SampleReport(jobID)
	baselineDoc := testsupport.SampleReport("job-baseline")
	baselineDoc.Issues = []report.IssueDetail{baselineDoc.Issues[1]}
	baselineScore := 84
	baselineDoc.Summary.Score = &baselineScore
	baselineDoc.Summary.TotalIssues = len(baselineDoc.Issues)
	diffResult := diff.ComputeDiff("job-baseline", baselineDoc, jobID, currentDoc)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/projects/demo-site/scan":
			writeJSONResponse(t, w, http.StatusAccepted, apiclient.SubmitJobResponse{JobID: jobID, Status: "accepted"})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/jobs/"+jobID:
			writeJSONResponse(t, w, http.StatusOK, apiclient.JobStatus{
				ID:                jobID,
				State:             apiclient.JobStateDone,
				ExpectedScanners:  []string{"axe"},
				CompletedScanners: []string{"axe"},
				CreatedAt:         time.Unix(1700001000, 0).UTC(),
				UpdatedAt:         time.Unix(1700001030, 0).UTC(),
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/jobs/"+jobID+"/results":
			writeJSONResponse(t, w, http.StatusOK, currentDoc)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/jobs/"+jobID+"/diff":
			writeJSONResponse(t, w, http.StatusOK, struct {
				diff.Result
				Project struct {
					Slug string `json:"slug"`
					Name string `json:"name"`
				} `json:"project"`
			}{
				Result: diffResult,
				Project: struct {
					Slug string `json:"slug"`
					Name string `json:"name"`
				}{
					Slug: "demo-site",
					Name: "Demo Site",
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
