package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/mattboback/stageflow/clients/cli/internal/testsupport"
)

func TestGenerateMarkdownDocsOmitsHiddenRegisteredCommands(t *testing.T) {
	root := newRootCmd(testsupport.StubEnv, io.Discard, io.Discard)

	aiCmd := findChildCommand(root, "ai")
	if aiCmd == nil {
		t.Fatal("expected ai command to be registered")
	}

	if !aiCmd.Hidden {
		t.Fatal("test expects ai to remain hidden in runtime help")
	}

	dir := t.TempDir()
	if err := generateMarkdownDocs(root, dir); err != nil {
		t.Fatalf("generateMarkdownDocs() error = %v", err)
	}

	if !aiCmd.Hidden {
		t.Fatal("generateMarkdownDocs() did not restore hidden state")
	}

	aiDoc := filepath.Join(dir, "stageflow_ai.md")
	if _, err := os.Stat(aiDoc); !os.IsNotExist(err) {
		t.Fatalf("expected hidden ai docs to be omitted at %s; stat err = %v", aiDoc, err)
	}

	rootDoc, err := os.ReadFile(filepath.Join(dir, "stageflow.md"))
	if err != nil {
		t.Fatalf("read generated root docs: %v", err)
	}

	if strings.Contains(string(rootDoc), "[stageflow ai](stageflow_ai.md)") {
		t.Fatalf("expected root docs to omit hidden registered ai command; root docs:\n%s", rootDoc)
	}
}

func TestVerifyGeneratedCommandDocsReportsMissingRegisteredCommandDocs(t *testing.T) {
	err := verifyGeneratedCommandDocs(t.TempDir(), []string{"stageflow.md", "stageflow_ai.md"})
	if err == nil {
		t.Fatal("expected missing docs error")
	}

	if !strings.Contains(err.Error(), "stageflow_ai.md") {
		t.Fatalf("expected missing docs error to name stageflow_ai.md; got %v", err)
	}
}

func findChildCommand(root *cobra.Command, name string) *cobra.Command {
	for _, child := range root.Commands() {
		if child.Name() == name {
			return child
		}
	}

	return nil
}
