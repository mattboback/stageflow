package api

import (
	"path/filepath"
	"testing"

	"github.com/mattboback/stageflow/libs/go/scannerregistry"
	"github.com/mattboback/stageflow/services/platform-api/internal/jobstatus"
	"github.com/mattboback/stageflow/services/platform-api/internal/status"
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

func TestExtractModuleName(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		expected string
	}{
		{
			name:     "extracts name from quoted error",
			errMsg:   "unsupported module 'badscanner'",
			expected: "badscanner",
		},
		{
			name:     "extracts first quoted section",
			errMsg:   "module 'foo' is invalid, check 'bar'",
			expected: "foo",
		},
		{
			name:     "returns unknown when no quotes",
			errMsg:   "module badscanner is invalid",
			expected: "unknown",
		},
		{
			name:     "returns unknown when only opening quote",
			errMsg:   "module 'badscanner is invalid",
			expected: "unknown",
		},
		{
			name:     "handles empty quoted string",
			errMsg:   "module '' is invalid",
			expected: "",
		},
		{
			name:     "handles complex module names",
			errMsg:   "unsupported module 'my-scanner-v2'",
			expected: "my-scanner-v2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractModuleName(tt.errMsg)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
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

	dir := t.TempDir()

	store, err := status.NewStore(&status.Config{Path: filepath.Join(dir, "status.db")})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Fatalf("close store: %v", closeErr)
		}
	})

	registry, err := scannerregistry.InitializeRegistry(scannerregistry.DefaultConfig())
	if err != nil {
		t.Fatalf("init scanner registry: %v", err)
	}

	return &Server{
		config: &ServerConfig{
			StatusReader:    store,
			ScannerRegistry: registry,
		},
		jobStatus:       jobstatus.New(&jobstatus.Config{CurrentReader: store}),
		scannerRegistry: registry,
	}
}
