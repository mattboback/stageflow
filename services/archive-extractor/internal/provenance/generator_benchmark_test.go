package provenance

import (
	"path/filepath"
	"testing"

	"github.com/mattboback/stageflow/platform/extractor/internal/discovery"
)

func BenchmarkGenerate_Small(b *testing.B) {
	gen := NewGenerator()

	pages := []discovery.HTMLPage{
		{ID: "page-001", Path: "/index.html", File: "/workspace/site/index.html"},
		{ID: "page-002", Path: "/about.html", File: "/workspace/site/about.html"},
	}

	b.ResetTimer()

	for range b.N {
		tmpDir := b.TempDir()

		outputPath := filepath.Join(tmpDir, "provenance.json")
		if _, err := gen.Generate("test-job", "http://localhost:8080", pages, outputPath); err != nil {
			b.Fatalf("failed to generate provenance: %v", err)
		}
	}
}

func BenchmarkGenerate_Large(b *testing.B) {
	gen := NewGenerator()

	pages := make([]discovery.HTMLPage, 1000)
	for i := range 1000 {
		pages[i] = discovery.HTMLPage{
			ID:   "page-001",
			Path: "/page.html",
			File: "/workspace/site/page.html",
		}
	}

	b.ResetTimer()

	for range b.N {
		tmpDir := b.TempDir()

		outputPath := filepath.Join(tmpDir, "provenance.json")
		if _, err := gen.Generate("test-job", "http://localhost:8080", pages, outputPath); err != nil {
			b.Fatalf("failed to generate provenance: %v", err)
		}
	}
}
