package extractor

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractFile(t *testing.T) {
	zipPath := createTestZIP(t, map[string]string{
		"test.html": "<html><body>Test</body></html>",
	})

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("Failed to open ZIP: %v", err)
	}
	defer func() {
		if cerr := r.Close(); cerr != nil {
			t.Fatalf("Failed to close ZIP reader: %v", cerr)
		}
	}()

	if len(r.File) == 0 {
		t.Fatal("ZIP is empty")
	}

	destPath := filepath.Join(t.TempDir(), "extracted.html")
	extractErr := extractFile(r.File[0], destPath)
	if extractErr != nil {
		t.Fatalf("Failed to extract file: %v", extractErr)
	}

	content, err := os.ReadFile(destPath) // #nosec G304 -- reading controlled temp path
	if err != nil {
		t.Fatalf("Failed to read extracted file: %v", err)
	}

	expected := "<html><body>Test</body></html>"
	if string(content) != expected {
		t.Errorf("Expected %q, got %q", expected, string(content))
	}
}
