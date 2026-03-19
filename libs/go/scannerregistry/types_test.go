package scannerregistry

import (
	"testing"
)

func TestDefinition_ToInfo(t *testing.T) {
	def := &Definition{
		ID:          "axe",
		Name:        "Axe Accessibility Scanner",
		Version:     "4.8.0",
		Description: "Accessibility testing tool",
		Categories:  []string{"accessibility", "testing"},
		Aliases:     []string{"axe-core", "aXe"},
		Image:       "scanner-runner:latest",
		Enabled:     true,
		BuiltIn:     true,
		Config: map[string]interface{}{
			"timeout": 30000,
		},
		Capabilities: Capabilities{
			SupportsScreenshots: true,
			SupportsConcurrency: false,
			RequiresBrowser:     true,
			SupportsOffline:     false,
			MaxConcurrency:      1,
		},
	}

	info := def.ToInfo()

	if info.ID != def.ID {
		t.Errorf("expected ID '%s', got '%s'", def.ID, info.ID)
	}

	if info.Name != def.Name {
		t.Errorf("expected Name '%s', got '%s'", def.Name, info.Name)
	}

	if info.Version != def.Version {
		t.Errorf("expected Version '%s', got '%s'", def.Version, info.Version)
	}

	if info.Description != def.Description {
		t.Errorf("expected Description '%s', got '%s'", def.Description, info.Description)
	}

	if len(info.Categories) != len(def.Categories) {
		t.Errorf("expected %d categories, got %d", len(def.Categories), len(info.Categories))
	}

	if info.Enabled != def.Enabled {
		t.Errorf("expected Enabled %v, got %v", def.Enabled, info.Enabled)
	}

	if info.BuiltIn != def.BuiltIn {
		t.Errorf("expected BuiltIn %v, got %v", def.BuiltIn, info.BuiltIn)
	}

	if info.Capabilities.SupportsScreenshots != def.Capabilities.SupportsScreenshots {
		t.Error("expected Capabilities to match")
	}
}

func TestDefinition_ToInfo_EmptyFields(t *testing.T) {
	def := &Definition{
		ID:   "minimal",
		Name: "Minimal Scanner",
	}

	info := def.ToInfo()

	if info.ID != "minimal" {
		t.Errorf("expected ID 'minimal', got '%s'", info.ID)
	}

	if info.Description != "" {
		t.Errorf("expected empty Description, got '%s'", info.Description)
	}

	if len(info.Categories) > 0 {
		t.Errorf("expected nil/empty Categories, got %v", info.Categories)
	}
}
