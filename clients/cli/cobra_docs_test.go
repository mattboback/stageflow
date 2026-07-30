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

// The hidden command is registered by the test rather than borrowed from the
// real command set: the CLI has no hidden commands of its own at the moment,
// and pinning this behaviour to whichever one happens to be hidden means the
// coverage disappears the next time that command does.
func TestGenerateMarkdownDocsOmitsHiddenRegisteredCommands(t *testing.T) {
	root := newRootCmd(testsupport.StubEnv, io.Discard, io.Discard)

	hidden := &cobra.Command{
		Use:    "secret",
		Short:  "Hidden from runtime help and from the generated docs",
		Hidden: true,
		RunE:   func(*cobra.Command, []string) error { return nil },
	}
	root.AddCommand(hidden)

	dir := t.TempDir()
	if err := generateMarkdownDocs(root, dir); err != nil {
		t.Fatalf("generateMarkdownDocs() error = %v", err)
	}

	if !hidden.Hidden {
		t.Fatal("generateMarkdownDocs() did not restore hidden state")
	}

	hiddenDoc := filepath.Join(dir, "stageflow_secret.md")
	if _, err := os.Stat(hiddenDoc); !os.IsNotExist(err) {
		t.Fatalf("expected hidden docs to be omitted at %s; stat err = %v", hiddenDoc, err)
	}

	rootDoc, err := os.ReadFile(filepath.Join(dir, "stageflow.md"))
	if err != nil {
		t.Fatalf("read generated root docs: %v", err)
	}

	if strings.Contains(string(rootDoc), "[stageflow secret](stageflow_secret.md)") {
		t.Fatalf("expected root docs to omit hidden registered command; root docs:\n%s", rootDoc)
	}
}

func TestVerifyGeneratedCommandDocsReportsMissingRegisteredCommandDocs(t *testing.T) {
	err := verifyGeneratedCommandDocs(t.TempDir(), []string{"stageflow.md", "stageflow_scan.md"})
	if err == nil {
		t.Fatal("expected missing docs error")
	}

	if !strings.Contains(err.Error(), "stageflow_scan.md") {
		t.Fatalf("expected missing docs error to name stageflow_scan.md; got %v", err)
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
