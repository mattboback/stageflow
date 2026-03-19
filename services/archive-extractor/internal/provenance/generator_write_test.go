package provenance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mattboback/stageflow/libs/go/models"
	"github.com/mattboback/stageflow/services/archive-extractor/internal/discovery"
)

func TestWriteToFile_ValidJSON(t *testing.T) {
	gen := NewGenerator()

	pages := []discovery.HTMLPage{
		{ID: "page-001", Path: "/index.html", File: "/workspace/site/index.html"},
	}

	jobID := "test-job-123"
	baseURL := "http://localhost:8080"
	outputPath := filepath.Join(t.TempDir(), "provenance.json")

	_, err := gen.Generate(jobID, baseURL, pages, outputPath)
	if err != nil {
		t.Fatalf("Failed to generate provenance: %v", err)
	}

	if _, statErr := os.Stat(outputPath); os.IsNotExist(statErr) {
		t.Fatalf("Provenance file was not created")
	}

	data, err := os.ReadFile(outputPath) // #nosec G304 -- reading controlled temp file
	if err != nil {
		t.Fatalf("Failed to read provenance file: %v", err)
	}

	var prov models.Provenance
	if unmarshalErr := json.Unmarshal(data, &prov); unmarshalErr != nil {
		t.Fatalf("Failed to parse provenance JSON: %v", unmarshalErr)
	}

	if prov.JobID != jobID {
		t.Errorf("Expected JobID %s, got %s", jobID, prov.JobID)
	}
}

func TestWriteToFile_Formatted(t *testing.T) {
	gen := NewGenerator()

	pages := []discovery.HTMLPage{
		{ID: "page-001", Path: "/index.html", File: "/workspace/site/index.html"},
	}

	outputPath := filepath.Join(t.TempDir(), "provenance.json")

	_, err := gen.Generate("test-job", "http://localhost:8080", pages, outputPath)
	if err != nil {
		t.Fatalf("Failed to generate provenance: %v", err)
	}

	data, err := os.ReadFile(outputPath) // #nosec G304 -- reading controlled temp file
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if !isPrettyPrintedJSON(string(data)) {
		t.Error("Expected pretty-printed JSON with indentation")
	}
}

func TestWriteToFile_InvalidPath(t *testing.T) {
	gen := NewGenerator()

	prov := &models.Provenance{
		Version: "1.0.0",
		JobID:   "test-job",
		BaseURL: "http://localhost:8080",
		Pages:   []models.Page{},
	}

	invalidPath := "/nonexistent/directory/provenance.json"

	err := gen.WriteToFile(prov, invalidPath)
	if err == nil {
		t.Error("Expected error for invalid path, got nil")
	}
}

func TestWriteToFile_PermissionDenied(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	gen := NewGenerator()

	prov := &models.Provenance{
		Version: "1.0.0",
		JobID:   "test-job",
		BaseURL: "http://localhost:8080",
		Pages:   []models.Page{},
	}

	tmpDir := t.TempDir()
	noWriteDir := filepath.Join(tmpDir, "nowrite")
	mustMkdir(t, noWriteDir, 0o555) // Read+execute only

	defer func() {
		_ = os.Chmod(noWriteDir, 0o750) // #nosec G302 -- restore permissions after test
	}()

	outputPath := filepath.Join(noWriteDir, "provenance.json")

	err := gen.WriteToFile(prov, outputPath)
	if err == nil {
		t.Error("Expected permission error, got nil")
	}
}
