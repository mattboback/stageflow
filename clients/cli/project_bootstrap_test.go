package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mattboback/stageflow/clients/cli/internal/testsupport"
)

func TestDetectProjectBootstrapSuggestion_UsesPlaceholderWhenNoCommandFound(t *testing.T) {
	root := t.TempDir()

	suggestion, err := detectProjectBootstrapSuggestion(root)
	testsupport.RequireNoErr(t, err)

	testsupport.RequireEqual(t, suggestion.Command, scaffoldDevStartCommandPlaceholder, "suggestion.Command")
	testsupport.RequireEqual(
		t,
		suggestion.CommandSource,
		"set `dev.start.cmd` to the command that makes your app reachable locally",
		"suggestion.CommandSource",
	)

	if !suggestion.IsPlaceholder {
		t.Fatalf("expected placeholder suggestion")
	}
}

func TestDetectProjectBootstrapSuggestion_UsesJustRunFrontendForWorkspace(t *testing.T) {
	root := t.TempDir()

	justfile := filepath.Join(root, "justfile")
	err := os.WriteFile(justfile, []byte("run SERVICE:\n\t@true\n"), 0o600)
	testsupport.RequireNoErr(t, err)

	frontendDir := filepath.Join(root, "frontend")
	err = os.MkdirAll(frontendDir, 0o750)
	testsupport.RequireNoErr(t, err)

	err = os.WriteFile(
		filepath.Join(frontendDir, "package.json"),
		[]byte(`{"scripts":{"dev":"vite dev"}}`),
		0o600,
	)
	testsupport.RequireNoErr(t, err)

	err = os.WriteFile(filepath.Join(frontendDir, "vite.config.ts"), []byte("export default {}\n"), 0o600)
	testsupport.RequireNoErr(t, err)

	suggestion, err := detectProjectBootstrapSuggestion(root)
	testsupport.RequireNoErr(t, err)

	testsupport.RequireEqual(
		t,
		suggestion.Command,
		"just run frontend", // stale-vocab-ok: tests generic fallback
		"suggestion.Command",
	) // stale-vocab-ok: tests generic fallback
	testsupport.RequireEqual(t, suggestion.URL, "http://127.0.0.1:5173", "suggestion.URL")

	if suggestion.IsPlaceholder {
		t.Fatalf("expected inferred command, got placeholder")
	}
}

func TestDefaultProjectConfigTemplate_UsesBootstrapSuggestion(t *testing.T) {
	template := defaultProjectConfigTemplate("http://localhost:8080", projectBootstrapSuggestion{
		Command:       "just run clients/web",
		CommandSource: "best guess from Justfile recipe `run`",
		URL:           "http://127.0.0.1:5173",
	})

	if !strings.Contains(template, `cmd: ["just", "run", "clients/web"]`) {
		t.Fatalf("template missing inferred command: %q", template)
	}

	if !strings.Contains(template, "best guess from Justfile recipe `run`") {
		t.Fatalf("template missing command source comment: %q", template)
	}

	if !strings.Contains(template, "url: http://127.0.0.1:5173") {
		t.Fatalf("template missing inferred URL: %q", template)
	}
}
