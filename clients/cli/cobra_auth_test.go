package main

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/mattboback/stageflow/clients/cli/internal/exitcode"
)

func TestAuthCaptureRequiresOutput(t *testing.T) {
	stdout, stderr, exitCode := runCLI(
		t,
		"stageflow",
		"auth",
		"capture",
		"https://example.com/login",
	)
	if exitCode == 0 {
		t.Fatalf("expected non-zero exit when --output omitted; stdout=%q stderr=%q", stdout, stderr)
	}

	if !strings.Contains(stderr, "required flag(s) \"output\" not set") &&
		!strings.Contains(stderr, "--output is required") {
		t.Fatalf("expected output-required error in stderr; got %q", stderr)
	}
}

func TestAuthCaptureRequiresURL(t *testing.T) {
	tmp := t.TempDir()
	out := filepath.Join(tmp, "state.json")

	stdout, stderr, exitCode := runCLI(
		t,
		"stageflow",
		"auth",
		"capture",
		"--output",
		out,
	)
	if exitCode == 0 {
		t.Fatalf("expected non-zero exit when URL omitted; stdout=%q stderr=%q", stdout, stderr)
	}

	if !strings.Contains(stderr, "accepts 1 arg") {
		t.Fatalf("expected arg-count error; got %q", stderr)
	}
}

func TestAuthCaptureWritesFileWithMode0600(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("0600 mode test is POSIX-specific")
	}

	tmp := t.TempDir()
	out := filepath.Join(tmp, "nested", "state.json")

	stubRunner := func(_ *cobra.Command, url, output string, _ []string) error {
		if url != "https://app.example.com/login" {
			return errors.New("unexpected url passed to runner: " + url)
		}

		// Simulate the file `npx playwright open --save-storage` would write,
		// with permissive mode the CLI must tighten to 0600.
		return os.WriteFile(output, []byte(`{"cookies":[],"origins":[]}`), 0o644)
	}

	cmd := newAuthCaptureCmd(stubRunner)

	rootCmd := &cobra.Command{Use: "stageflow"}
	rootCmd.AddCommand(cmd)
	rootCmd.SetArgs([]string{"capture", "--output", out, "https://app.example.com/login"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	info, err := os.Stat(out)
	if err != nil {
		t.Fatalf("expected output file: %v", err)
	}

	if got, want := info.Mode().Perm(), os.FileMode(0o600); got != want {
		t.Fatalf("output file mode = %o, want %o", got, want)
	}
}

func TestAuthCaptureFailsOnEmptyFile(t *testing.T) {
	tmp := t.TempDir()
	out := filepath.Join(tmp, "state.json")

	stubRunner := func(_ *cobra.Command, _, output string, _ []string) error {
		return os.WriteFile(output, []byte{}, 0o600)
	}

	cmd := newAuthCaptureCmd(stubRunner)
	rootCmd := &cobra.Command{Use: "stageflow"}
	rootCmd.AddCommand(cmd)
	rootCmd.SetArgs([]string{"capture", "--output", out, "https://app.example.com/login"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for empty storage-state file")
	}

	if !strings.Contains(err.Error(), "empty storage-state") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAuthCaptureFailsWhenRunnerErrors(t *testing.T) {
	tmp := t.TempDir()
	out := filepath.Join(tmp, "state.json")

	stubRunner := func(_ *cobra.Command, _, _ string, _ []string) error {
		return exitcode.Error{Code: 2, Err: errors.New("boom")}
	}

	cmd := newAuthCaptureCmd(stubRunner)
	rootCmd := &cobra.Command{Use: "stageflow"}
	rootCmd.AddCommand(cmd)
	rootCmd.SetArgs([]string{"capture", "--output", out, "https://app.example.com/login"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error from runner")
	}

	var exitErr exitcode.Error
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected exitcode.Error; got %v", err)
	}

	if exitErr.Code != 2 {
		t.Fatalf("exit code = %d, want 2", exitErr.Code)
	}
}
