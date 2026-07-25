// Scanner listing endpoint: exercises handlers_scanners.go.

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleListScanners_WithRegistry(t *testing.T) {
	server, _, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scanners", http.NoBody)
	rr := httptest.NewRecorder()

	server.handleListScanners(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp struct {
		Total   int `json:"total"`
		Enabled int `json:"enabled"`
	}

	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Total == 0 {
		t.Fatal("expected at least one scanner, got total=0")
	}

	if resp.Enabled == 0 {
		t.Fatal("expected at least one enabled scanner")
	}
}

func TestHandleListScanners_NilRegistry(t *testing.T) {
	server := &Server{
		config:          &ServerConfig{},
		scannerRegistry: nil,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scanners", http.NoBody)
	rr := httptest.NewRecorder()

	server.handleListScanners(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp struct {
		Scanners []struct {
			ID string `json:"id"`
		} `json:"scanners"`
		Total int `json:"total"`
	}

	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Total != 1 {
		t.Fatalf("expected fallback total=1, got %d", resp.Total)
	}

	if resp.Scanners[0].ID != "axe" {
		t.Fatalf("expected fallback scanner id=axe, got %q", resp.Scanners[0].ID)
	}
}

func TestHandleListScanners_MethodNotAllowed(t *testing.T) {
	server, _, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/scanners", http.NoBody)
	rr := httptest.NewRecorder()

	server.handleListScanners(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

// --- URL submit behavior ---
