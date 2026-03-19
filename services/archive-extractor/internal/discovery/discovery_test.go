package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverHTML_SimpleDirectory(t *testing.T) {
	// Use testdata fixtures
	testdataDir := filepath.Join("..", "..", "testdata", "html", "simple")

	pages, err := DiscoverHTML(testdataDir)
	if err != nil {
		t.Fatalf("Failed to discover HTML: %v", err)
	}

	// Should find 2 HTML files: index.html and about.htm
	if len(pages) != 2 {
		t.Errorf("Expected 2 pages, got %d", len(pages))
	}

	assertPageFields(t, pages)
	assertSimpleExtensions(t, pages)
}

func assertPageFields(t *testing.T, pages []HTMLPage) {
	t.Helper()

	for i, page := range pages {
		if page.ID == "" {
			t.Errorf("Page %d has empty ID", i)
		}

		if page.Path == "" {
			t.Errorf("Page %d has empty Path", i)
		}

		if page.File == "" {
			t.Errorf("Page %d has empty File", i)
		}

		// Verify ID format
		if len(page.ID) < 8 || page.ID[:5] != "page-" {
			t.Errorf("Page %d has invalid ID format: %s", i, page.ID)
		}

		// Verify path starts with /
		if page.Path[0] != '/' {
			t.Errorf("Page %d path should start with /: %s", i, page.Path)
		}
	}
}

func assertSimpleExtensions(t *testing.T, pages []HTMLPage) {
	t.Helper()

	// Verify we found both .html and .htm files
	foundHTML := false
	foundHTM := false

	for _, page := range pages {
		if filepath.Ext(page.File) == ".html" {
			foundHTML = true
		}

		if filepath.Ext(page.File) == ".htm" {
			foundHTM = true
		}
	}

	if !foundHTML {
		t.Error("Expected to find .html files")
	}

	if !foundHTM {
		t.Error("Expected to find .htm files")
	}
}

func TestDiscoverHTML_NestedDirectory(t *testing.T) {
	testdataDir := filepath.Join("..", "..", "testdata", "html", "nested")

	pages, err := DiscoverHTML(testdataDir)
	if err != nil {
		t.Fatalf("Failed to discover HTML: %v", err)
	}

	// Should find 2 HTML files: page.html and subdir/deep.html
	if len(pages) != 2 {
		t.Errorf("Expected 2 pages, got %d", len(pages))
	}

	// Verify nested paths are correct
	foundDeep := false

	for _, page := range pages {
		if page.Path == "/subdir/deep.html" {
			foundDeep = true
		}
	}

	if !foundDeep {
		t.Error("Expected to find nested file at /subdir/deep.html")
	}
}

func TestDiscoverHTML_EmptyDirectory(t *testing.T) {
	testdataDir := filepath.Join("..", "..", "testdata", "html", "empty")

	_, err := DiscoverHTML(testdataDir)
	if err == nil {
		t.Error("Expected error for empty directory, got nil")
	}

	if err != nil && err.Error() != "no HTML files found in "+testdataDir {
		t.Errorf("Expected 'no HTML files found' error, got: %v", err)
	}
}

func TestDiscoverHTML_NonexistentDirectory(t *testing.T) {
	_, err := DiscoverHTML("/nonexistent/directory")
	if err == nil {
		t.Error("Expected error for nonexistent directory, got nil")
	}
}

func TestDiscoverHTML_FileInsteadOfDirectory(t *testing.T) {
	// Create a temp file
	tmpFile := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(tmpFile, []byte("test"), 0o600); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	_, err := DiscoverHTML(tmpFile)
	if err == nil {
		t.Error("Expected error when passing file instead of directory")
	}
}

func TestDiscoverHTML_OnlyNonHTMLFiles(t *testing.T) {
	// Create directory with only non-HTML files
	tmpDir := t.TempDir()
	writeTestFile(t, filepath.Join(tmpDir, "readme.txt"), []byte("test"))
	writeTestFile(t, filepath.Join(tmpDir, "style.css"), []byte("body {}"))
	writeTestFile(t, filepath.Join(tmpDir, "script.js"), []byte("console.log()"))

	_, err := DiscoverHTML(tmpDir)
	if err == nil {
		t.Error("Expected error when no HTML files exist")
	}
}

func TestDiscoverHTML_MixedFiles(t *testing.T) {
	// Create directory with mixed file types
	tmpDir := t.TempDir()
	writeTestFile(t, filepath.Join(tmpDir, "page1.html"), []byte("<html></html>"))
	writeTestFile(t, filepath.Join(tmpDir, "page2.htm"), []byte("<html></html>"))
	writeTestFile(t, filepath.Join(tmpDir, "readme.txt"), []byte("test"))
	writeTestFile(t, filepath.Join(tmpDir, "style.css"), []byte("body {}"))

	pages, err := DiscoverHTML(tmpDir)
	if err != nil {
		t.Fatalf("Failed to discover HTML: %v", err)
	}

	// Should find only 2 HTML files
	if len(pages) != 2 {
		t.Errorf("Expected 2 HTML files, got %d", len(pages))
	}
}

func TestDiscoverHTML_CaseInsensitiveExtensions(t *testing.T) {
	// Create files with different case extensions
	tmpDir := t.TempDir()
	writeTestFile(t, filepath.Join(tmpDir, "page1.HTML"), []byte("<html></html>"))
	writeTestFile(t, filepath.Join(tmpDir, "page2.Html"), []byte("<html></html>"))
	writeTestFile(t, filepath.Join(tmpDir, "page3.HTM"), []byte("<html></html>"))

	pages, err := DiscoverHTML(tmpDir)
	if err != nil {
		t.Fatalf("Failed to discover HTML: %v", err)
	}

	// Should find all 3 files regardless of case
	if len(pages) != 3 {
		t.Errorf("Expected 3 HTML files with mixed case, got %d", len(pages))
	}
}

func TestDiscoverHTML_IDSequencing(t *testing.T) {
	// Create directory with multiple HTML files
	tmpDir := t.TempDir()
	for i := 1; i <= 5; i++ {
		filename := filepath.Join(tmpDir, "page"+string(rune('0'+i))+".html")
		writeTestFile(t, filename, []byte("<html></html>"))
	}

	pages, err := DiscoverHTML(tmpDir)
	if err != nil {
		t.Fatalf("Failed to discover HTML: %v", err)
	}

	if len(pages) != 5 {
		t.Fatalf("Expected 5 pages, got %d", len(pages))
	}

	// Verify IDs are sequential
	for i, page := range pages {
		expectedID := fmt.Sprintf("page-%03d", i+1)

		if page.ID != expectedID {
			t.Errorf("Expected ID %s, got %s", expectedID, page.ID)
		}
	}
}

func TestDiscoverHTML_PathFormat(t *testing.T) {
	// Create nested structure
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "sub", "nested")
	makeDir(t, subDir)
	writeTestFile(t, filepath.Join(subDir, "page.html"), []byte("<html></html>"))

	pages, err := DiscoverHTML(tmpDir)
	if err != nil {
		t.Fatalf("Failed to discover HTML: %v", err)
	}

	if len(pages) != 1 {
		t.Fatalf("Expected 1 page, got %d", len(pages))
	}

	// Verify path uses forward slashes (URL format)
	expectedPath := "/sub/nested/page.html"
	if pages[0].Path != expectedPath {
		t.Errorf("Expected path %s, got %s", expectedPath, pages[0].Path)
	}
}

func writeTestFile(t *testing.T, path string, data []byte) {
	t.Helper()

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
}

func makeDir(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(path, 0o750); err != nil {
		t.Fatalf("failed to create directory %s: %v", path, err)
	}
}

func writeBenchFile(b *testing.B, path string, data []byte) {
	b.Helper()

	if err := os.WriteFile(path, data, 0o600); err != nil {
		b.Fatalf("failed to write file %s: %v", path, err)
	}
}

func makeBenchDir(b *testing.B, path string) {
	b.Helper()

	if err := os.MkdirAll(path, 0o750); err != nil {
		b.Fatalf("failed to create directory %s: %v", path, err)
	}
}

func TestDiscoverHTML_SymlinksIgnored(t *testing.T) {
	// Create directory with symlink
	tmpDir := t.TempDir()

	// Create original file
	originalFile := filepath.Join(tmpDir, "original.html")
	writeTestFile(t, originalFile, []byte("<html></html>"))

	// Create symlink (skip test if symlinks not supported)
	symlinkFile := filepath.Join(tmpDir, "link.html")
	if err := os.Symlink(originalFile, symlinkFile); err != nil {
		t.Skip("Symlinks not supported on this platform")
	}

	pages, err := DiscoverHTML(tmpDir)
	if err != nil {
		t.Fatalf("Failed to discover HTML: %v", err)
	}

	if len(pages) != 1 {
		t.Fatalf("Expected 1 page (symlink ignored), got %d", len(pages))
	}

	if pages[0].File != originalFile {
		t.Fatalf("Expected discovered file to be original, got %s", pages[0].File)
	}
}

func TestIsHTMLFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"index.html", true},
		{"page.htm", true},
		{"INDEX.HTML", true},
		{"PAGE.HTM", true},
		{"style.css", false},
		{"script.js", false},
		{"readme.txt", false},
		{"image.png", false},
		{"data.json", false},
		{"file.HTML", true},
		{"file.Html", true},
		{"file.HTM", true},
		{"file.Htm", true},
		{"/path/to/file.html", true},
		{"/path/to/file.htm", true},
		{"/path/to/file.txt", false},
		{"noextension", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isHTMLFile(tt.path)
			if result != tt.expected {
				t.Errorf("isHTMLFile(%q) = %v, expected %v", tt.path, result, tt.expected)
			}
		})
	}
}

// Benchmark discovery performance.
func BenchmarkDiscoverHTML_Small(b *testing.B) {
	tmpDir := b.TempDir()
	for i := range 10 {
		filename := filepath.Join(tmpDir, "page"+string(rune('0'+i))+".html")
		writeBenchFile(b, filename, []byte("<html></html>"))
	}

	b.ResetTimer()

	for range b.N {
		_, _ = DiscoverHTML(tmpDir)
	}
}

func BenchmarkDiscoverHTML_Large(b *testing.B) {
	tmpDir := b.TempDir()

	// Create nested structure with many files
	for i := range 100 {
		subDir := filepath.Join(tmpDir, "dir"+string(rune('0'+i%10)))
		makeBenchDir(b, subDir)
		filename := filepath.Join(subDir, "page"+string(rune('0'+i))+".html")
		writeBenchFile(b, filename, []byte("<html></html>"))
	}

	b.ResetTimer()

	for range b.N {
		_, _ = DiscoverHTML(tmpDir)
	}
}
