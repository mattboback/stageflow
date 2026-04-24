package projectmode

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveProjectRoot_GitRootFromNestedDir(t *testing.T) {
	root := t.TempDir()

	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o750); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}

	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0o750); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	got, err := ResolveProjectRoot(nested)
	if err != nil {
		t.Fatalf("ResolveProjectRoot error: %v", err)
	}

	if got != root {
		t.Fatalf("ResolveProjectRoot(%q) = %q, want %q", nested, got, root)
	}
}

func TestResolveProjectRoot_NoGitReturnsExplicitDir(t *testing.T) {
	root := t.TempDir()

	got, err := ResolveProjectRoot(root)
	if err != nil {
		t.Fatalf("ResolveProjectRoot error: %v", err)
	}

	if got != root {
		t.Fatalf("ResolveProjectRoot(%q) = %q, want %q", root, got, root)
	}
}

func TestResolveProjectRoot_FilePathReturnsError(t *testing.T) {
	root := t.TempDir()

	filePath := filepath.Join(root, "file.txt")
	if err := os.WriteFile(filePath, []byte("ok\n"), 0o640); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := ResolveProjectRoot(filePath)
	if err == nil {
		t.Fatalf("ResolveProjectRoot err = nil, want non-nil")
	}

	if !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("ResolveProjectRoot err = %q, want to contain %q", err.Error(), "not a directory")
	}
}
