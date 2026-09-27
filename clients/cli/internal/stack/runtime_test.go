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

	b.WriteString(`case "$*" in
  compose*config)
    if [ -n "$FAKE_COMPOSE_CONFIG" ]; then
      printf '%s' "$FAKE_COMPOSE_CONFIG"
    else
      cat <<EOF
name: ${COMPOSE_PROJECT_NAME:-stageflow_dev}
networks:
  stageflow_net:
    name: ${COMPOSE_PROJECT_NAME:-stageflow_dev}_net
services:
  orchestrator:
    environment:
      EXTRACTION_IMAGE: localhost/stageflow/extractor:latest
      SCANNER_IMAGE: localhost/stageflow/scanner-runner:latest
EOF
    fi
    ;;
esac
`)

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

func TestUpHappyPathStreamsComposeUp(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "calls.log")
	podman := writeFakePodman(t, logPath, nil)
	opts := testOptions(t, podman)

	var stdout, stderr bytes.Buffer

	if err := Up(context.Background(), opts, &stdout, &stderr); err != nil {
		t.Fatalf("Up() error = %v", err)
	}

	calls := readLog(t, logPath)

	if len(calls) != 5 {
		t.Fatalf("expected 5 calls (config, network inspect, image exists x2, up), got %d: %v",
			len(calls), calls)
	}

	if !strings.HasPrefix(calls[1], "network inspect stageflow_test_net") {
		t.Fatalf("call[1] = %q, want network inspect", calls[1])
	}

	last := calls[len(calls)-1]
	if !strings.Contains(last, "compose -p stageflow_test") || !strings.Contains(last, " up -d") {
		t.Fatalf("final call = %q, want a compose up -d invocation", last)
	}

	if strings.Count(last, " -f ") != 1 || !strings.Contains(last, "podman-compose.yml") {
		t.Fatalf("final call = %q, want only the base compose file", last)
	}
}

func TestUpCreatesNetworkWhenMissing(t *testing.T) {
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
}

func TestUpResolvesProjectFromSelectedEnvironmentFile(t *testing.T) {
	for _, external := range []bool{false, true} {
		t.Run(fmt.Sprintf("external=%t", external), func(t *testing.T) {
			t.Setenv("COMPOSE_PROJECT_NAME", "")
			t.Setenv("FAKE_COMPOSE_CONFIG", `name: file_project
networks:
  stageflow_net:
    name: file_project_net
services:
  orchestrator:
    environment:
      EXTRACTION_IMAGE: localhost/stageflow/extractor:latest
      SCANNER_IMAGE: localhost/stageflow/scanner-runner:latest
`)
			logPath := filepath.Join(t.TempDir(), "calls.log")
			opts := testOptions(t, writeFakePodman(t, logPath, nil))
			opts.Project = ""
			envFile := filepath.Join(opts.Root, ".env")

			if external {
				envFile = filepath.Join(t.TempDir(), "selected.env")
				opts.EnvFile = envFile
			}

			if err := os.WriteFile(envFile, []byte("COMPOSE_PROJECT_NAME=file_project\n"), 0o600); err != nil {
				t.Fatal(err)
			}

			var stdout, stderr bytes.Buffer
			if err := Up(context.Background(), opts, &stdout, &stderr); err != nil {
				t.Fatalf("Up() error = %v", err)
			}

			calls := readLog(t, logPath)
			if strings.Contains(calls[0], " -p ") || !strings.Contains(calls[0], "--env-file "+envFile+" config") {
				t.Fatalf("config must let selected file determine project: %s", calls[0])
			}

			last := calls[len(calls)-1]
			if !strings.Contains(last, "compose -p file_project ") ||
				!strings.Contains(last, "--env-file "+envFile+" up -d") {
				t.Fatalf("up must preserve rendered project and selected file: %s", last)
			}
		})
	}
}

func TestUpUsesDefaultRenderedProject(t *testing.T) {
	t.Setenv("COMPOSE_PROJECT_NAME", "")
	logPath := filepath.Join(t.TempDir(), "calls.log")
	opts := testOptions(t, writeFakePodman(t, logPath, nil))
	opts.Project = ""

	var stdout, stderr bytes.Buffer
	if err := Up(context.Background(), opts, &stdout, &stderr); err != nil {
		t.Fatalf("Up() error = %v", err)
	}

	calls := readLog(t, logPath)
	if last := calls[len(calls)-1]; !strings.Contains(last, "compose -p "+DefaultProject+" ") {
		t.Fatalf("up must use rendered default project: %s", last)
	}
}

func TestUpUsesRenderedNetworkAndJobImages(t *testing.T) {
	t.Setenv("FAKE_COMPOSE_CONFIG", `name: stageflow_test
networks:
  stageflow_net:
    name: configured_network
services:
  orchestrator:
    environment:
      EXTRACTION_IMAGE: localhost/stageflow/extractor:custom-tag
      SCANNER_IMAGE: localhost/stageflow/scanner-runner:custom-tag
`)
	logPath := filepath.Join(t.TempDir(), "calls.log")
	podman := writeFakePodman(t, logPath, map[string]int{"network inspect": 1})
	opts := testOptions(t, podman)
	opts.Env = EnvLocal
	envFile := filepath.Join(opts.Root, ".env")
	envContent := []byte("IMAGE_TAG=custom-tag\nSTAGEFLOW_NETWORK_NAME=configured_network\n")

	if err := os.WriteFile(envFile, envContent, 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if err := Up(context.Background(), opts, &stdout, &stderr); err != nil {
		t.Fatalf("Up() error = %v", err)
	}

	calls := strings.Join(readLog(t, logPath), "\n")
	for _, want := range []string{
		"--env-file " + envFile + " config",
		"--env-file " + envFile + " up -d",
		"network inspect configured_network",
		"network create configured_network",
		"image exists localhost/stageflow/extractor:custom-tag",
		"image exists localhost/stageflow/scanner-runner:custom-tag",
	} {
		if !strings.Contains(calls, want) {
			t.Errorf("calls missing %q: %s", want, calls)
		}
	}
}

func TestUpRejectsInvalidRenderedConfiguration(t *testing.T) {
	for _, config := range []string{"services: {}", "networks:\n  stageflow_net:\n    name: [private-secret]"} {
		t.Run(config, func(t *testing.T) {
			t.Setenv("FAKE_COMPOSE_CONFIG", config)
			logPath := filepath.Join(t.TempDir(), "calls.log")
			opts := testOptions(t, writeFakePodman(t, logPath, nil))

			var stdout, stderr bytes.Buffer

			err := Up(context.Background(), opts, &stdout, &stderr)
			if err == nil {
				t.Fatal("Up() succeeded with invalid rendered configuration")
			}

			if strings.Contains(err.Error()+stdout.String()+stderr.String(), "private-secret") {
				t.Fatal("rendered configuration value leaked in diagnostic")
			}

			if calls := readLog(t, logPath); len(calls) != 1 || !strings.HasSuffix(calls[0], " config") {
				t.Fatalf("invalid configuration must stop before network or stack mutations: %v", calls)
			}
		})
	}
}

func TestUpFailsFastWhenJobImagesAreMissing(t *testing.T) {
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
		if strings.Contains(c, " up ") {
			t.Fatalf("expected compose up to be skipped when images are missing, got call: %q", c)
		}
	}
}

func TestUpRefusesToRunOnAProtectedHost(t *testing.T) {
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
