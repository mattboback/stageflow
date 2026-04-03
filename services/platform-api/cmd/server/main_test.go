package main

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mattboback/stageflow/libs/go/scannerregistry"
)

func TestNewHTTPServer(t *testing.T) {
	t.Parallel()

	handler := http.NewServeMux()
	port := 8080

	server := newHTTPServer(port, handler)

	if server == nil {
		t.Fatal("expected server to be non-nil")
	}

	expectedAddr := ":8080"
	if server.Addr != expectedAddr {
		t.Errorf("expected Addr %s, got %s", expectedAddr, server.Addr)
	}

	if server.Handler != handler {
		t.Error("expected handler to match provided handler")
	}

	if server.ReadHeaderTimeout != 5*time.Second {
		t.Errorf("expected ReadHeaderTimeout 5s, got %v", server.ReadHeaderTimeout)
	}

	if server.ReadTimeout != 15*time.Second {
		t.Errorf("expected ReadTimeout 15s, got %v", server.ReadTimeout)
	}

	if server.WriteTimeout != 0 {
		t.Errorf("expected WriteTimeout 0 (disabled for SSE), got %v", server.WriteTimeout)
	}

	if server.IdleTimeout != 60*time.Second {
		t.Errorf("expected IdleTimeout 60s, got %v", server.IdleTimeout)
	}
}

func TestNewHTTPServer_DifferentPorts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		port         int
		expectedAddr string
	}{
		{port: 80, expectedAddr: ":80"},
		{port: 3000, expectedAddr: ":3000"},
		{port: 9999, expectedAddr: ":9999"},
	}

	for _, tt := range tests {
		t.Run(tt.expectedAddr, func(t *testing.T) {
			t.Parallel()

			handler := http.NewServeMux()
			server := newHTTPServer(tt.port, handler)

			if server.Addr != tt.expectedAddr {
				t.Errorf("expected Addr %s, got %s", tt.expectedAddr, server.Addr)
			}
		})
	}
}

func TestLoadScannerOverrides_EmptyPath(_ *testing.T) {
	logger := slog.Default()

	overrides := loadScannerOverrides(logger, "")

	// With empty path, should try working directory
	// Result could be nil if no overrides file exists
	_ = overrides // No error expected
}

func TestLoadScannerOverrides_NonexistentPath(t *testing.T) {
	logger := slog.Default()

	overrides := loadScannerOverrides(logger, "/nonexistent/path/to/config.yaml")

	// Should return nil and log warning
	if overrides != nil {
		t.Error("expected nil overrides for nonexistent path")
	}
}

func TestLoadScannerOverrides_ValidPath(t *testing.T) {
	logger := slog.Default()

	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "scanner-overrides.yaml")

	configContent := `
scanners:
  test-scanner:
    enabled: true
`

	if err := os.WriteFile(configPath, []byte(configContent), 0o600); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	overrides := loadScannerOverrides(logger, configPath)

	if overrides == nil {
		t.Fatal("expected overrides to be non-nil for valid config")
	}

	if len(overrides.Scanners) != 1 {
		t.Errorf("expected 1 scanner override, got %d", len(overrides.Scanners))
	}

	if _, ok := overrides.Scanners["test-scanner"]; !ok {
		t.Error("expected test-scanner to be in overrides")
	}
}

func TestLoadScannerRegistry_WithNilOverrides(t *testing.T) {
	logger := slog.Default()

	registry := loadScannerRegistry(logger, "")

	if registry == nil {
		t.Fatal("expected registry to be non-nil")
	}

	// Should have default scanners loaded
	if registry.Count() == 0 {
		t.Error("expected registry to have at least one scanner")
	}
}

func TestLoadScannerRegistry_WithInvalidConfig(t *testing.T) {
	logger := slog.Default()

	// Create temporary invalid config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	invalidContent := `
this is not valid yaml content [[[
`

	if err := os.WriteFile(configPath, []byte(invalidContent), 0o600); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	registry := loadScannerRegistry(logger, configPath)

	// Should fall back to defaults
	if registry == nil {
		t.Fatal("expected registry to fall back to defaults")
	}
}

func TestLoadScannerRegistry_AppliesOverrides(t *testing.T) {
	logger := slog.Default()

	// Create temporary config file with overrides
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "scanner-overrides.yaml")

	configContent := `
scanners:
  axe:
    enabled: false
`

	if err := os.WriteFile(configPath, []byte(configContent), 0o600); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	registry := loadScannerRegistry(logger, configPath)

	if registry == nil {
		t.Fatal("expected registry to be non-nil")
	}

	// Check that override was applied
	axe, ok := registry.Get("axe")
	if !ok {
		t.Fatal("expected axe scanner to exist in registry")
	}

	if axe.Enabled {
		t.Error("expected axe scanner to be disabled by override")
	}
}

func TestLoadScannerRegistry_CountsScannersCorrectly(t *testing.T) {
	logger := slog.Default()

	registry := loadScannerRegistry(logger, "")

	if registry == nil {
		t.Fatal("expected registry to be non-nil")
	}

	totalCount := registry.Count()
	enabledCount := registry.CountEnabled()

	if totalCount == 0 {
		t.Error("expected at least one scanner in registry")
	}

	if enabledCount > totalCount {
		t.Errorf("enabled count (%d) cannot exceed total count (%d)", enabledCount, totalCount)
	}
}

func TestLoadScannerOverrides_WhitespacePath(_ *testing.T) {
	logger := slog.Default()

	overrides := loadScannerOverrides(logger, "   ")

	// Should treat as empty path and try working directory
	_ = overrides // No error expected
}

func TestNewHTTPServer_HandlerIsSet(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := newHTTPServer(8080, mux)

	if server.Handler == nil {
		t.Error("expected handler to be set")
	}

	// Verify handler works
	if server.Handler != mux {
		t.Error("handler should be the same as provided mux")
	}
}

func TestLoadScannerRegistry_FallbackToDefaults(t *testing.T) {
	logger := slog.Default()

	// Use a path that will cause override loading to fail
	registry := loadScannerRegistry(logger, "/nonexistent/config.yaml")

	if registry == nil {
		t.Fatal("expected registry to fall back to defaults")
	}

	// Should have default scanners
	defaultConfig, err := scannerregistry.DefaultConfig()
	if err != nil {
		t.Fatalf("failed to load default config: %v", err)
	}

	if registry.Count() != len(defaultConfig.Scanners) {
		t.Errorf("expected %d scanners from defaults, got %d", len(defaultConfig.Scanners), registry.Count())
	}
}

func TestNewHTTPServer_TimeoutsForSSE(t *testing.T) {
	t.Parallel()

	server := newHTTPServer(8080, http.NewServeMux())

	// WriteTimeout should be 0 to support long-lived SSE connections
	if server.WriteTimeout != 0 {
		t.Errorf("WriteTimeout should be 0 for SSE support, got %v", server.WriteTimeout)
	}

	// Other timeouts should be set for security
	if server.ReadHeaderTimeout == 0 {
		t.Error("ReadHeaderTimeout should be set to prevent slowloris attacks")
	}

	if server.ReadTimeout == 0 {
		t.Error("ReadTimeout should be set")
	}

	if server.IdleTimeout == 0 {
		t.Error("IdleTimeout should be set to clean up idle connections")
	}
}

func TestLoadScannerOverrides_WorkingDirectory(t *testing.T) {
	logger := slog.Default()

	// Save current directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	defer func() {
		if chdirErr := os.Chdir(origDir); chdirErr != nil {
			t.Errorf("failed to restore working directory: %v", chdirErr)
		}
	}()

	// Create temp directory with config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "scanners.yaml")

	configContent := `
scanners:
  wd-test:
    enabled: true
`

	if writeErr := os.WriteFile(configPath, []byte(configContent), 0o600); writeErr != nil {
		t.Fatalf("failed to create test config: %v", writeErr)
	}

	// Change to temp directory
	if chdirErr := os.Chdir(tmpDir); chdirErr != nil {
		t.Fatalf("failed to change directory: %v", chdirErr)
	}

	// Load with empty path - should find config in working directory
	overrides := loadScannerOverrides(logger, "")

	if overrides == nil {
		t.Fatal("expected overrides to be loaded from working directory")
	}

	if len(overrides.Scanners) != 1 {
		t.Errorf("expected 1 scanner override, got %d", len(overrides.Scanners))
	}

	if _, ok := overrides.Scanners["wd-test"]; !ok {
		t.Error("expected wd-test to be in overrides")
	}
}
