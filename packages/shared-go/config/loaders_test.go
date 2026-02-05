package config

import "testing"

func TestLoadNATSConfig(t *testing.T) {
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

func TestLoadMinIOConfig(t *testing.T) {
	// Test defaults
	cfg := LoadMinIOConfig()

	if cfg.Endpoint != "localhost:9000" {
		t.Errorf("expected default Endpoint, got %q", cfg.Endpoint)
	}

	if cfg.AccessKey != "" {
		t.Errorf("expected default AccessKey to be empty, got %q", cfg.AccessKey)
	}

	if cfg.SecretKey != "" {
		t.Errorf("expected default SecretKey to be empty, got %q", cfg.SecretKey)
	}

	if cfg.PublicEndpoint != "example.com" {
		t.Errorf("expected default PublicEndpoint example.com, got %q", cfg.PublicEndpoint)
	}

	if cfg.UseSSL {
		t.Error("expected UseSSL to be false by default")
	}

	if !cfg.PublicUseSSL {
		t.Error("expected PublicUseSSL to be true by default")
	}

	if cfg.Region != "us-east-1" {
		t.Errorf("expected default Region us-east-1, got %q", cfg.Region)
	}

	// Test with env override
	t.Setenv("MINIO_ENDPOINT", "minio:9000")
	t.Setenv("MINIO_USE_SSL", "true")
	t.Setenv("MINIO_REGION", "custom")
	t.Setenv("MINIO_ROOT_USER", "root-user")
	t.Setenv("MINIO_ROOT_PASSWORD", "root-password")
	t.Setenv("MINIO_ACCESS_KEY", "app-user")
	t.Setenv("MINIO_SECRET_KEY", "app-password")
	t.Setenv("STAGEFLOW_PUBLIC_DOMAIN", "stageflow.example")

	cfg = LoadMinIOConfig()

	if cfg.Endpoint != "minio:9000" {
		t.Errorf("expected minio:9000, got %q", cfg.Endpoint)
	}

	if !cfg.UseSSL {
		t.Error("expected UseSSL to be true")
	}

	if cfg.Region != "custom" {
		t.Errorf("expected custom region, got %q", cfg.Region)
	}

	if cfg.AccessKey != "app-user" {
		t.Errorf("expected AccessKey app-user, got %q", cfg.AccessKey)
	}

	if cfg.SecretKey != "app-password" {
		t.Errorf("expected SecretKey app-password, got %q", cfg.SecretKey)
	}

	if cfg.PublicEndpoint != "stageflow.example" {
		t.Errorf("expected PublicEndpoint stageflow.example, got %q", cfg.PublicEndpoint)
	}
}
