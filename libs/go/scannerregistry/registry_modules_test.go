package scannerregistry

import "testing"

const (
	moduleAxe             = defaultModuleAxe
	moduleAxeAlias        = "axe-core"
	moduleLighthouse      = "lighthouse"
	moduleSEO             = "seo"
	moduleDisabled        = "disabled"
	moduleUnknown         = "unknown-scanner"
	categoryAccessibility = "accessibility"
	categoryPerformance   = "performance"
)

func registerDefinition(t *testing.T, reg *Registry, def *Definition) {
	t.Helper()

	if err := reg.Register(def); err != nil {
		t.Fatalf("Register(%s) error: %v", def.ID, err)
	}
}

func newRegistryWithDefaults(t *testing.T) *Registry {
	t.Helper()

	reg := NewRegistry("")
	registerDefinition(t, reg, &Definition{
		ID:         moduleAxe,
		Name:       "Axe",
		Categories: []string{categoryAccessibility},
		Aliases:    []string{moduleAxeAlias},
		Enabled:    true,
	})
	registerDefinition(t, reg, &Definition{
		ID:         moduleLighthouse,
		Name:       "Lighthouse",
		Categories: []string{categoryPerformance, categoryAccessibility},
		Enabled:    true,
	})
	registerDefinition(t, reg, &Definition{
		ID:         moduleSEO,
		Name:       "SEO",
		Categories: []string{moduleSEO},
		Enabled:    false,
	})

	return reg
}

func newRegistryWithDisabled(t *testing.T) *Registry {
	t.Helper()

	reg := NewRegistry("")
	registerDefinition(t, reg, &Definition{ID: moduleAxe, Name: "Axe", Enabled: true})
	registerDefinition(t, reg, &Definition{ID: moduleDisabled, Name: "Disabled", Enabled: false})

	return reg
}

func TestRegistry_ResolveModules_EmptyDefaultsToAxe(t *testing.T) {
	reg := newRegistryWithDefaults(t)

	result := reg.ResolveModules(nil)
	if len(result) != 1 || result[0] != moduleAxe {
		t.Errorf("expected [%s], got %v", moduleAxe, result)
	}
}

func TestRegistry_ResolveModules_ByID(t *testing.T) {
	reg := newRegistryWithDefaults(t)

	result := reg.ResolveModules([]string{moduleLighthouse})
	if len(result) != 1 || result[0] != moduleLighthouse {
		t.Errorf("expected [%s], got %v", moduleLighthouse, result)
	}
}

func TestRegistry_ResolveModules_ByAlias(t *testing.T) {
	reg := newRegistryWithDefaults(t)

	result := reg.ResolveModules([]string{moduleAxeAlias})
	if len(result) != 1 || result[0] != moduleAxe {
		t.Errorf("expected [%s], got %v", moduleAxe, result)
	}
}

func TestRegistry_ResolveModules_ByCategory(t *testing.T) {
	reg := newRegistryWithDefaults(t)

	result := reg.ResolveModules([]string{categoryPerformance})
	if len(result) != 1 || result[0] != moduleLighthouse {
		t.Errorf("expected [%s], got %v", moduleLighthouse, result)
	}
}

func TestRegistry_ResolveModules_Deduplicates(t *testing.T) {
	reg := newRegistryWithDefaults(t)

	result := reg.ResolveModules([]string{moduleAxe, moduleAxeAlias, categoryAccessibility})
	axeCount := 0

	for _, id := range result {
		if id == moduleAxe {
			axeCount++
		}
	}

	if axeCount != 1 {
		t.Errorf("expected %s to appear once, got %d times in %v", moduleAxe, axeCount, result)
	}
}

func TestRegistry_ResolveModules_PassesThroughUnknown(t *testing.T) {
	reg := newRegistryWithDefaults(t)

	result := reg.ResolveModules([]string{moduleUnknown})
	if len(result) != 1 || result[0] != moduleUnknown {
		t.Errorf("expected [%s], got %v", moduleUnknown, result)
	}
}

func TestRegistry_ResolveModules_SkipsDisabled(t *testing.T) {
	reg := newRegistryWithDefaults(t)

	result := reg.ResolveModules([]string{moduleSEO})

	for _, id := range result {
		if id == moduleSEO {
			t.Fatalf("disabled scanner %q should not be included, got %v", moduleSEO, result)
		}
	}
}

func TestRegistry_ResolveModules_SkipsEmptyTokens(t *testing.T) {
	reg := newRegistryWithDefaults(t)

	result := reg.ResolveModules([]string{"", "  ", moduleAxe})
	if len(result) != 1 || result[0] != moduleAxe {
		t.Errorf("expected [%s], got %v", moduleAxe, result)
	}
}

func TestRegistry_ResolveModules_Sorted(t *testing.T) {
	reg := newRegistryWithDefaults(t)

	result := reg.ResolveModules([]string{moduleLighthouse, moduleAxe})
	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}

	if result[0] != moduleAxe || result[1] != moduleLighthouse {
		t.Errorf("expected sorted [%s, %s], got %v", moduleAxe, moduleLighthouse, result)
	}
}

func TestRegistry_ResolveModulesStrict_EmptyDefaultsToAxe(t *testing.T) {
	reg := newRegistryWithDisabled(t)

	result, err := reg.ResolveModulesStrict(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 || result[0] != moduleAxe {
		t.Errorf("expected [%s], got %v", moduleAxe, result)
	}
}

func TestRegistry_ResolveModulesStrict_ResolvesValid(t *testing.T) {
	reg := newRegistryWithDisabled(t)

	result, err := reg.ResolveModulesStrict([]string{moduleAxe})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 || result[0] != moduleAxe {
		t.Errorf("expected [%s], got %v", moduleAxe, result)
	}
}

func TestRegistry_ResolveModulesStrict_UnknownModule(t *testing.T) {
	reg := newRegistryWithDisabled(t)

	_, err := reg.ResolveModulesStrict([]string{"unknown"})
	if err == nil {
		t.Fatal("expected error for unknown module")
	}
}

func TestRegistry_ResolveModulesStrict_DisabledModule(t *testing.T) {
	reg := newRegistryWithDisabled(t)

	_, err := reg.ResolveModulesStrict([]string{moduleDisabled})
	if err == nil {
		t.Fatal("expected error for disabled scanner")
	}
}

func TestRegistry_ResolveModulesStrict_AllTokensEmpty(t *testing.T) {
	reg := newRegistryWithDisabled(t)

	_, err := reg.ResolveModulesStrict([]string{"", "  "})
	if err == nil {
		t.Fatal("expected error when no scanners selected")
	}
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
