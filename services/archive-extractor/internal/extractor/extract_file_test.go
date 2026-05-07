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

	n, extractErr := extractFile(r.File[0], destPath, maxZipEntryUncompressedSize)
	if extractErr != nil {
		t.Fatalf("Failed to extract file: %v", extractErr)
	}

	expected := "<html><body>Test</body></html>"
	if n != int64(len(expected)) {
		t.Fatalf("Expected %d extracted bytes, got %d", len(expected), n)
	}

	content, err := os.ReadFile(destPath) // #nosec G304 -- reading controlled temp path
	if err != nil {
		t.Fatalf("Failed to read extracted file: %v", err)
	}

	if string(content) != expected {
		t.Errorf("Expected %q, got %q", expected, string(content))
	}
}
