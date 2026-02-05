package extractor

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func createTestZIP(tb testing.TB, files map[string]string) string {
	tb.Helper()

	tmpDir := tb.TempDir()
	zipPath := filepath.Join(tmpDir, "test.zip")

	f, err := os.Create(zipPath) // #nosec G304 -- temp path under test control
	if err != nil {
		tb.Fatalf("Failed to create ZIP file: %v", err)
	}

	defer func() {
		if cerr := f.Close(); cerr != nil {
			tb.Fatalf("Failed to close ZIP file: %v", cerr)
		}
	}()

	zw := zip.NewWriter(f)

	defer func() {
		if cerr := zw.Close(); cerr != nil {
			tb.Fatalf("Failed to close ZIP writer: %v", cerr)
		}
	}()

	for name, content := range files {
		fw, createErr := zw.Create(name)
		if createErr != nil {
			tb.Fatalf("Failed to create ZIP entry %s: %v", name, createErr)
		}

		if _, writeErr := io.WriteString(fw, content); writeErr != nil {
			tb.Fatalf("Failed to write ZIP entry %s: %v", name, writeErr)
		}
	}

	return zipPath
}
