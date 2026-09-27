package scannerregistry

import (
	"sort"
	"strings"
	"testing"

	"github.com/mattboback/stageflow/libs/go/scannercatalog"
)

func assertConfigMatchesManifests(t *testing.T, cfg *Config, manifests []scannercatalog.ScannerManifest) {
	t.Helper()

	if cfg == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	if cfg.DefaultImage == "" {
		t.Fatal("DefaultConfig() missing DefaultImage")
	}

	if len(cfg.Scanners) != len(manifests) {
		t.Fatalf("expected %d scanners, got %d", len(manifests), len(cfg.Scanners))
	}

	for _, m := range manifests {
		def, ok := cfg.Scanners[m.Id]
		if !ok || def == nil {
			t.Fatalf("missing scanner definition for %s", m.Id)
		}

		if def.ID != m.Id || def.Name != m.Name || def.Version != m.Version {
			t.Fatalf("definition mismatch for %s: %+v vs %+v", m.Id, def, m)
		}

		if !def.Enabled || !def.BuiltIn {
			t.Fatalf("expected %s enabled+builtIn", m.Id)
		}
	}
}

func assertDefaultModules(t *testing.T, registry *Registry) {
	t.Helper()

	defaultModules, err := registry.ResolveModulesStrict(nil)
	if err != nil {
		t.Fatalf("ResolveModulesStrict(nil) error: %v", err)
	}

	want := make([]string, 0, len(registry.List()))
	for _, def := range registry.List() {
		want = append(want, def.ID)
	}

	sort.Strings(want)

	if strings.Join(defaultModules, ",") != strings.Join(want, ",") {
		t.Fatalf("default modules = %v, want the whole catalog %v", defaultModules, want)
	}
}

func assertCategoryResolution(t *testing.T, registry *Registry) {
	t.Helper()

	cats := registry.Categories()
	sort.Strings(cats)

	for _, category := range cats {
		_ = registry.ListByCategory(category)
	}
}

func TestDefaultConfigMatchesBuiltinCatalog(t *testing.T) {
	manifests, err := scannercatalog.BuiltinManifests()
	if err != nil {
		t.Fatalf("BuiltinManifests() error: %v", err)
	}

	cfg, err := DefaultConfig()
	if err != nil {
		t.Fatalf("DefaultConfig() error: %v", err)
	}

	assertConfigMatchesManifests(t, cfg, manifests)

	registry, err := InitializeRegistry(cfg)
	if err != nil {
		t.Fatalf("InitializeRegistry() error: %v", err)
	}

	assertDefaultModules(t, registry)
	assertCategoryResolution(t, registry)
}
