package scannerregistry

import (
	"sort"
	"testing"

	"github.com/mattboback/stageflow/packages/shared-go/scannercatalog"
)

func TestDefaultConfigMatchesBuiltinCatalog(t *testing.T) {
	manifests, err := scannercatalog.BuiltinManifests()
	if err != nil {
		t.Fatalf("BuiltinManifests() error: %v", err)
	}

	cfg := DefaultConfig()
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

	registry, err := InitializeRegistry(DefaultConfig())
	if err != nil {
		t.Fatalf("InitializeRegistry() error: %v", err)
	}

	defaultModules, err := registry.ResolveModulesStrict(nil)
	if err != nil {
		t.Fatalf("ResolveModulesStrict(nil) error: %v", err)
	}
	if len(defaultModules) != 1 || defaultModules[0] != defaultModuleAxe {
		t.Fatalf("expected default module %q, got %v", defaultModuleAxe, defaultModules)
	}

	// Ensure categories resolve deterministically.
	cats := registry.Categories()
	sort.Strings(cats)
	for _, category := range cats {
		_ = registry.ListByCategory(category)
	}
}
