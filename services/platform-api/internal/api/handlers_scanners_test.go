// Scanner listing endpoint: exercises handlers_scanners.go.

package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// The golden file is production's response on 2026-10-02, byte for byte.
func TestHandleListScanners_MatchesGolden(t *testing.T) {
	server, _, _ := newTestServer(t)

	rr := httptest.NewRecorder()
	server.handleListScanners(rr, httptest.NewRequest(http.MethodGet, "/api/v1/scanners", http.NoBody))

	want, err := os.ReadFile(filepath.Join("testdata", "scanners.golden.json"))
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(rr.Body.Bytes(), want) {
		t.Fatalf("scanner listing changed:\n got: %s\nwant: %s", rr.Body.Bytes(), want)
	}
}

func TestHandleListScanners_CountsCatalog(t *testing.T) {
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
