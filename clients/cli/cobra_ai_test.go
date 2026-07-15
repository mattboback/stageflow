package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
)

func TestFetchProvenanceUsesInjectedHTTPClient(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"rawResults":{"steps":[{"url":"https://example.com","action":{"type":"click"}}]}}]`))
	}))
	defer server.Close()

	artifacts, err := json.Marshal(map[string]any{
		"scanner_artifacts": map[string]any{
			"ai-navigator": map[string]any{"results_json": server.URL},
		},
	})
	if err != nil {
		t.Fatalf("marshal artifacts: %v", err)
	}

	full, compressed := fetchProvenance(
		context.Background(),
		server.Client(),
		apiclient.JobStatus{Artifacts: artifacts},
	)
	if len(full) != 1 {
		t.Fatalf("full provenance length = %d", len(full))
	}

	if len(compressed) != 1 || compressed[0].URL != "https://example.com" || compressed[0].Action != "click" {
		t.Fatalf("compressed provenance = %#v", compressed)
	}
}
