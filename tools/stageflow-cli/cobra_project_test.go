package main

import (
	"bytes"
	"errors"
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

func TestRunProjectDoctorCommand_PlaceholderPreflight(t *testing.T) {
	root := t.TempDir()

	_, err := scaffoldProjectConfig(root, "http://localhost:8080")
	requireNoErr(t, err)

	_, err = scaffoldProjectGuide(root)
	requireNoErr(t, err)

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
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
	if err == nil {
		t.Fatalf("runProjectDoctorCommand err = nil, want non-nil")
	}

	var exitErr exitCodeError
	if !errors.As(err, &exitErr) {
		t.Fatalf("runProjectDoctorCommand err = %T, want exitCodeError", err)
	}

	if !strings.Contains(err.Error(), "project config is not set up yet") {
		t.Fatalf("runProjectDoctorCommand err = %q, want setup guidance", err.Error())
	}
}
