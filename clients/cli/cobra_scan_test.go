package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/mattboback/stageflow/clients/cli/internal/exitcode"
	"github.com/mattboback/stageflow/clients/cli/internal/testsupport"
)

func TestNewScanCmd_DefaultsScreenshotCaptureOn(t *testing.T) {
	cmd := newScanCmd(&rootOptions{})
	flag := cmd.Flags().Lookup("screenshot")

	if flag == nil {
		t.Fatal("missing --screenshot flag")
	}

	testsupport.RequireEqual(t, flag.DefValue, "true", "screenshot default")
}

func TestInterpretProjectDiffError_MissingBaseline(t *testing.T) {
	state, matched := interpretProjectDiffError(
		"demo-site",
		"job-123",
		errors.New(`API request failed with status 404: {"error":"No baseline set for project"}`),
	)
	if !matched {
		t.Fatalf("expected missing-baseline error to match")
	}

	testsupport.RequireEqual(t, state.baseline.Status, projectBaselineStatusMissing, "state.baseline.Status")
	testsupport.RequireEqual(
		t,
		state.baseline.PromoteCommand,
		"Promote this scan: stageflow project promote demo-site --job-id job-123",
		"state.baseline.PromoteCommand",
	)
}

func TestInterpretProjectDiffError_CurrentBaseline(t *testing.T) {
	state, matched := interpretProjectDiffError(
		"demo-site",
		"job-123",
		errors.New(`API request failed with status 400: {"error":"Cannot diff against self"}`),
	)
	if !matched {
		t.Fatalf("expected current-baseline error to match")
	}

	testsupport.RequireEqual(t, state.baseline.Status, projectBaselineStatusCurrent, "state.baseline.Status")
	testsupport.RequireEqual(
		t,
		state.baseline.Message,
		"This scan is the current baseline. Run a new scan to see a diff.",
		"state.baseline.Message",
	)
}

func TestRunScanCmd_RejectsMixedPathAndURLTargets(t *testing.T) {
	dir := t.TempDir()
	testsupport.RequireNoErr(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html></html>"), 0o600))

	cmd := newScanCmd(&rootOptions{apiURL: "http://localhost:8080"})
	err := runScanCmd(
		cmd,
		&rootOptions{apiURL: "http://localhost:8080"},
		scanCommandOptions{},
		[]string{dir, "https://example.com"},
	)

	var ece exitcode.Error
	if !errors.As(err, &ece) || ece.Code != 2 {
		t.Fatalf("expected exit-code-2 error for mixed targets, got %v", err)
	}
}

func TestRunScanCmd_RejectsAuthFlagsForPathTargets(t *testing.T) {
	dir := t.TempDir()
	testsupport.RequireNoErr(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html></html>"), 0o600))

	cmd := newScanCmd(&rootOptions{apiURL: "http://localhost:8080"})
	opts := scanCommandOptions{authStatePath: "state.json"}
	err := runScanCmd(cmd, &rootOptions{apiURL: "http://localhost:8080"}, opts, []string{dir})

	var ece exitcode.Error
	if !errors.As(err, &ece) || ece.Code != 2 {
		t.Fatalf("expected exit-code-2 error for auth flags on path target, got %v", err)
	}
}
