package extractor

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateZIP_Valid(t *testing.T) {
	zipPath := createTestZIP(t, map[string]string{
		"index.html": "<html><body>Test</body></html>",
		"page.html":  "<html><body>Page</body></html>",
	})

	info, err := os.Stat(zipPath)
	if err != nil {
		t.Fatalf("Failed to stat ZIP: %v", err)
	}

	if validateErr := validateZIP(zipPath, info.Size()); validateErr != nil {
		t.Errorf("Expected valid ZIP to pass validation, got error: %v", validateErr)
	}
}

func TestValidateZIP_PathTraversal(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "double dot traversal",
			filename: "../../etc/passwd",
			wantErr:  true,
			errMsg:   "path traversal detected",
		},
		{
			name:     "single dot traversal",
			filename: "../secret.html",
			wantErr:  true,
			errMsg:   "path traversal detected",
		},
		{
			name:     "hidden traversal",
			filename: "normal/../../../etc/file.html",
			wantErr:  true,
			errMsg:   "path traversal detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zipPath := createTestZIP(t, map[string]string{
				tt.filename: "malicious content",
			})

			info, err := os.Stat(zipPath)
			if err != nil {
				t.Fatalf("Failed to stat ZIP: %v", err)
			}

			err = validateZIP(zipPath, info.Size())

			if tt.wantErr && err == nil {
				t.Errorf("Expected error containing %q, got nil", tt.errMsg)
			}

			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Expected error containing %q, got: %v", tt.errMsg, err)
			}

			if !tt.wantErr && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestValidateZIP_AbsolutePath(t *testing.T) {
	zipPath := createTestZIP(t, map[string]string{
		"/etc/passwd.html": "malicious",
	})

	info, err := os.Stat(zipPath)
	if err != nil {
		t.Fatalf("Failed to stat ZIP: %v", err)
	}

	err = validateZIP(zipPath, info.Size())
	if err == nil {
		t.Error("Expected error for absolute path, got nil")
	}

	if err != nil && !strings.Contains(err.Error(), "absolute path") {
		t.Errorf("Expected 'absolute path' error, got: %v", err)
	}
}

func TestValidateZIP_SizeLimit(t *testing.T) {
	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "large.zip")

	f, err := os.Create(zipPath) // #nosec G304 -- path is under test control
	if err != nil {
		t.Fatalf("Failed to create ZIP: %v", err)
	}

	zw := zip.NewWriter(f)

	fw, err := zw.Create("large-file.html")
	if err != nil {
		t.Fatalf("Failed to create ZIP entry: %v", err)
	}

	content := []byte("<html><body>Large file</body></html>")
	if _, writeErr := fw.Write(content); writeErr != nil {
		t.Fatalf("Failed to write content: %v", writeErr)
	}

	if closeZipErr := zw.Close(); closeZipErr != nil {
		t.Fatalf("Failed to close ZIP writer: %v", closeZipErr)
	}

	if closeFileErr := f.Close(); closeFileErr != nil {
		t.Fatalf("Failed to close ZIP file: %v", closeFileErr)
	}

	info, statErr := os.Stat(zipPath)
	if statErr != nil {
		t.Fatalf("Failed to stat ZIP: %v", statErr)
	}

	if validateErr := validateZIP(zipPath, info.Size()); validateErr != nil {
		t.Errorf("Normal size ZIP should pass, got: %v", validateErr)
	}
}

func TestValidateZIP_ExpansionRatio(t *testing.T) {
	largeContent := strings.Repeat("<html><body>Test</body></html>", 100)

	zipPath := createTestZIP(t, map[string]string{
		"compressed.html": largeContent,
	})

	info, statErr := os.Stat(zipPath)
	if statErr != nil {
		t.Fatalf("Failed to stat ZIP: %v", statErr)
	}

	if validateErr := validateZIP(zipPath, info.Size()); validateErr != nil {
		t.Errorf("Expected normal compression to pass, got: %v", validateErr)
	}
}

func TestValidateZIP_MaxEntries(t *testing.T) {
	const limit = 5000

	zipPath := createZIPWithNEntries(t, limit+1, func(i int) string {
		return fmt.Sprintf("f%d.html", i)
	})

	info, err := os.Stat(zipPath)
	if err != nil {
		t.Fatalf("Failed to stat ZIP: %v", err)
	}

	if validateErr := validateZIP(zipPath, info.Size()); validateErr == nil {
		t.Fatalf("expected max entries validation error, got nil")
	}

	zipPath2 := createZIPWithNEntries(t, limit, func(i int) string {
		return fmt.Sprintf("g%d.html", i)
	})

	info2, err := os.Stat(zipPath2)
	if err != nil {
		t.Fatalf("Failed to stat ZIP: %v", err)
	}

	if validateErr := validateZIP(zipPath2, info2.Size()); validateErr != nil {
		t.Fatalf("expected under-limit zip to pass, got %v", validateErr)
	}
}

func createZIPWithNEntries(t *testing.T, count int, filename func(i int) string) string {
	t.Helper()

	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "many.zip")

	f, err := os.Create(zipPath) // #nosec G304 -- temp path under test control
	if err != nil {
		t.Fatalf("failed to create zip: %v", err)
	}

	zw := zip.NewWriter(f)

	for i := range count {
		w, createErr := zw.Create(filename(i))
		if createErr != nil {
			t.Fatalf("create entry: %v", createErr)
		}

		if _, writeErr := w.Write([]byte("hi")); writeErr != nil {
			t.Fatalf("write entry: %v", writeErr)
		}
	}

	if closeZipErr := zw.Close(); closeZipErr != nil {
		t.Fatalf("close zip writer: %v", closeZipErr)
	}

	if closeFileErr := f.Close(); closeFileErr != nil {
		t.Fatalf("close zip file: %v", closeFileErr)
	}

	return zipPath
}

func BenchmarkExtractZIP(b *testing.B) {
	files := make(map[string]string)
	for i := range 100 {
		files[filepath.Join("pages", fmt.Sprintf("page%03d.html", i))] =
			"<html><body>Page content here</body></html>"
	}

	zipPath := createTestZIP(b, files)

	b.ResetTimer()

	for range b.N {
		destDir := b.TempDir()
		if err := extractZIP(zipPath, destDir); err != nil {
			b.Fatalf("Failed to extract ZIP: %v", err)
		}
	}
}

func BenchmarkValidateZIP(b *testing.B) {
	buf := bytes.Repeat([]byte("a"), 4096)

	files := map[string]string{
		"index.html": string(buf),
		"page.html":  string(buf),
	}

	zipPath := createTestZIP(b, files)

	info, err := os.Stat(zipPath)
	if err != nil {
		b.Fatalf("Failed to stat ZIP: %v", err)
	}

	b.ResetTimer()

	for range b.N {
		if validateErr := validateZIP(zipPath, info.Size()); validateErr != nil {
			b.Fatalf("validate: %v", validateErr)
		}
	}
}
