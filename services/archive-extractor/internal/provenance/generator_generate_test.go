package provenance

import (
	"path/filepath"
	"testing"

	"github.com/mattboback/stageflow/platform/extractor/internal/discovery"
)

func TestGenerate_Basic(t *testing.T) {
	gen := NewGenerator()

	pages := []discovery.HTMLPage{
		{ID: "page-001", Path: "/index.html", File: "/workspace/site/index.html"},
		{ID: "page-002", Path: "/about.html", File: "/workspace/site/about.html"},
	}

	jobID := "test-job-123"
	baseURL := "http://localhost:8080"
	outputPath := filepath.Join(t.TempDir(), "provenance.json")

	prov, err := gen.Generate(jobID, baseURL, pages, outputPath)
	if err != nil {
		t.Fatalf("Failed to generate provenance: %v", err)
	}

	if prov.Version == "" {
		t.Error("Expected Version to be set")
	}

	if prov.JobID != jobID {
		t.Errorf("Expected JobID %s, got %s", jobID, prov.JobID)
	}

	if prov.BaseURL != baseURL {
		t.Errorf("Expected BaseURL %s, got %s", baseURL, prov.BaseURL)
	}

	if len(prov.Pages) != 2 {
		t.Fatalf("Expected 2 pages, got %d", len(prov.Pages))
	}

	if prov.Pages[0].ID != "page-001" {
		t.Errorf("Expected page ID page-001, got %s", prov.Pages[0].ID)
	}

	if prov.Pages[0].Path != "/index.html" {
		t.Errorf("Expected path /index.html, got %s", prov.Pages[0].Path)
	}

	expectedURL := "http://localhost:8080/index.html"
	if prov.Pages[0].URL != expectedURL {
		t.Errorf("Expected URL %s, got %s", expectedURL, prov.Pages[0].URL)
	}
}

func TestGenerate_EmptyPages(t *testing.T) {
	gen := NewGenerator()

	jobID := "test-job-123"
	baseURL := "http://localhost:8080"
	outputPath := filepath.Join(t.TempDir(), "provenance.json")

	prov, err := gen.Generate(jobID, baseURL, nil, outputPath)
	if err != nil {
		t.Fatalf("Failed to generate provenance: %v", err)
	}

	if len(prov.Pages) != 0 {
		t.Errorf("Expected 0 pages, got %d", len(prov.Pages))
	}
}

func TestGenerate_NestedPaths(t *testing.T) {
	gen := NewGenerator()

	pages := []discovery.HTMLPage{
		{ID: "page-001", Path: "/index.html", File: "/workspace/site/index.html"},
		{ID: "page-002", Path: "/docs/guide.html", File: "/workspace/site/docs/guide.html"},
		{ID: "page-003", Path: "/docs/api/reference.html", File: "/workspace/site/docs/api/reference.html"},
	}

	outputPath := filepath.Join(t.TempDir(), "provenance.json")

	prov, err := gen.Generate("test-job-123", "http://localhost:8080", pages, outputPath)
	if err != nil {
		t.Fatalf("Failed to generate provenance: %v", err)
	}

	expectedURLs := []string{
		"http://localhost:8080/index.html",
		"http://localhost:8080/docs/guide.html",
		"http://localhost:8080/docs/api/reference.html",
	}

	for i, expectedURL := range expectedURLs {
		if prov.Pages[i].URL != expectedURL {
			t.Errorf("Expected URL %s, got %s", expectedURL, prov.Pages[i].URL)
		}
	}
}

func TestGenerate_DifferentBaseURL(t *testing.T) {
	gen := NewGenerator()

	pages := []discovery.HTMLPage{
		{ID: "page-001", Path: "/index.html", File: "/workspace/site/index.html"},
	}

	tests := []struct {
		name    string
		baseURL string
		wantURL string
	}{
		{
			name:    "localhost with custom port",
			baseURL: "http://localhost:3000",
			wantURL: "http://localhost:3000/index.html",
		},
		{
			name:    "IP address",
			baseURL: "http://192.168.1.10:8080",
			wantURL: "http://192.168.1.10:8080/index.html",
		},
		{
			name:    "HTTPS",
			baseURL: "https://example.com",
			wantURL: "https://example.com/index.html",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputPath := filepath.Join(t.TempDir(), "provenance.json")

			prov, err := gen.Generate("test-job", tt.baseURL, pages, outputPath)
			if err != nil {
				t.Fatalf("Failed to generate provenance: %v", err)
			}

			if prov.Pages[0].URL != tt.wantURL {
				t.Errorf("Expected URL %s, got %s", tt.wantURL, prov.Pages[0].URL)
			}
		})
	}
}

func TestGenerate_LargeNumberOfPages(t *testing.T) {
	gen := NewGenerator()

	pages := make([]discovery.HTMLPage, 1000)
	for i := range 1000 {
		pages[i] = discovery.HTMLPage{
			ID:   "page-001",
			Path: "/page.html",
			File: "/workspace/site/page.html",
		}
	}

	outputPath := filepath.Join(t.TempDir(), "provenance.json")

	prov, err := gen.Generate("test-job-123", "http://localhost:8080", pages, outputPath)
	if err != nil {
		t.Fatalf("Failed to generate provenance: %v", err)
	}

	if len(prov.Pages) != 1000 {
		t.Errorf("Expected 1000 pages, got %d", len(prov.Pages))
	}
}
