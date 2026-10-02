package config

import "testing"

func TestLoadNATSConfig(t *testing.T) {
	// Clear env vars that might be set by just/.env
	t.Setenv("NATS_URL", "")
	t.Setenv("NATS_MAX_RECONNECTS", "")
	t.Setenv("NATS_RECONNECT_WAIT", "")
	t.Setenv("NATS_CONNECT_TIMEOUT", "")

	// Test defaults
	cfg := LoadNATSConfig()

	if cfg.URL != "nats://localhost:4222" {
		t.Errorf("expected default URL, got %q", cfg.URL)
	}

	if cfg.MaxReconnects != 10 {
		t.Errorf("expected MaxReconnects 10, got %d", cfg.MaxReconnects)
	}

	// Test with env override
	t.Setenv("NATS_URL", "nats://custom:4222")

	cfg = LoadNATSConfig()

	if cfg.URL != "nats://custom:4222" {
		t.Errorf("expected custom URL, got %q", cfg.URL)
	}
}
