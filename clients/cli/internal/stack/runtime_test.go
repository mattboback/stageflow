package stack

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFakePodman writes a shell script standing in for the podman binary.
// Each invocation appends its argv (space-joined) as one line to logPath.
// exitOn maps an argv prefix (e.g. "network inspect") to the exit code that
// invocation should return; unmatched invocations exit 0.
func writeFakePodman(t *testing.T, logPath string, exitOn map[string]int) string {
	t.Helper()

	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "podman")

	var b strings.Builder

	b.WriteString("#!/bin/sh\n")
	fmt.Fprintf(&b, "echo \"$*\" >> %q\n", logPath)

	for prefix, code := range exitOn {
		fmt.Fprintf(&b, "case \"$*\" in\n  %q*) exit %d ;;\nesac\n", prefix, code)
	}

	b.WriteString("exit 0\n")

	if err := os.WriteFile(scriptPath, []byte(b.String()), 0o700); err != nil {
		t.Fatal(err)
	}

	return scriptPath
}

func readLog(t *testing.T, logPath string) []string {
	t.Helper()

	data, err := os.ReadFile(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil
	}

	return lines
}

func testOptions(t *testing.T, podmanBin string) Options {
	t.Helper()

	return Options{
		Root:      writeStageflowCheckout(t),
		Env:       EnvDev,
		Project:   "stageflow_test",
		PodmanBin: podmanBin,
	}
}

func TestUp(t *testing.T) {
	t.Run("happy path streams compose up -d", func(t *testing.T) {
		logPath := filepath.Join(t.TempDir(), "calls.log")
		podman := writeFakePodman(t, logPath, nil)
		opts := testOptions(t, podman)

		var stdout, stderr bytes.Buffer

		if err := Up(context.Background(), opts, &stdout, &stderr); err != nil {
			t.Fatalf("Up() error = %v", err)
		}

		calls := readLog(t, logPath)

		if len(calls) != 4 {
			t.Fatalf("expected 4 podman invocations (network inspect, image exists x2, compose up), got %d: %v",
				len(calls), calls)
		}

		if !strings.HasPrefix(calls[0], "network inspect stageflow_test_net") {
			t.Fatalf("call[0] = %q, want network inspect", calls[0])
		}

		last := calls[len(calls)-1]
		if !strings.Contains(last, "compose -p stageflow_test") || !strings.Contains(last, " up -d") {
			t.Fatalf("final call = %q, want a compose up -d invocation", last)
		}

		if !strings.Contains(last, "podman-compose.test.yml") {
			t.Fatalf("final call = %q, want the dev overlay file", last)
		}
	})

	t.Run("creates network when missing", func(t *testing.T) {
		logPath := filepath.Join(t.TempDir(), "calls.log")
		podman := writeFakePodman(t, logPath, map[string]int{"network inspect": 1})
		opts := testOptions(t, podman)

		var stdout, stderr bytes.Buffer

		if err := Up(context.Background(), opts, &stdout, &stderr); err != nil {
			t.Fatalf("Up() error = %v", err)
		}

		calls := readLog(t, logPath)

		found := false

		for _, c := range calls {
			if strings.HasPrefix(c, "network create stageflow_test_net") {
				found = true
			}
		}

		if !found {
			t.Fatalf("expected a network create call, got %v", calls)
		}
	})

	t.Run("fails fast when job images are missing", func(t *testing.T) {
		logPath := filepath.Join(t.TempDir(), "calls.log")
		podman := writeFakePodman(t, logPath, map[string]int{"image exists": 1})
		opts := testOptions(t, podman)

		var stdout, stderr bytes.Buffer

		err := Up(context.Background(), opts, &stdout, &stderr)
		if err == nil {
			t.Fatal("Up() error = nil, want non-nil")
		}

		if !strings.Contains(err.Error(), "just images") {
			t.Fatalf("Up() error = %q, want a hint to run `just images`", err.Error())
		}

		calls := readLog(t, logPath)
		for _, c := range calls {
			if strings.Contains(c, "compose") {
				t.Fatalf("expected compose up to be skipped when images are missing, got call: %q", c)
			}
		}
	})

	t.Run("refuses to run on a protected host", func(t *testing.T) {
		t.Setenv("STAGEFLOW_PROTECTED_HOST", mustHostname(t))
		t.Setenv("STAGEFLOW_ALLOW_VPS_LOCAL_STACKS", "")

		logPath := filepath.Join(t.TempDir(), "calls.log")
		podman := writeFakePodman(t, logPath, nil)
		opts := testOptions(t, podman)

		var stdout, stderr bytes.Buffer

		err := Up(context.Background(), opts, &stdout, &stderr)
		if err == nil {
			t.Fatal("Up() error = nil, want non-nil")
		}

		if readLog(t, logPath) != nil {
			t.Fatal("expected no podman invocations when the protected-host guard trips")
		}
	})
}

func TestDown(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "calls.log")
	podman := writeFakePodman(t, logPath, nil)
	opts := testOptions(t, podman)

	var stdout, stderr bytes.Buffer

	if err := Down(context.Background(), opts, &stdout, &stderr); err != nil {
		t.Fatalf("Down() error = %v", err)
	}

	calls := readLog(t, logPath)
	if len(calls) != 1 || !strings.Contains(calls[0], " down") {
		t.Fatalf("calls = %v, want a single compose down invocation", calls)
	}
}

func TestStatus(t *testing.T) {
	t.Run("text", func(t *testing.T) {
		logPath := filepath.Join(t.TempDir(), "calls.log")
		podman := writeFakePodman(t, logPath, nil)
		opts := testOptions(t, podman)

		var stdout, stderr bytes.Buffer

		if err := Status(context.Background(), opts, &stdout, &stderr, false); err != nil {
			t.Fatalf("Status() error = %v", err)
		}

		calls := readLog(t, logPath)
		if len(calls) != 1 || !strings.Contains(calls[0], " ps") || strings.Contains(calls[0], "--format json") {
			t.Fatalf("calls = %v, want a plain compose ps invocation", calls)
		}
	})

	t.Run("json", func(t *testing.T) {
		logPath := filepath.Join(t.TempDir(), "calls.log")
		podman := writeFakePodman(t, logPath, nil)
		opts := testOptions(t, podman)

		var stdout, stderr bytes.Buffer

		if err := Status(context.Background(), opts, &stdout, &stderr, true); err != nil {
			t.Fatalf("Status() error = %v", err)
		}

		calls := readLog(t, logPath)
		if len(calls) != 1 || !strings.Contains(calls[0], "ps --format json") {
			t.Fatalf("calls = %v, want a compose ps --format json invocation", calls)
		}
	})
}

func mustHostname(t *testing.T) string {
	t.Helper()

	host, err := os.Hostname()
	if err != nil {
		t.Fatal(err)
	}

	return host
}
