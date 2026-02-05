package scannerregistry

import "testing"

const (
	scannerNameAxe = "Axe"
	defaultImage   = "default:latest"
)

func TestNewRegistry(t *testing.T) {
	reg := NewRegistry("default-image:latest")

	if reg == nil {
		t.Fatal("expected non-nil registry")
	}

	if reg.scanners == nil {
		t.Error("expected scanners map to be initialized")
	}

	if reg.aliasMap == nil {
		t.Error("expected aliasMap to be initialized")
	}

	if reg.categoryMap == nil {
		t.Error("expected categoryMap to be initialized")
	}

	if reg.defaultImage != "default-image:latest" {
		t.Errorf("expected defaultImage 'default-image:latest', got '%s'", reg.defaultImage)
	}
}

func TestRegistry_Register_ValidScanner(t *testing.T) {
	reg := NewRegistry("")
	def := &Definition{
		ID:         moduleAxe,
		Name:       "Axe Scanner",
		Categories: []string{categoryAccessibility},
		Aliases:    []string{moduleAxeAlias},
		Enabled:    true,
	}

	if err := reg.Register(def); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	got, ok := reg.Get(moduleAxe)
	if !ok {
		t.Fatal("expected scanner to be registered")
	}

	if got.Name != "Axe Scanner" {
		t.Errorf("expected Name 'Axe Scanner', got '%s'", got.Name)
	}
}

func TestRegistry_Register_RejectsEmptyID(t *testing.T) {
	reg := NewRegistry("")
	def := &Definition{ID: "", Name: "Scanner"}

	if err := reg.Register(def); err == nil {
		t.Fatal("expected error for empty ID")
	}
}

func TestRegistry_Register_RejectsEmptyName(t *testing.T) {
	reg := NewRegistry("")
	def := &Definition{ID: "test", Name: ""}

	if err := reg.Register(def); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestRegistry_Register_DuplicateAlias(t *testing.T) {
	reg := NewRegistry("")
	def1 := &Definition{ID: "scanner1", Name: "Scanner 1", Aliases: []string{"shared-alias"}}
	def2 := &Definition{ID: "scanner2", Name: "Scanner 2", Aliases: []string{"shared-alias"}}

	registerDefinition(t, reg, def1)

	if err := reg.Register(def2); err == nil {
		t.Fatal("expected error for duplicate alias")
	}
}

func TestRegistry_Register_AllowsAliasUpdate(t *testing.T) {
	reg := NewRegistry("")
	def := &Definition{ID: "scanner1", Name: "Scanner 1", Aliases: []string{"my-alias"}}

	registerDefinition(t, reg, def)

	def.Name = "Scanner 1 Updated"
	if err := reg.Register(def); err != nil {
		t.Fatalf("Second Register() error = %v", err)
	}
}

func TestRegistry_Register_UpdatesExistingScanner(t *testing.T) {
	reg := NewRegistry("")
	def := &Definition{ID: moduleAxe, Name: "Axe v1", Categories: []string{"old-category"}}
	updated := &Definition{ID: moduleAxe, Name: "Axe v2", Categories: []string{"new-category"}}

	registerDefinition(t, reg, def)
	registerDefinition(t, reg, updated)

	got, _ := reg.Get(moduleAxe)
	if got.Name != "Axe v2" {
		t.Errorf("expected Name 'Axe v2', got '%s'", got.Name)
	}

	cats := reg.Categories()
	for _, c := range cats {
		if c == "old-category" {
			t.Fatal("old category should have been removed")
		}
	}
}

func TestRegistry_Unregister(t *testing.T) {
	t.Run("removes existing scanner", func(t *testing.T) {
		reg := NewRegistry("")
		def := &Definition{
			ID:         moduleAxe,
			Name:       scannerNameAxe,
			Categories: []string{categoryAccessibility},
			Aliases:    []string{moduleAxeAlias},
		}

		registerDefinition(t, reg, def)

		ok := reg.Unregister(moduleAxe)
		if !ok {
			t.Error("expected Unregister to return true")
		}

		_, found := reg.Get(moduleAxe)
		if found {
			t.Error("expected scanner to be removed")
		}

		// Alias should also be removed
		_, found = reg.Resolve(moduleAxeAlias)
		if found {
			t.Error("expected alias to be removed")
		}
	})

	t.Run("returns false for non-existent scanner", func(t *testing.T) {
		reg := NewRegistry("")

		ok := reg.Unregister("non-existent")
		if ok {
			t.Error("expected Unregister to return false for non-existent scanner")
		}
	})
}

func TestRegistry_Get(t *testing.T) {
	reg := NewRegistry("")
	def := &Definition{ID: moduleAxe, Name: scannerNameAxe}
	registerDefinition(t, reg, def)

	t.Run("finds registered scanner", func(t *testing.T) {
		got, ok := reg.Get(moduleAxe)
		if !ok {
			t.Fatal("expected to find scanner")
		}

		if got.ID != moduleAxe {
			t.Errorf("expected ID %q, got %q", moduleAxe, got.ID)
		}
	})

	t.Run("returns false for non-existent", func(t *testing.T) {
		_, ok := reg.Get("non-existent")
		if ok {
			t.Error("expected not to find non-existent scanner")
		}
	})
}

func TestRegistry_Resolve(t *testing.T) {
	reg := NewRegistry("")
	def := &Definition{
		ID:      moduleAxe,
		Name:    scannerNameAxe,
		Aliases: []string{moduleAxeAlias, "AXE"},
	}
	registerDefinition(t, reg, def)

	t.Run("resolves by ID", func(t *testing.T) {
		got, ok := reg.Resolve(moduleAxe)
		if !ok || got.ID != moduleAxe {
			t.Error("expected to resolve by ID")
		}
	})

	t.Run("resolves by alias", func(t *testing.T) {
		got, ok := reg.Resolve(moduleAxeAlias)
		if !ok || got.ID != moduleAxe {
			t.Error("expected to resolve by alias")
		}
	})

	t.Run("resolves by alias case-insensitively", func(t *testing.T) {
		got, ok := reg.Resolve("AXE")
		if !ok || got.ID != moduleAxe {
			t.Error("expected to resolve case-insensitively")
		}
	})

	t.Run("returns false for unknown", func(t *testing.T) {
		_, ok := reg.Resolve("unknown")
		if ok {
			t.Error("expected not to resolve unknown")
		}
	})
}

func TestRegistry_List(t *testing.T) {
	reg := NewRegistry("")
	registerDefinition(t, reg, &Definition{ID: "zscanner", Name: "Z Scanner"})
	registerDefinition(t, reg, &Definition{ID: "ascanner", Name: "A Scanner"})
	registerDefinition(t, reg, &Definition{ID: "mscanner", Name: "M Scanner"})

	list := reg.List()

	if len(list) != 3 {
		t.Fatalf("expected 3 scanners, got %d", len(list))
	}

	// Should be sorted by ID
	if list[0].ID != "ascanner" {
		t.Errorf("expected first scanner to be 'ascanner', got '%s'", list[0].ID)
	}

	if list[2].ID != "zscanner" {
		t.Errorf("expected last scanner to be 'zscanner', got '%s'", list[2].ID)
	}
}

func TestRegistry_ListEnabled(t *testing.T) {
	reg := NewRegistry("")
	registerDefinition(t, reg, &Definition{ID: "enabled1", Name: "Enabled 1", Enabled: true})
	registerDefinition(t, reg, &Definition{ID: "disabled1", Name: "Disabled 1", Enabled: false})
	registerDefinition(t, reg, &Definition{ID: "enabled2", Name: "Enabled 2", Enabled: true})

	list := reg.ListEnabled()

	if len(list) != 2 {
		t.Fatalf("expected 2 enabled scanners, got %d", len(list))
	}

	for _, s := range list {
		if !s.Enabled {
			t.Errorf("expected all scanners to be enabled, got disabled: %s", s.ID)
		}
	}
}

func TestRegistry_Categories(t *testing.T) {
	reg := NewRegistry("")
	registerDefinition(t, reg, &Definition{ID: "s1", Name: "S1", Categories: []string{"cat1", "cat2"}})
	registerDefinition(t, reg, &Definition{ID: "s2", Name: "S2", Categories: []string{"cat2", "cat3"}})

	cats := reg.Categories()

	if len(cats) != 3 {
		t.Fatalf("expected 3 categories, got %d", len(cats))
	}

	// Should be sorted
	expected := []string{"cat1", "cat2", "cat3"}
	for i, c := range cats {
		if c != expected[i] {
			t.Errorf("expected category %d to be '%s', got '%s'", i, expected[i], c)
		}
	}
}

func TestRegistry_ListByCategory(t *testing.T) {
	reg := NewRegistry("")
	registerDefinition(
		t,
		reg,
		&Definition{ID: moduleAxe, Name: scannerNameAxe, Categories: []string{categoryAccessibility}, Enabled: true},
	)
	registerDefinition(
		t,
		reg,
		&Definition{
			ID:         moduleLighthouse,
			Name:       "Lighthouse",
			Categories: []string{categoryPerformance, categoryAccessibility},
			Enabled:    true,
		},
	)
	registerDefinition(
		t,
		reg,
		&Definition{ID: moduleDisabled, Name: "Disabled", Categories: []string{categoryAccessibility}, Enabled: false},
	)

	t.Run("returns enabled scanners in category", func(t *testing.T) {
		list := reg.ListByCategory(categoryAccessibility)

		if len(list) != 2 {
			t.Fatalf("expected 2 scanners in accessibility, got %d", len(list))
		}
	})

	t.Run("returns nil for unknown category", func(t *testing.T) {
		list := reg.ListByCategory("unknown")
		if list != nil {
			t.Errorf("expected nil for unknown category, got %v", list)
		}
	})
}

func TestRegistry_GetImage(t *testing.T) {
	reg := NewRegistry(defaultImage)
	registerDefinition(t, reg, &Definition{ID: "custom", Name: "Custom", Image: "custom:v1"})
	registerDefinition(t, reg, &Definition{ID: "noimage", Name: "No Image"})

	t.Run("returns scanner image", func(t *testing.T) {
		img := reg.GetImage("custom")
		if img != "custom:v1" {
			t.Errorf("expected 'custom:v1', got '%s'", img)
		}
	})

	t.Run("returns default image when not set", func(t *testing.T) {
		img := reg.GetImage("noimage")
		if img != defaultImage {
			t.Errorf("expected %q, got %q", defaultImage, img)
		}
	})

	t.Run("returns default for unknown scanner", func(t *testing.T) {
		img := reg.GetImage("unknown")
		if img != defaultImage {
			t.Errorf("expected %q, got %q", defaultImage, img)
		}
	})
}

func TestRegistry_Count(t *testing.T) {
	reg := NewRegistry("")

	if reg.Count() != 0 {
		t.Errorf("expected 0, got %d", reg.Count())
	}

	registerDefinition(t, reg, &Definition{ID: "s1", Name: "S1"})
	registerDefinition(t, reg, &Definition{ID: "s2", Name: "S2"})

	if reg.Count() != 2 {
		t.Errorf("expected 2, got %d", reg.Count())
	}
}

func TestRegistry_CountEnabled(t *testing.T) {
	reg := NewRegistry("")
	registerDefinition(t, reg, &Definition{ID: "enabled1", Name: "E1", Enabled: true})
	registerDefinition(t, reg, &Definition{ID: "enabled2", Name: "E2", Enabled: true})
	registerDefinition(t, reg, &Definition{ID: "disabled1", Name: "D1", Enabled: false})

	if reg.CountEnabled() != 2 {
		t.Errorf("expected 2 enabled, got %d", reg.CountEnabled())
	}
}
