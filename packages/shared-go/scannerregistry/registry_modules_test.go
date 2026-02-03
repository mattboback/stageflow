package scannerregistry

import (
	"testing"
)

func TestRegistry_ResolveModules(t *testing.T) {
	reg := NewRegistry("")
	_ = reg.Register(&Definition{ID: "axe", Name: "Axe", Categories: []string{"accessibility"}, Aliases: []string{"axe-core"}, Enabled: true})
	_ = reg.Register(&Definition{ID: "lighthouse", Name: "Lighthouse", Categories: []string{"performance", "accessibility"}, Enabled: true})
	_ = reg.Register(&Definition{ID: "seo", Name: "SEO", Categories: []string{"seo"}, Enabled: false})

	t.Run("empty modules returns default axe", func(t *testing.T) {
		result := reg.ResolveModules([]string{})
		if len(result) != 1 || result[0] != "axe" {
			t.Errorf("expected [axe], got %v", result)
		}
	})

	t.Run("resolves by scanner ID", func(t *testing.T) {
		result := reg.ResolveModules([]string{"lighthouse"})
		if len(result) != 1 || result[0] != "lighthouse" {
			t.Errorf("expected [lighthouse], got %v", result)
		}
	})

	t.Run("resolves by alias", func(t *testing.T) {
		result := reg.ResolveModules([]string{"axe-core"})
		if len(result) != 1 || result[0] != "axe" {
			t.Errorf("expected [axe], got %v", result)
		}
	})

	t.Run("resolves by category", func(t *testing.T) {
		result := reg.ResolveModules([]string{"performance"})
		if len(result) != 1 || result[0] != "lighthouse" {
			t.Errorf("expected [lighthouse], got %v", result)
		}
	})

	t.Run("deduplicates results", func(t *testing.T) {
		result := reg.ResolveModules([]string{"axe", "axe-core", "accessibility"})
		// axe appears via ID, alias, and category - should only appear once
		axeCount := 0
		for _, id := range result {
			if id == "axe" {
				axeCount++
			}
		}
		if axeCount != 1 {
			t.Errorf("expected axe to appear once, got %d times in %v", axeCount, result)
		}
	})

	t.Run("passes through unknown modules", func(t *testing.T) {
		result := reg.ResolveModules([]string{"unknown-scanner"})
		if len(result) != 1 || result[0] != "unknown-scanner" {
			t.Errorf("expected [unknown-scanner], got %v", result)
		}
	})

	t.Run("skips disabled scanners", func(t *testing.T) {
		result := reg.ResolveModules([]string{"seo"})
		// Disabled scanner should not be included
		for _, id := range result {
			if id == "seo" {
				t.Errorf("disabled scanner 'seo' should not be included, got %v", result)
			}
		}
	})

	t.Run("skips empty tokens", func(t *testing.T) {
		result := reg.ResolveModules([]string{"", "  ", "axe"})
		if len(result) != 1 || result[0] != "axe" {
			t.Errorf("expected [axe], got %v", result)
		}
	})

	t.Run("results are sorted", func(t *testing.T) {
		result := reg.ResolveModules([]string{"lighthouse", "axe"})
		if len(result) != 2 {
			t.Fatalf("expected 2 results, got %d", len(result))
		}
		if result[0] != "axe" || result[1] != "lighthouse" {
			t.Errorf("expected sorted [axe, lighthouse], got %v", result)
		}
	})
}

func TestRegistry_ResolveModulesStrict(t *testing.T) {
	reg := NewRegistry("")
	_ = reg.Register(&Definition{ID: "axe", Name: "Axe", Categories: []string{"accessibility"}, Enabled: true})
	_ = reg.Register(&Definition{ID: "disabled", Name: "Disabled", Enabled: false})

	t.Run("empty modules returns default axe", func(t *testing.T) {
		result, err := reg.ResolveModulesStrict([]string{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 || result[0] != "axe" {
			t.Errorf("expected [axe], got %v", result)
		}
	})

	t.Run("resolves valid module", func(t *testing.T) {
		result, err := reg.ResolveModulesStrict([]string{"axe"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 || result[0] != "axe" {
			t.Errorf("expected [axe], got %v", result)
		}
	})

	t.Run("returns error for unknown module", func(t *testing.T) {
		_, err := reg.ResolveModulesStrict([]string{"unknown"})
		if err == nil {
			t.Error("expected error for unknown module")
		}
	})

	t.Run("returns error for disabled scanner", func(t *testing.T) {
		_, err := reg.ResolveModulesStrict([]string{"disabled"})
		if err == nil {
			t.Error("expected error for disabled scanner")
		}
	})

	t.Run("returns error when all tokens resolve to empty", func(t *testing.T) {
		_, err := reg.ResolveModulesStrict([]string{"", "  "})
		if err == nil {
			t.Error("expected error when no scanners selected")
		}
	})
}

func TestRegistry_ResolveModules_NoDefaultAxe(t *testing.T) {
	reg := NewRegistry("")
	// No axe registered

	result := reg.ResolveModules([]string{})
	if result != nil {
		t.Errorf("expected nil when axe not available, got %v", result)
	}
}

func TestRegistry_ResolveModulesStrict_NoDefaultAxe(t *testing.T) {
	reg := NewRegistry("")
	// No axe registered

	_, err := reg.ResolveModulesStrict([]string{})
	if err == nil {
		t.Error("expected error when no scanners selected")
	}
}
