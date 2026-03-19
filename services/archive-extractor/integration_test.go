//go:build integration
// +build integration

package main

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mattboback/stageflow/libs/go/models"
	"github.com/mattboback/stageflow/platform/extractor/internal/discovery"
	"github.com/mattboback/stageflow/platform/extractor/internal/extractor"
	"github.com/mattboback/stageflow/platform/extractor/internal/provenance"
	"github.com/mattboback/stageflow/platform/extractor/internal/server"
)

func TestIntegration_EndToEnd(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION") != "" {
		t.Skip("Skipping integration test")
	}

	workspace := t.TempDir()
	siteDir := filepath.Join(workspace, "site")

	zipPath := createTestSiteZIP(t, workspace)

	if err := extractor.ExtractZIPToDir(zipPath, siteDir); err != nil {
		t.Fatalf("Failed to extract ZIP: %v", err)
	}

	if _, err := os.Stat(filepath.Join(siteDir, "index.html")); os.IsNotExist(err) {
		t.Fatal("Expected index.html to be extracted")
	}

	pages, err := discovery.DiscoverHTML(siteDir)
	if err != nil {
		t.Fatalf("Failed to discover HTML: %v", err)
	}
	if len(pages) < 1 {
		t.Fatal("Expected at least 1 HTML page to be discovered")
	}

	siteServer := server.NewStaticServer(&server.Config{
		SiteDir: siteDir,
		Addr:    "127.0.0.1:0",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := siteServer.Start(ctx); err != nil {
		t.Fatalf("Failed to start static server: %v", err)
	}
	defer func() { _ = siteServer.Stop(context.Background()) }()

	listenAddr := siteServer.ListenerAddr()
	if listenAddr == "" {
		t.Fatal("Expected server to expose listener address")
	}
	baseURL := "http://" + listenAddr

	waitForHTTP200(t, baseURL+"/index.html", 5*time.Second)

	gen := provenance.NewGenerator()
	provenancePath := filepath.Join(workspace, "provenance.json")

	prov, err := gen.Generate("integration-test", baseURL, pages, provenancePath)
	if err != nil {
		t.Fatalf("Failed to generate provenance: %v", err)
	}

	data, err := os.ReadFile(provenancePath) // #nosec G304 -- controlled temp file
	if err != nil {
		t.Fatalf("Failed to read provenance: %v", err)
	}
	var parsedProv models.Provenance
	if err := json.Unmarshal(data, &parsedProv); err != nil {
		t.Fatalf("Provenance is not valid JSON: %v", err)
	}

	client := &http.Client{Timeout: 5 * time.Second}

	for _, page := range prov.Pages {
		resp, err := client.Get(page.URL)
		if err != nil {
			t.Errorf("Failed to fetch %s: %v", page.URL, err)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 for %s, got %d", page.URL, resp.StatusCode)
		}
		if len(body) == 0 {
			t.Errorf("Empty response for %s", page.URL)
		}
	}

	resp, err := client.Get(baseURL + "/index.html")
	if err != nil {
		t.Fatalf("Failed to make CORS check request: %v", err)
	}
	_ = resp.Body.Close()

	if corsOrigin := resp.Header.Get("Access-Control-Allow-Origin"); corsOrigin != "*" {
		t.Errorf("Expected CORS origin *, got %s", corsOrigin)
	}
}

func TestIntegration_ZIPSecurity(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION") != "" {
		t.Skip("Skipping integration test")
	}

	workspace := t.TempDir()
	siteDir := filepath.Join(workspace, "site")

	testdataPath := filepath.Join("testdata", "path-traversal.zip")
	if _, err := os.Stat(testdataPath); os.IsNotExist(err) {
		t.Skip("Test ZIP not available")
	}

	err := extractor.ExtractZIPToDir(testdataPath, siteDir)
	if err == nil {
		t.Fatal("Expected path traversal ZIP to be rejected")
	}

	if _, statErr := os.Stat(filepath.Join(workspace, "etc", "evil.html")); !os.IsNotExist(statErr) {
		t.Fatal("Path traversal attack succeeded - malicious file created")
	}
}

func waitForHTTP200(t *testing.T, url string, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 500 * time.Millisecond}

	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
			lastErr = fmt.Errorf("status %d", resp.StatusCode)
		} else {
			lastErr = err
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("server never became ready at %s: %v", url, lastErr)
}

func createTestSiteZIP(t *testing.T, workspace string) string {
	t.Helper()

	zipPath := filepath.Join(workspace, "test-site.zip")
	f, err := os.Create(zipPath) // #nosec G304 -- temp path
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	defer func() { _ = f.Close() }()

	zw := zip.NewWriter(f)
	addZipFile(t, zw, "index.html", "<html><body>Index</body></html>")
	addZipFile(t, zw, "about.html", "<html><body>About</body></html>")
	addZipFile(t, zw, "docs/guide.html", "<html><body>Guide</body></html>")

	if err := zw.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close zip file: %v", err)
	}

	return zipPath
}

func addZipFile(t *testing.T, zw *zip.Writer, name, contents string) {
	t.Helper()

	w, err := zw.Create(name)
	if err != nil {
		t.Fatalf("create zip entry %q: %v", name, err)
	}
	if _, err := w.Write([]byte(contents)); err != nil {
		t.Fatalf("write zip entry %q: %v", name, err)
	}
}
