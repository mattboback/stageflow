package main

import (
	"context"
	"strings"
	"testing"

	"github.com/mattboback/stageflow/services/archive-extractor/internal/server"
)

func TestServerLossAfterReadinessFails(t *testing.T) {
	ctx := context.Background()

	site := server.NewStaticServer(&server.Config{SiteDir: t.TempDir(), Port: "0"})
	if err := site.Start(ctx); err != nil {
		t.Fatal(err)
	}

	if err := site.Stop(ctx); err != nil {
		t.Fatal(err)
	}

	err := waitForServer(ctx, site)
	if err == nil || !strings.Contains(err.Error(), "before scanners finished") {
		t.Fatalf("server loss accepted: %v", err)
	}
}
func TestServerRemainsUntilCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	site := server.NewStaticServer(&server.Config{SiteDir: t.TempDir(), Port: "0"})
	if err := site.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = site.Stop(context.Background()) }()

	cancel()

	if err := waitForServer(ctx, site); err != nil {
		t.Fatal(err)
	}
}

func TestLoadConfigUsesChildOfWritableSiteMount(t *testing.T) {
	t.Setenv("SITE_DIR", "")
	t.Setenv("INPUT_PATH", "")

	cfg := loadConfig()
	if cfg.SiteDir != "/site/content" || cfg.InputPath != "/job/input.zip" {
		t.Fatalf("defaults: input=%q site=%q", cfg.InputPath, cfg.SiteDir)
	}

	t.Setenv("SITE_DIR", "/site/custom")
	t.Setenv("INPUT_PATH", "/job/custom.zip")

	cfg = loadConfig()
	if cfg.SiteDir != "/site/custom" || cfg.InputPath != "/job/custom.zip" {
		t.Fatalf("consumed environment: input=%q site=%q", cfg.InputPath, cfg.SiteDir)
	}
}
