package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)

	exitCode := run([]string{"stageflow", "version"}, stubEnv, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0; stderr=%s", exitCode, stderr.String())
	}

	if got := strings.TrimSpace(stdout.String()); got == "" {
		t.Fatalf("expected version output, got empty")
	}
}
