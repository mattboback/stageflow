package extractor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractLocalZIP(t *testing.T) {
	zipPath := createTestZIP(t, map[string]string{"index.html": "<html>Index</html>"})

	dest := t.TempDir()
	if err := NewExtractor().Extract(context.Background(), zipPath, dest); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dest, "index.html"))
	if err != nil || string(data) != "<html>Index</html>" {
		t.Fatalf("invalid extracted content %s: %v", data, err)
	}
}
func TestExtractRejectsOversizedLocalZIP(t *testing.T) {
	input := filepath.Join(t.TempDir(), "large.zip")

	file, err := os.Create(input)
	if err != nil {
		t.Fatal(err)
	}

	if truncateErr := file.Truncate(maxZipCompressedSize + 1); truncateErr != nil {
		t.Fatal(truncateErr)
	}

	_ = file.Close()

	if extractErr := NewExtractor().Extract(context.Background(), input, t.TempDir()); extractErr == nil {
		t.Fatal("oversized ZIP accepted")
	}
}
