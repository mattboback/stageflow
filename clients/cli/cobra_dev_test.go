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

func TestRunDevScanCommand_PlaceholderPreflight(t *testing.T) {
	root := t.TempDir()

	_, err := scaffoldProjectConfig(root, "http://localhost:8080")
	requireNoErr(t, err)

	_, err = scaffoldProjectGuide(root)
	requireNoErr(t, err)

	var stdout bytes.Buffer

	cmd := &cobra.Command{}
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)

	runErr := runDevScanCommand(
		cmd,
		[]string{root},
		&rootOptions{apiURL: "http://localhost:8080"},
		func(string) string { return "" },
		&devScanCmdOptions{
			Timeout: time.Minute,
			Report:  reportCommandOptions{maxIssues: defaultMaxIssues},
		},
	)
	if runErr == nil {
		t.Fatalf("runDevScanCommand err = nil, want non-nil")
	}

	if !strings.Contains(runErr.Error(), "project config is not set up yet") {
		t.Fatalf("runDevScanCommand err = %q, want setup guidance", runErr.Error())
	}

	if strings.Contains(stdout.String(), "[dev] starting:") {
		t.Fatalf("runDevScanCommand unexpectedly attempted to start dev server")
	}
}

func TestRunDevInitCommand_ScaffoldsFiles(t *testing.T) {
	root := t.TempDir()

	var stdout bytes.Buffer

	cmd := &cobra.Command{}
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)

	err := runDevInitCommand(
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

	if !strings.Contains(stdout.String(), "Created StageFlow dev-loop bootstrap:") {
		t.Fatalf("expected bootstrap output, got: %q", stdout.String())
	}
}

func TestRunDevInitCommand_JSON(t *testing.T) {
	root := t.TempDir()

	var stdout bytes.Buffer

	cmd := &cobra.Command{}
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)

	err := runDevInitCommand(
		cmd,
		[]string{root},
		&rootOptions{apiURL: "http://localhost:8080", outputFormatRaw: "json"},
	)
	requireNoErr(t, err)

	var payload devInitEnvelope
	requireNoErr(t, json.NewDecoder(bytes.NewReader(stdout.Bytes())).Decode(&payload))
	requireEqual(t, payload.Schema, "stageflow-cli/dev-init@v1", "payload.Schema")
	requireEqual(t, payload.ProjectRoot, root, "payload.ProjectRoot")
	requireEqual(t, payload.Created, true, "payload.Created")

	if len(payload.NextSteps) == 0 {
		t.Fatalf("expected next steps in init payload")
	}
}

func TestRunDevDoctorCommand_MissingConfigShowsInitHint(t *testing.T) {
	root := t.TempDir()

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := runDevDoctorCommand(
		cmd,
		[]string{root},
		&rootOptions{apiURL: "http://localhost:8080"},
		func(string) string { return "" },
		&devDoctorCmdOptions{
			Timeout: time.Minute,
		},
	)
	if err == nil {
		t.Fatalf("runDevDoctorCommand err = nil, want non-nil")
	}

	if !strings.Contains(err.Error(), "stageflow dev init") {
		t.Fatalf("runDevDoctorCommand err = %q, want init hint", err.Error())
	}
}

func TestRunDevDoctorCommand_SkipDevPasses(t *testing.T) {
	root := t.TempDir()

	writeProjectConfig(t, root, `version: 2
stageflow:
  api_url: http://localhost:8080
scan:
  urls:
    - https://example.com
  scanners: [axe]
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

	err := runDevDoctorCommand(
		cmd,
		[]string{root},
		&rootOptions{apiURL: "http://localhost:8080"},
		func(string) string { return "" },
		&devDoctorCmdOptions{
			Timeout: time.Minute,
			SkipDev: true,
		},
	)
	requireNoErr(t, err)

	if !strings.Contains(stdout.String(), "Doctor checks passed") {
		t.Fatalf("expected doctor success output, got: %q", stdout.String())
	}
}

func TestRunDevDoctorCommand_SkipDevJSON(t *testing.T) {
	root := t.TempDir()

	writeProjectConfig(t, root, `version: 2
stageflow:
  api_url: http://localhost:8080
scan:
  urls:
    - http://127.0.0.1:5173
  scanners: [axe]
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

	err := runDevDoctorCommand(
		cmd,
		[]string{root},
		&rootOptions{apiURL: "http://localhost:8080", outputFormatRaw: "json"},
		func(string) string { return "" },
		&devDoctorCmdOptions{
			Timeout: time.Minute,
			SkipDev: true,
		},
	)
	requireNoErr(t, err)

	var payload devDoctorEnvelope
	requireNoErr(t, json.NewDecoder(bytes.NewReader(stdout.Bytes())).Decode(&payload))
	requireEqual(t, payload.Schema, "stageflow-cli/dev-doctor@v1", "payload.Schema")
	requireEqual(t, payload.Passed, true, "payload.Passed")
	requireEqual(t, payload.AutoAllowPrivateTargets, true, "payload.AutoAllowPrivateTargets")
	requireEqual(t, payload.RemoteProject.Configured, false, "payload.RemoteProject.Configured")

	if len(payload.Checks) != 3 {
		t.Fatalf("expected 3 checks, got %d", len(payload.Checks))
	}

	requireEqual(t, payload.Checks[2].Status, "skipped", "payload.Checks[2].Status")
}

func TestWriteDevDoctorJSONComputesPassedFromChecks(t *testing.T) {
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

	var payload devDoctorEnvelope
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

func TestRunDevDoctorCommand_SkipDevJSONRemoteProject(t *testing.T) {
	root := t.TempDir()

	writeProjectConfig(t, root, `version: 2
stageflow:
  api_url: http://localhost:8080
  project: hosted-demo
scan:
  urls:
    - https://example.com
  scanners: [axe]
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

	err := runDevDoctorCommand(
		cmd,
		[]string{root},
		&rootOptions{apiURL: "http://localhost:8080", outputFormatRaw: "json"},
		func(string) string { return "" },
		&devDoctorCmdOptions{
			Timeout: time.Minute,
			SkipDev: true,
		},
	)
	requireNoErr(t, err)

	var payload devDoctorEnvelope
	requireNoErr(t, json.NewDecoder(bytes.NewReader(stdout.Bytes())).Decode(&payload))
	requireEqual(t, payload.RemoteProject.Configured, true, "payload.RemoteProject.Configured")
	requireEqual(t, payload.RemoteProject.Slug, "hosted-demo", "payload.RemoteProject.Slug")
	requireEqual(
		t,
		payload.RemoteProject.RecommendedScanCommand,
		`stageflow project scan hosted-demo --format json`,
		"payload.RemoteProject.RecommendedScanCommand",
	)
	requireEqual(
		t,
		payload.RemoteProject.PromoteCommandTemplate,
		`stageflow project promote hosted-demo --job-id <job-id>`,
		"payload.RemoteProject.PromoteCommandTemplate",
	)
}

func TestRunDevDoctorCommand_PlaceholderSkippedWithSkipDev(t *testing.T) {
	root := t.TempDir()

	_, err := scaffoldProjectConfig(root, "http://localhost:8080")
	requireNoErr(t, err)

	_, err = scaffoldProjectGuide(root)
	requireNoErr(t, err)

	var stdout bytes.Buffer

	cmd := &cobra.Command{}
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})

	err = runDevDoctorCommand(
		cmd,
		[]string{root},
		&rootOptions{apiURL: "http://localhost:8080"},
		func(string) string { return "" },
		&devDoctorCmdOptions{
			Timeout: time.Minute,
			SkipDev: true,
		},
	)
	requireNoErr(t, err)

	if !strings.Contains(stdout.String(), "Doctor checks passed") {
		t.Fatalf("expected doctor success output, got: %q", stdout.String())
	}
}

func TestResolveProjectStageflowUsesConfigAPIURL(t *testing.T) {
	cmd := &cobra.Command{}

	apiURL, apiKey := resolveProjectStageflow(
		cmd,
		&rootOptions{apiURL: "http://localhost:8080"},
		projectConfig{
			Stageflow: projectStageflowCfg{
				APIURL:    "https://hosted.stageflow.example",
				APIKeyEnv: "HOSTED_STAGEFLOW_KEY",
			},
		},
		func(name string) string {
			requireEqual(t, name, "HOSTED_STAGEFLOW_KEY", "env name")

			return "secret-token"
		},
	)

	requireEqual(t, apiURL, "https://hosted.stageflow.example", "apiURL")
	requireEqual(t, apiKey, "secret-token", "apiKey")
}

func TestResolveProjectStageflowExplicitAPIWins(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("api", "", "")
	requireNoErr(t, cmd.Flags().Set("api", "https://explicit.stageflow.example"))

	apiURL, _ := resolveProjectStageflow(
		cmd,
		&rootOptions{apiURL: "https://explicit.stageflow.example"},
		projectConfig{
			Stageflow: projectStageflowCfg{
				APIURL: "https://hosted.stageflow.example",
			},
		},
		func(string) string { return "" },
	)

	requireEqual(t, apiURL, "https://explicit.stageflow.example", "apiURL")
}
