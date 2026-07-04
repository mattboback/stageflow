package staticsite

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/mattboback/stageflow/clients/cli/internal/testsupport"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	testsupport.RequireNoErr(t, os.MkdirAll(filepath.Dir(path), 0o750))
	testsupport.RequireNoErr(t, os.WriteFile(path, []byte(content), 0o600))
}

func zipEntryNames(t *testing.T, zipPath string) []string {
	t.Helper()

	reader, err := zip.OpenReader(zipPath)
	testsupport.RequireNoErr(t, err)

	defer func() { _ = reader.Close() }()

	names := make([]string, 0, len(reader.File))
	for _, f := range reader.File {
		names = append(names, f.Name)
	}

	return names
}

func TestPackageDirectory(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "index.html"), "<html></html>")
	writeFile(t, filepath.Join(dir, "assets", "app.css"), "body{}")
	writeFile(t, filepath.Join(dir, ".git", "HEAD"), "ref: refs/heads/main")
	writeFile(t, filepath.Join(dir, "node_modules", "pkg", "x.js"), "junk")

	target, err := Package(dir)
	testsupport.RequireNoErr(t, err)

	defer target.Cleanup()

	testsupport.RequireEqual(t, target.HasRootIndex, true, "HasRootIndex")

	names := zipEntryNames(t, target.ZipPath)
	testsupport.RequireDeepEqual(t, names, []string{"assets/app.css", "index.html"}, "zip entries")
}

func TestPackageDirectoryWithoutIndex(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "about.html"), "<html></html>")

	target, err := Package(dir)
	testsupport.RequireNoErr(t, err)

	defer target.Cleanup()

	testsupport.RequireEqual(t, target.HasRootIndex, false, "HasRootIndex")
}

func TestPackageEmptyDirectoryFails(t *testing.T) {
	_, err := Package(t.TempDir())
	if err == nil {
		t.Fatal("expected error for empty directory")
	}
}

func TestPackageExistingZip(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "index.html"), "<html></html>")

	built, err := Package(dir)
	testsupport.RequireNoErr(t, err)

	defer built.Cleanup()

	stable := filepath.Join(t.TempDir(), "site.zip")
	data, err := os.ReadFile(built.ZipPath)
	testsupport.RequireNoErr(t, err)
	testsupport.RequireNoErr(t, os.WriteFile(stable, data, 0o600))

	target, err := Package(stable)
	testsupport.RequireNoErr(t, err)

	defer target.Cleanup()

	testsupport.RequireEqual(t, target.ZipPath, stable, "existing zip used as-is")
	testsupport.RequireEqual(t, target.HasRootIndex, true, "HasRootIndex")
}

func TestIsPathTarget(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "site.zip")
	writeFile(t, zipPath, "not really a zip")

	cases := []struct {
		arg  string
		want bool
	}{
		{dir, true},
		{zipPath, true},
		{"https://example.com", false},
		{"example.com", false},
		{filepath.Join(dir, "missing"), false},
	}

	for _, tc := range cases {
		testsupport.RequireEqual(t, IsPathTarget(tc.arg), tc.want, tc.arg)
	}
}
