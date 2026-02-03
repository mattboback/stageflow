package extractor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractZIP_Valid(t *testing.T) {
	zipPath := createTestZIP(t, map[string]string{
		"index.html":       "<html><body>Index</body></html>",
		"pages/page1.html": "<html><body>Page 1</body></html>",
		"pages/page2.html": "<html><body>Page 2</body></html>",
	})

	destDir := t.TempDir()

	if err := extractZIP(zipPath, destDir); err != nil {
		t.Fatalf("Failed to extract ZIP: %v", err)
	}

	files := []string{
		"index.html",
		"pages/page1.html",
		"pages/page2.html",
	}

	for _, file := range files {
		path := filepath.Join(destDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s to exist", file)
		}
	}

	content, err := os.ReadFile(filepath.Join(destDir, "index.html")) // #nosec G304 -- reading controlled temp path
	if err != nil {
		t.Fatalf("Failed to read extracted file: %v", err)
	}

	expected := "<html><body>Index</body></html>"
	if string(content) != expected {
		t.Errorf("Expected content %q, got %q", expected, string(content))
	}
}

func TestExtractZIP_PathTraversalBlocked(t *testing.T) {
	zipPath := createTestZIP(t, map[string]string{
		"../evil.html": "malicious",
	})

	root := t.TempDir()
	destDir := filepath.Join(root, "dest")

	err := extractZIP(zipPath, destDir)
	if err == nil {
		t.Error("Expected error for path traversal, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "path traversal") && !strings.Contains(err.Error(), "illegal file path") {
		t.Errorf("Expected traversal error, got: %v", err)
	}

	evilPath := filepath.Join(root, "evil.html")
	if _, statErr := os.Stat(evilPath); !os.IsNotExist(statErr) {
		t.Errorf("Path traversal attack succeeded - file created outside destDir")
	}
}
