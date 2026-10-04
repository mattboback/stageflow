package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The home page terminal prints a transcript of this command against the
// /demo reports. summarize-demo.ts captures it when the fixture is
// regenerated; this keeps it equal to what the CLI prints, so a change to the
// diff output cannot leave the home page showing an older format.
const demoRepoRoot = "../../../../"

func TestDemoTerminalTranscriptMatchesDiffOutput(t *testing.T) {
	var transcript struct {
		Command  string `json:"command"`
		Output   string `json:"output"`
		ExitCode int    `json:"exitCode"`
	}

	readDemoJSON(t, "clients/web/app/lib/demo/demo-terminal.json", &transcript)

	dir := t.TempDir()

	for name, source := range map[string]string{
		"baseline.json": "clients/web/public/demo/baseline.json",
		"current.json":  "clients/web/public/demo/report.json",
	} {
		var unified json.RawMessage

		readDemoJSON(t, source, &unified)

		// `stageflow diff` reads `scan --format json` envelopes.
		envelope, marshalErr := json.Marshal(map[string]any{
			"schema": "stageflow-cli/report@v1",
			"report": unified,
		})
		if marshalErr != nil {
			t.Fatalf("marshal %s envelope: %v", name, marshalErr)
		}

		writeErr := os.WriteFile(filepath.Join(dir, name), envelope, 0o600)
		if writeErr != nil {
			t.Fatalf("write %s: %v", name, writeErr)
		}
	}

	restore := withWorkingDir(t, dir)
	defer restore()

	stdout, stderr, exitCode := runCLI(t, strings.Fields(transcript.Command)...)
	if exitCode != transcript.ExitCode {
		t.Fatalf("exit code = %d, transcript says %d; stderr:\n%s", exitCode, transcript.ExitCode, stderr)
	}

	got := strings.TrimRight(stdout, "\n")
	if got != transcript.Output {
		t.Fatalf(
			"diff output drifted from the home page transcript; run just demo-fixture.\ngot:\n%s\nwant:\n%s",
			got,
			transcript.Output,
		)
	}
}

func readDemoJSON(t *testing.T, relative string, into any) {
	t.Helper()

	data, readErr := os.ReadFile(filepath.Join(demoRepoRoot, relative))
	if readErr != nil {
		t.Fatalf("read %s: %v", relative, readErr)
	}

	unmarshalErr := json.Unmarshal(data, into)
	if unmarshalErr != nil {
		t.Fatalf("parse %s: %v", relative, unmarshalErr)
	}
}
