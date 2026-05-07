package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestHasScaffoldPlaceholderDevCommand(t *testing.T) {
	cfg := projectConfig{
		Dev: projectDevCfg{
			Start: projectDevStartCfg{
				Cmd: []string{scaffoldDevStartCommandPlaceholder},
			},
		},
	}

	if !hasScaffoldPlaceholderDevCommand(cfg) {
		t.Fatalf("expected placeholder command to be detected")
	}
}

func TestRunProjectCommand_PlaceholderPreflight(t *testing.T) {
	root := t.TempDir()

	_, err := scaffoldProjectConfig(root, "http://localhost:8080")
	requireNoErr(t, err)

	_, err = scaffoldProjectGuide(root)
	requireNoErr(t, err)

	var stdout bytes.Buffer

	cmd := &cobra.Command{}
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)

	runErr := runProjectCommand(
		cmd,
		[]string{root},
		&rootOptions{apiURL: "http://localhost:8080"},
		func(string) string { return "" },
		&projectCmdOptions{
			Timeout:   time.Minute,
			MaxIssues: defaultMaxIssues,
		},
	)
	if runErr == nil {
		t.Fatalf("runProjectCommand err = nil, want non-nil")
	}

	if !strings.Contains(runErr.Error(), "project config is not set up yet") {
		t.Fatalf("runProjectCommand err = %q, want setup guidance", runErr.Error())
	}

	if strings.Contains(stdout.String(), "[dev] starting:") {
		t.Fatalf("runProjectCommand unexpectedly attempted to start dev server")
	}
}

func TestRunProjectInitCommand_ScaffoldsFiles(t *testing.T) {
	root := t.TempDir()

	var stdout bytes.Buffer

	cmd := &cobra.Command{}
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)

	err := runProjectInitCommand(
		cmd,
		[]string{root},
		&rootOptions{apiURL: "http://localhost:8080"},
	)
	requireNoErr(t, err)

	configPath := filepath.Join(root, ".stageflow", "config.yaml")
	guidePath := filepath.Join(root, ".stageflow", "README.md")

	if _, statErr := os.Stat(configPath); statErr != nil {
		t.Fatalf("missing config scaffold: %v", statErr)
	}

	if _, statErr := os.Stat(guidePath); statErr != nil {
		t.Fatalf("missing guide scaffold: %v", statErr)
	}

	if !strings.Contains(stdout.String(), "Created StageFlow project bootstrap:") {
		t.Fatalf("expected bootstrap output, got: %q", stdout.String())
	}
}

func TestRunProjectInitCommand_JSON(t *testing.T) {
	root := t.TempDir()

	var stdout bytes.Buffer

	cmd := &cobra.Command{}
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)

	err := runProjectInitCommand(
		cmd,
		[]string{root},
		&rootOptions{apiURL: "http://localhost:8080", outputFormatRaw: "json"},
	)
	requireNoErr(t, err)

	var payload projectInitEnvelope
	requireNoErr(t, json.NewDecoder(bytes.NewReader(stdout.Bytes())).Decode(&payload))
	requireEqual(t, payload.Schema, "stageflow-cli/project-init@v1", "payload.Schema")
	requireEqual(t, payload.ProjectRoot, root, "payload.ProjectRoot")
	requireEqual(t, payload.Created, true, "payload.Created")

	if len(payload.NextSteps) == 0 {
		t.Fatalf("expected next steps in init payload")
	}
}

func TestRunProjectDoctorCommand_MissingConfigShowsInitHint(t *testing.T) {
	root := t.TempDir()

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := runProjectDoctorCommand(
		cmd,
		[]string{root},
		&rootOptions{apiURL: "http://localhost:8080"},
		func(string) string { return "" },
		&projectDoctorCmdOptions{
			Timeout: time.Minute,
		},
	)
	if err == nil {
		t.Fatalf("runProjectDoctorCommand err = nil, want non-nil")
	}

	if !strings.Contains(err.Error(), "stageflow project init") {
		t.Fatalf("runProjectDoctorCommand err = %q, want init hint", err.Error())
	}
}

func TestRunProjectDoctorCommand_SkipDevPasses(t *testing.T) {
	root := t.TempDir()

	writeProjectConfig(t, root, `version: 1
stageflow:
  api_url: http://localhost:8080
scan:
  urls:
    - https://example.com
  scanners: axe
dev:
  start:
    cmd: ["echo", "dev"]
  ready:
    url: http://127.0.0.1:3000
`)

	var stdout bytes.Buffer

	cmd := &cobra.Command{}
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})

	err := runProjectDoctorCommand(
		cmd,
		[]string{root},
		&rootOptions{apiURL: "http://localhost:8080"},
		func(string) string { return "" },
		&projectDoctorCmdOptions{
			Timeout: time.Minute,
			SkipDev: true,
		},
	)
	requireNoErr(t, err)

	if !strings.Contains(stdout.String(), "Doctor checks passed") {
		t.Fatalf("expected doctor success output, got: %q", stdout.String())
	}
}

func TestRunProjectDoctorCommand_SkipDevJSON(t *testing.T) {
	root := t.TempDir()

	writeProjectConfig(t, root, `version: 1
stageflow:
  api_url: http://localhost:8080
scan:
  urls:
    - http://127.0.0.1:5173
  scanners: axe
dev:
  start:
    cmd: ["echo", "dev"]
  ready:
    url: http://127.0.0.1:5173
`)

	var stdout bytes.Buffer

	cmd := &cobra.Command{}
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})

	err := runProjectDoctorCommand(
		cmd,
		[]string{root},
		&rootOptions{apiURL: "http://localhost:8080", outputFormatRaw: "json"},
		func(string) string { return "" },
		&projectDoctorCmdOptions{
			Timeout: time.Minute,
			SkipDev: true,
		},
	)
	requireNoErr(t, err)

	var payload projectDoctorEnvelope
	requireNoErr(t, json.NewDecoder(bytes.NewReader(stdout.Bytes())).Decode(&payload))
	requireEqual(t, payload.Schema, "stageflow-cli/project-doctor@v1", "payload.Schema")
	requireEqual(t, payload.Passed, true, "payload.Passed")
	requireEqual(t, payload.AutoAllowPrivateTargets, true, "payload.AutoAllowPrivateTargets")
	requireEqual(t, payload.HostedMemory.Configured, false, "payload.HostedMemory.Configured")

	if len(payload.Checks) != 3 {
		t.Fatalf("expected 3 checks, got %d", len(payload.Checks))
	}

	requireEqual(t, payload.Checks[2].Status, "skipped", "payload.Checks[2].Status")
}

func TestWriteProjectDoctorJSONComputesPassedFromChecks(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer

	err := writeProjectDoctorJSON(
		&stdout,
		"/project",
		"/project/.stageflow/config.yaml",
		"http://localhost:8080",
		[]string{"https://example.com"},
		projectStageflowCfg{},
		false,
		[]projectDoctorCheck{
			{Name: "config", Status: "passed"},
			{Name: "dev-readiness", Status: "failed", Message: "not ready"},
		},
	)
	requireNoErr(t, err)

	var payload projectDoctorEnvelope
	requireNoErr(t, json.NewDecoder(bytes.NewReader(stdout.Bytes())).Decode(&payload))
	requireEqual(t, payload.Passed, false, "payload.Passed")
}

func TestAbbreviateIDHandlesShortIDs(t *testing.T) {
	t.Parallel()

	requireEqual(t, abbreviateID("", 8), "", "empty id")
	requireEqual(t, abbreviateID("short", 8), "short", "short id")
	requireEqual(t, abbreviateID("123456789", 8), "12345678", "long id")
	requireEqual(t, abbreviateID("short", 0), "short", "zero max")
}

func TestRunProjectDoctorCommand_SkipDevJSONHostedMemory(t *testing.T) {
	root := t.TempDir()

	writeProjectConfig(t, root, `version: 1
stageflow:
  api_url: http://localhost:8080
  remote_project: hosted-demo
  remote_api_url: https://hosted.stageflow.example
scan:
  urls:
    - https://example.com
  scanners: axe
dev:
  start:
    cmd: ["echo", "dev"]
  ready:
    url: http://127.0.0.1:5173
`)

	var stdout bytes.Buffer

	cmd := &cobra.Command{}
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})

	err := runProjectDoctorCommand(
		cmd,
		[]string{root},
		&rootOptions{apiURL: "http://localhost:8080", outputFormatRaw: "json"},
		func(string) string { return "" },
		&projectDoctorCmdOptions{
			Timeout: time.Minute,
			SkipDev: true,
		},
	)
	requireNoErr(t, err)

	var payload projectDoctorEnvelope
	requireNoErr(t, json.NewDecoder(bytes.NewReader(stdout.Bytes())).Decode(&payload))
	requireEqual(t, payload.HostedMemory.Configured, true, "payload.HostedMemory.Configured")
	requireEqual(t, payload.HostedMemory.ProjectSlug, "hosted-demo", "payload.HostedMemory.ProjectSlug")
	requireEqual(t, payload.HostedMemory.APIURL, "https://hosted.stageflow.example", "payload.HostedMemory.APIURL")
	requireEqual(
		t,
		payload.HostedMemory.RecommendedScanCommand,
		`stageflow scan --project hosted-demo --format json --api https://hosted.stageflow.example`,
		"payload.HostedMemory.RecommendedScanCommand",
	)
	requireEqual(
		t,
		payload.HostedMemory.PromoteCommandTemplate,
		`stageflow project promote hosted-demo --job-id <job-id> --api https://hosted.stageflow.example`,
		"payload.HostedMemory.PromoteCommandTemplate",
	)
}

func TestRunProjectDoctorCommand_PlaceholderSkippedWithSkipDev(t *testing.T) {
	root := t.TempDir()

	_, err := scaffoldProjectConfig(root, "http://localhost:8080")
	requireNoErr(t, err)

	_, err = scaffoldProjectGuide(root)
	requireNoErr(t, err)

	var stdout bytes.Buffer

	cmd := &cobra.Command{}
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})

	err = runProjectDoctorCommand(
		cmd,
		[]string{root},
		&rootOptions{apiURL: "http://localhost:8080"},
		func(string) string { return "" },
		&projectDoctorCmdOptions{
			Timeout: time.Minute,
			SkipDev: true,
		},
	)
	requireNoErr(t, err)

	if !strings.Contains(stdout.String(), "Doctor checks passed") {
		t.Fatalf("expected doctor success output, got: %q", stdout.String())
	}
}
