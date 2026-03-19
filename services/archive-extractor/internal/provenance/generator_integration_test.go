package provenance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mattboback/stageflow/libs/go/models"
	"github.com/mattboback/stageflow/platform/extractor/internal/discovery"
)

func TestGenerate_Integration(t *testing.T) {
	gen := NewGenerator()

	tmpDir := t.TempDir()
	siteDir := filepath.Join(tmpDir, "site")
	mustMkdirAll(t, siteDir, 0o750)

	htmlFiles := map[string]string{
		"index.html":       "<html><body>Index</body></html>",
		"about.html":       "<html><body>About</body></html>",
		"docs/guide.html":  "<html><body>Guide</body></html>",
		"docs/api/ref.htm": "<html><body>Reference</body></html>",
	}

	for path, content := range htmlFiles {
		fullPath := filepath.Join(siteDir, path)
		mustMkdirAll(t, filepath.Dir(fullPath), 0o750)
		mustWriteFile(t, fullPath, []byte(content))
	}

	pages, err := discovery.DiscoverHTML(siteDir)
	if err != nil {
		t.Fatalf("Failed to discover HTML: %v", err)
	}

	provenancePath := filepath.Join(tmpDir, "provenance.json")

	prov, err := gen.Generate("integration-test", "http://localhost:8080", pages, provenancePath)
	if err != nil {
		t.Fatalf("Failed to generate provenance: %v", err)
	}

	if len(prov.Pages) != 4 {
		t.Errorf("Expected 4 pages, got %d", len(prov.Pages))
	}

	data, err := os.ReadFile(provenancePath) // #nosec G304 -- reading controlled temp file
	if err != nil {
		t.Fatalf("Failed to read provenance: %v", err)
	}

	var parsedProv models.Provenance
	if unmarshalErr := json.Unmarshal(data, &parsedProv); unmarshalErr != nil {
		t.Fatalf("Failed to parse provenance: %v", unmarshalErr)
	}

	if len(parsedProv.Pages) != 4 {
		t.Errorf("Parsed provenance has wrong pages length: %d", len(parsedProv.Pages))
	}
}
