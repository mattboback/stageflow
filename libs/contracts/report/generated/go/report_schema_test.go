package report

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestUnifiedReportRejectsMissingRequiredFields(t *testing.T) {
	var doc UnifiedReportV2

	err := json.Unmarshal([]byte(`{"version":"2.0.0"}`), &doc)
	if err == nil {
		t.Fatal("expected missing required report fields to fail")
	}

	if !strings.Contains(err.Error(), "required") {
		t.Fatalf("expected required-field error, got %v", err)
	}
}

func TestBoundingBoxRejectsNegativeDimensions(t *testing.T) {
	var box BoundingBox

	err := json.Unmarshal([]byte(`{"x":0,"y":0,"width":10,"height":-1}`), &box)
	if err == nil {
		t.Fatal("expected negative bounding-box height to fail")
	}

	if !strings.Contains(err.Error(), "height") {
		t.Fatalf("expected height validation error, got %v", err)
	}
}
