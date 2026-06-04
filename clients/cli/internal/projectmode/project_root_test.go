package projectmode

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveProjectRoot_GitRootFromNestedDir(t *testing.T) {
	root := t.TempDir()

	writeGitHead(t, root)

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

func TestResolveProjectRoot_IgnoresEmptyGitDirectoryInParent(t *testing.T) {
	root := t.TempDir()

	emptyGitParent := filepath.Join(root, "parent")
	if err := os.MkdirAll(filepath.Join(emptyGitParent, ".git"), 0o750); err != nil {
		t.Fatalf("mkdir empty parent .git: %v", err)
	}

	projectDir := filepath.Join(emptyGitParent, "project")
	if err := os.MkdirAll(projectDir, 0o750); err != nil {
		t.Fatalf("mkdir project dir: %v", err)
	}

	got, err := ResolveProjectRoot(projectDir)
	if err != nil {
		t.Fatalf("ResolveProjectRoot error: %v", err)
	}

	if got != projectDir {
		t.Fatalf("ResolveProjectRoot(%q) = %q, want %q", projectDir, got, projectDir)
	}
}

func TestFindGitRoot_WorktreeGitFile(t *testing.T) {
	root := t.TempDir()

	gitDir := filepath.Join(root, ".git-worktree")
	if err := os.MkdirAll(gitDir, 0o750); err != nil {
		t.Fatalf("mkdir git dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(root, ".git"), []byte("gitdir: "+gitDir+"\n"), 0o600); err != nil {
		t.Fatalf("write .git file: %v", err)
	}

	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0o750); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	got, ok, err := FindGitRoot(nested)
	if err != nil {
		t.Fatalf("FindGitRoot error: %v", err)
	}

	if !ok {
		t.Fatalf("FindGitRoot did not find root from %q", nested)
	}

	if got != root {
		t.Fatalf("FindGitRoot(%q) = %q, want %q", nested, got, root)
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

func writeGitHead(t *testing.T, root string) {
	t.Helper()

	dotGit := filepath.Join(root, ".git")
	if err := os.MkdirAll(dotGit, 0o750); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dotGit, "HEAD"), []byte("ref: refs/heads/main\n"), 0o600); err != nil {
		t.Fatalf("write .git/HEAD: %v", err)
	}
}
