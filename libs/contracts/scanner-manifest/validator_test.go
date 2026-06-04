package scannermanifest

import (
	"strings"
	"testing"
)

func TestValidateManifestJSONRejectsMissingRequiredFields(t *testing.T) {
	err := ValidateManifestJSON([]byte(`{"id":"axe"}`))
	if err == nil {
		t.Fatal("expected missing required manifest fields to fail")
	}

	if !strings.Contains(err.Error(), "required") {
		t.Fatalf("expected required-field validation error, got %v", err)
	}
}

func TestValidateManifestJSONRejectsInvalidConfigSchema(t *testing.T) {
	err := ValidateManifestJSON([]byte(`{
		"id": "example",
		"name": "Example",
		"version": "1.0.0",
		"capabilities": {
			"categories": ["custom"],
			"outputFormats": ["json"],
			"supportsScreenshots": false,
			"supportsConcurrency": true,
			"requiresBrowser": false
		},
		"entry": { "module": "./index.js" },
		"configSchema": { "type": 42 }
	}`))
	if err == nil {
		t.Fatal("expected invalid configSchema to fail")
	}

	if !strings.Contains(err.Error(), "configSchema") {
		t.Fatalf("expected configSchema validation error, got %v", err)
	}
}

func TestValidateManifestJSONRejectsScreenshotWithoutBrowser(t *testing.T) {
	err := ValidateManifestJSON([]byte(`{
		"id": "example",
		"name": "Example",
		"version": "1.0.0",
		"capabilities": {
			"categories": ["custom"],
			"outputFormats": ["json"],
			"supportsScreenshots": true,
			"supportsConcurrency": true,
			"requiresBrowser": false
		},
		"entry": { "module": "./index.js" }
	}`))
	if err == nil {
		t.Fatal("expected screenshot support without browser requirement to fail")
	}

	if !strings.Contains(err.Error(), "supportsScreenshots") {
		t.Fatalf("expected capabilities validation error, got %v", err)
	}
}
