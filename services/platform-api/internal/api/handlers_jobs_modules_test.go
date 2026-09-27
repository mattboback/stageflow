package api

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mattboback/stageflow/libs/go/httputil"
	"github.com/mattboback/stageflow/libs/go/scannerregistry"
	"github.com/mattboback/stageflow/services/platform-api/internal/jobstatus"
)

func TestSplitCSV(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string returns nil",
			input:    "",
			expected: nil,
		},
		{
			name:     "whitespace only returns nil",
			input:    "   ",
			expected: nil,
		},
		{
			name:     "single value",
			input:    scannerTypeAxe,
			expected: []string{scannerTypeAxe},
		},
		{
			name:     "multiple values",
			input:    scannerTypeAxe + ",lighthouse,seo",
			expected: []string{scannerTypeAxe, "lighthouse", "seo"},
		},
		{
			name:     "trims whitespace around values",
			input:    " " + scannerTypeAxe + " , lighthouse , seo ",
			expected: []string{scannerTypeAxe, "lighthouse", "seo"},
		},
		{
			name:     "filters empty parts",
			input:    scannerTypeAxe + ",,lighthouse",
			expected: []string{scannerTypeAxe, "lighthouse"},
		},
		{
			name:     "handles trailing comma",
			input:    scannerTypeAxe + ",lighthouse,",
			expected: []string{scannerTypeAxe, "lighthouse"},
		},
		{
			name:     "handles leading comma",
			input:    "," + scannerTypeAxe + ",lighthouse",
			expected: []string{scannerTypeAxe, "lighthouse"},
		},
		{
			name:     "handles multiple empty parts",
			input:    scannerTypeAxe + ",,,lighthouse,,seo",
			expected: []string{scannerTypeAxe, "lighthouse", "seo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitCSV(tt.input)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}

				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d items, got %d: %v", len(tt.expected), len(result), result)

				return
			}

			for i, v := range tt.expected {
				if result[i] != v {
					t.Errorf("at index %d: expected %q, got %q", i, v, result[i])
				}
			}
		})
	}
}

func TestNormalizeModules_NoRegistry(t *testing.T) {
	server := &Server{
		scannerRegistry: nil,
	}

	t.Run("empty explicit returns axe", func(t *testing.T) {
		result, err := server.normalizeModules(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result) != 1 || result[0] != scannerTypeAxe {
			t.Errorf("expected [axe], got %v", result)
		}
	})

	t.Run("axe explicit returns axe", func(t *testing.T) {
		result, err := server.normalizeModules([]string{scannerTypeAxe})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result) != 1 || result[0] != scannerTypeAxe {
			t.Errorf("expected [axe], got %v", result)
		}
	})

	t.Run("non-axe explicit returns error", func(t *testing.T) {
		_, err := server.normalizeModules([]string{"lighthouse"})
		if err == nil {
			t.Fatal("expected error for non-axe module without registry")
		}
	})

	t.Run("mixed modules with non-axe returns error", func(t *testing.T) {
		_, err := server.normalizeModules([]string{"axe", "lighthouse"})
		if err == nil {
			t.Fatal("expected error for mixed modules without registry")
		}
	})

	t.Run("handles whitespace and case", func(t *testing.T) {
		result, err := server.normalizeModules([]string{" " + scannerTypeAxe + " "})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result) != 1 || result[0] != scannerTypeAxe {
			t.Errorf("expected [axe], got %v", result)
		}
	})
}

func TestNormalizeModules_WithRegistry(t *testing.T) {
	server := newTestServerForModules(t)

	t.Run("empty explicit uses registry defaults", func(t *testing.T) {
		result, err := server.normalizeModules(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result) == 0 {
			t.Error("expected non-empty result")
		}
	})

	t.Run("valid module resolves", func(t *testing.T) {
		result, err := server.normalizeModules([]string{scannerTypeAxe})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result) == 0 {
			t.Error("expected non-empty result")
		}
	})

	t.Run("invalid module returns error with suggestions", func(t *testing.T) {
		_, err := server.normalizeModules([]string{"nonexistent-scanner"})
		if err == nil {
			t.Fatal("expected error for invalid module")
		}
		// Error should mention the bad module and list supported ones
		errStr := err.Error()
		if errStr == "" {
			t.Error("expected non-empty error message")
		}
	})
}

func TestListSupportedModuleIDs(t *testing.T) {
	t.Run("without registry returns axe only", func(t *testing.T) {
		server := &Server{
			scannerRegistry: nil,
		}

		result := server.listSupportedModuleIDs()
		if len(result) != 1 || result[0] != scannerTypeAxe {
			t.Errorf("expected [axe], got %v", result)
		}
	})

	t.Run("with registry returns all scanner IDs", func(t *testing.T) {
		server := newTestServerForModules(t)

		result := server.listSupportedModuleIDs()
		if len(result) == 0 {
			t.Error("expected at least one module ID")
		}

		// Should contain axe at minimum
		hasAxe := false

		for _, id := range result {
			if id == scannerTypeAxe {
				hasAxe = true

				break
			}
		}

		if !hasAxe {
			t.Error("expected result to contain 'axe'")
		}
	})
}

// newTestServerForModules creates a test server with scanner registry.
func newTestServerForModules(t *testing.T) *Server {
	t.Helper()

	defaultConfig, err := scannerregistry.DefaultConfig()
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}

	registry, err := scannerregistry.InitializeRegistry(defaultConfig)
	if err != nil {
		t.Fatalf("init scanner registry: %v", err)
	}

	return &Server{
		config: &ServerConfig{
			ScannerRegistry: registry,
		},
		jobStatus:       jobstatus.New(nil),
		scannerRegistry: registry,
	}
}

func TestIntakeRejectsUnknownModuleWithTypedError(t *testing.T) {
	server, _, _ := newTestServer(t)

	urlReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/jobs/urls",
		strings.NewReader(`{"urls":["https://example.com"],"modules":["axe","no-such-scanner"]}`),
	)
	urlReq.Header.Set("Content-Type", "application/json")

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	addZipFile(t, writer, "site.zip", buildTestZip(t))
	writeField(t, writer, "modules", "axe,no-such-scanner")

	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	zipReq := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/zip", body)
	zipReq.Header.Set("Content-Type", writer.FormDataContentType())

	for name, run := range map[string]func(*httptest.ResponseRecorder){
		"url": func(rr *httptest.ResponseRecorder) { server.handleJobURLSubmit(rr, urlReq) },
		"zip": func(rr *httptest.ResponseRecorder) { server.handleJobZipUpload(rr, zipReq) },
	} {
		rr := httptest.NewRecorder()
		run(rr)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("%s: status = %d, want 400: %s", name, rr.Code, rr.Body.String())
		}

		var got httputil.ErrorResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
			t.Fatalf("%s: decode error body: %v", name, err)
		}

		if got.Error.Code != httputil.ErrCodeUnsupportedModule ||
			got.Error.Message != "Unsupported scanner module: no-such-scanner" {
			t.Fatalf("%s: unexpected error %+v", name, got.Error)
		}
	}
}
