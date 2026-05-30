package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiffLiveTarget_NormalizesAndAllowsPrivateTargetsForLocalAPI(t *testing.T) {
	server, captured, _ := newCapturingScanAPI(t)
	defer server.Close()

	baselinePath := writeDiffBaselineReport(t)

	stdout, stderr, exitCode := runCLI(
		t,
		"stageflow",
		"--api",
		server.URL,
		"diff",
		baselinePath,
		"127.0.0.1:5173",
		"--no-stream",
	)
	if exitCode != 0 {
		t.Fatalf("exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
	}

	if !strings.Contains(stderr, "Detected private/loopback targets") {
		t.Fatalf("expected private-target notice in stderr, got %q", stderr)
	}

	captured.mu.Lock()
	defer captured.mu.Unlock()

	requireEqual(t, captured.hits, 1, "captured.hits")
	requireDeepEqual(t, captured.body.URLs, []string{"http://127.0.0.1:5173"}, "captured.body.URLs")
	requireEqual(t, captured.body.AllowPrivateTargets, true, "captured.body.AllowPrivateTargets")
}

func TestDiffLiveTarget_RejectsPrivateTargetsForRemoteAPI(t *testing.T) {
	baselinePath := writeDiffBaselineReport(t)

	for _, target := range []string{
		"http://127.0.0.1:5173",
		"http://10.0.0.42:3000",
	} {
		t.Run(target, func(t *testing.T) {
			stdout, stderr, exitCode := runCLI(
				t,
				"stageflow",
				"--api",
				"https://stageflow.example",
				"diff",
				baselinePath,
				target,
				"--no-stream",
			)
			if exitCode != 2 {
				t.Fatalf("exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
			}

			if !strings.Contains(stderr, "refusing to submit private/loopback targets") {
				t.Fatalf("expected private-target refusal in stderr, got %q", stderr)
			}
		})
	}
}

func TestDiffCurrentTarget_PrefersExistingSchemelessFile(t *testing.T) {
	dir := t.TempDir()
	cleanup := withWorkingDir(t, dir)
	defer cleanup()

	baselinePath := writeDiffBaselineReport(t)
	writeDiffReportFile(t, filepath.Join(dir, "example.com"), "job-current")

	stdout, stderr, exitCode := runCLI(t, "stageflow", "diff", baselinePath, "example.com")
	if exitCode != 0 {
		t.Fatalf("exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
	}

	if strings.Contains(stderr, "Job submitted") {
		t.Fatalf("expected existing file to be used without live scan, got stderr=%q", stderr)
	}
}

func writeDiffBaselineReport(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "baseline.json")
	writeDiffReportFile(t, path, "job-baseline")

	return path
}

func writeDiffReportFile(t *testing.T, path, jobID string) {
	t.Helper()

	env := reportEnvelope{
		Schema: "stageflow-cli/report@v1",
		Job:    jobMeta{ID: jobID, State: jobStateDone},
		Report: sampleReport(jobID),
	}

	data, err := json.Marshal(env)
	requireNoErr(t, err)

	err = os.WriteFile(path, data, 0o600)
	requireNoErr(t, err)
}
