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
	for i := 0; i < b.N; i++ {
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
	for i := 0; i < 1000; i++ {
		pages[i] = discovery.HTMLPage{
			ID:   "page-001",
			Path: "/page.html",
			File: "/workspace/site/page.html",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tmpDir := b.TempDir()
		outputPath := filepath.Join(tmpDir, "provenance.json")
		if _, err := gen.Generate("test-job", "http://localhost:8080", pages, outputPath); err != nil {
			b.Fatalf("failed to generate provenance: %v", err)
		}
	}
}
