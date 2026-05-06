package main

import (
	"errors"
	"testing"
)

func TestNewScanCmd_DefaultsScreenshotCaptureOn(t *testing.T) {
	cmd := newScanCmd(&rootOptions{})
	flag := cmd.Flags().Lookup("screenshot")
	if flag == nil {
		t.Fatal("missing --screenshot flag")
	}

	requireEqual(t, flag.DefValue, "true", "screenshot default")
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

	requireEqual(t, state.baseline.Status, projectBaselineStatusMissing, "state.baseline.Status")
	requireEqual(
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

	requireEqual(t, state.baseline.Status, projectBaselineStatusCurrent, "state.baseline.Status")
	requireEqual(
		t,
		state.baseline.Message,
		"This scan is the current baseline. Run a new scan to see a diff.",
		"state.baseline.Message",
	)
}
