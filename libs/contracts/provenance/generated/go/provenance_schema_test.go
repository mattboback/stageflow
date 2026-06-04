package provenance

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestProvenanceRejectsMissingRequiredFields(t *testing.T) {
	var doc Provenance

	err := json.Unmarshal([]byte(`{"version":"1.0.0"}`), &doc)
	if err == nil {
		t.Fatal("expected missing required provenance fields to fail")
	}

	if !strings.Contains(err.Error(), "required") {
		t.Fatalf("expected required-field error, got %v", err)
	}
}

func TestPageEntryRejectsBlankID(t *testing.T) {
	var page PageEntry

	err := json.Unmarshal([]byte(`{"id":""}`), &page)
	if err == nil {
		t.Fatal("expected blank page id to fail")
	}

	if !strings.Contains(err.Error(), "id") {
		t.Fatalf("expected id validation error, got %v", err)
	}
}
