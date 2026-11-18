package models

import (
	"encoding/json"
	"testing"
)

func TestProvenance_Validate(t *testing.T) {
	t.Parallel()

	ok := &Provenance{
		Version: "1.0.0",
		JobID:   "job-123",
		BaseURL: "http://localhost:8080",
		Pages: []Page{
			{ID: "page-001", Path: "/index.html", URL: "http://localhost:8080/index.html"},
			{ID: "page-002", Path: "/about.html", URL: "http://localhost:8080/about.html"},
		},
	}
	if err := ok.Validate(); err != nil {
		t.Fatalf("expected valid provenance: %v", err)
	}

	bad := &Provenance{
		Version: "1.0.0",
		JobID:   "job-123",
		BaseURL: "http://localhost:8080",
		Pages: []Page{
			{ID: "page-001", Path: "index.html", URL: "http://localhost:8080/index.html"},
		},
	}
	if err := bad.Validate(); err == nil {
		t.Fatalf("expected invalid provenance due to page.path missing leading slash")
	}
}

func TestProvenance_JSONTags(t *testing.T) {
	t.Parallel()

	p := &Provenance{
		Version: "1.0.0",
		JobID:   "job-123",
		BaseURL: "http://localhost:8080",
		Pages:   []Page{{ID: "page-001", Path: "/index.html", URL: "http://localhost:8080/index.html"}},
	}

	b, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]any
	if unmarshalErr := json.Unmarshal(b, &m); unmarshalErr != nil {
		t.Fatalf("unmarshal: %v", unmarshalErr)
	}

	for _, k := range []string{"version", "job_id", "base_url", "pages"} {
		if _, ok := m[k]; !ok {
			t.Fatalf("expected key %q in JSON: %s", k, string(b))
		}
	}
}
