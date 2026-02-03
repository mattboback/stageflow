// Package provenance builds provenance metadata for extracted sites.
package provenance

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/mattboback/stageflow/packages/shared-go/models"
	"github.com/mattboback/stageflow/platform/extractor/internal/discovery"
)

// Generator produces provenance.json for scanners and UIs.
type Generator struct{}

const provenanceVersion = "1.0.0"

// NewGenerator returns a stateless provenance generator.
func NewGenerator() *Generator {
	return &Generator{}
}

// Generate assembles provenance from discovery output and writes it to outputPath.
func (g *Generator) Generate(jobID, baseURL string, pages []discovery.HTMLPage, outputPath string) (*models.Provenance, error) {
	u, parseErr := url.Parse(baseURL)
	if parseErr != nil {
		return nil, fmt.Errorf("invalid baseURL %q: %w", baseURL, parseErr)
	}

	provenancePages := make([]models.Page, len(pages))
	for i, page := range pages {
		pageURL := u.ResolveReference(&url.URL{Path: page.Path}).String()
		provenancePages[i] = models.Page{
			ID:   page.ID,
			Path: page.Path,
			URL:  pageURL,
		}
	}

	provenance := &models.Provenance{
		Version: provenanceVersion,
		JobID:   jobID,
		BaseURL: baseURL,
		Pages:   provenancePages,
	}

	if err := g.WriteToFile(provenance, outputPath); err != nil {
		return nil, fmt.Errorf("failed to write provenance file: %w", err)
	}

	return provenance, nil
}

// WriteToFile persists provenance as indented JSON for readable debugging.
func (g *Generator) WriteToFile(provenance *models.Provenance, path string) error {
	data, err := json.MarshalIndent(provenance, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal provenance: %w", err)
	}

	dir := filepath.Dir(path)
	if mkErr := os.MkdirAll(dir, 0o750); mkErr != nil {
		return fmt.Errorf("failed to create output directory: %w", mkErr)
	}

	tmp, err := os.CreateTemp(dir, "provenance-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	defer func(name string) { _ = os.Remove(name) }(tmp.Name())

	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()

		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if chmodErr := os.Chmod(tmpName, 0o600); chmodErr != nil {
		return fmt.Errorf("failed to chmod temp file: %w", chmodErr)
	}

	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("failed to replace provenance file: %w", err)
	}

	return nil
}
