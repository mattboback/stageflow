package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestNewStaticServerDefaults(t *testing.T) {
	s := NewStaticServer(&Config{SiteDir: "/workspace/site"})
	if s.addr != "127.0.0.1:8080" {
		t.Fatalf("expected default addr 127.0.0.1:8080, got %s", s.addr)
	}
	if s.siteDir != "/workspace/site" {
		t.Fatalf("expected siteDir /workspace/site, got %s", s.siteDir)
	}
}

func TestStaticServerServesFilesAndCORS(t *testing.T) {
	siteDir := t.TempDir()
	indexPath := filepath.Join(siteDir, "index.html")
	if err := os.WriteFile(indexPath, []byte("<html><body>Test</body></html>"), 0o600); err != nil {
		t.Fatalf("write index.html: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := NewStaticServer(&Config{
		SiteDir: siteDir,
		Addr:    "127.0.0.1:0",
	})
	if err := s.Start(ctx); err != nil {
		if errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES) {
			t.Skipf("skipping: cannot bind to local TCP port in this environment: %v", err)
		}

		t.Fatalf("start: %v", err)
	}
	defer func() { _ = s.Stop(context.Background()) }()

	host, port, err := net.SplitHostPort(s.listener.Addr().String())
	if err != nil {
		t.Fatalf("split addr: %v", err)
	}
	if host == "" {
		host = "127.0.0.1"
	}

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, fmt.Sprintf("http://%s:%s/index.html", host, port), http.NoBody)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected CORS origin '*', got %q", got)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if len(body) == 0 {
		t.Fatalf("expected body content")
	}
}
