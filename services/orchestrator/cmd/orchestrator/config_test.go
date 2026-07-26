package main

import (
	"strings"
	"testing"
	"time"

	sharedconfig "github.com/mattboback/stageflow/libs/go/config"
)

func TestConfigValidateJobEventsSettings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		mutate        func(cfg *Config)
		expectedError string
	}{
		{
			name: "retention cannot be negative",
			mutate: func(cfg *Config) {
				cfg.JobEventsRetentionDays = -1
			},
			expectedError: "JOB_EVENTS_RETENTION_DAYS must be >= 0",
		},
		{
			name: "prune interval cannot be negative",
			mutate: func(cfg *Config) {
				cfg.JobEventsPruneIntervalMinutes = -1
			},
			expectedError: "JOB_EVENTS_PRUNE_INTERVAL_MINUTES must be >= 0",
		},
		{
			name: "prune batch size cannot be negative",
			mutate: func(cfg *Config) {
				cfg.JobEventsPruneBatchSize = -1
			},
			expectedError: "JOB_EVENTS_PRUNE_BATCH_SIZE must be >= 0",
		},
		{
			name: "retention requires prune interval",
			mutate: func(cfg *Config) {
				cfg.JobEventsRetentionDays = 30
				cfg.JobEventsPruneIntervalMinutes = 0
			},
			expectedError: "JOB_EVENTS_PRUNE_INTERVAL_MINUTES must be > 0 when retention is enabled",
		},
		{
			name: "retention requires prune batch size",
			mutate: func(cfg *Config) {
				cfg.JobEventsRetentionDays = 30
				cfg.JobEventsPruneBatchSize = 0
			},
			expectedError: "JOB_EVENTS_PRUNE_BATCH_SIZE must be > 0 when retention is enabled",
		},
		{
			name: "admin rate limit rps cannot be negative",
			mutate: func(cfg *Config) {
				cfg.AdminRateLimitRPS = -1
			},
			expectedError: "ORCHESTRATOR_ADMIN_RATE_LIMIT_RPS must be >= 0",
		},
		{
			name: "admin rate limit burst cannot be negative",
			mutate: func(cfg *Config) {
				cfg.AdminRateLimitBurst = -1
			},
			expectedError: "ORCHESTRATOR_ADMIN_RATE_LIMIT_BURST must be >= 0",
		},
		{
			name: "admin rate limit burst requires rps",
			mutate: func(cfg *Config) {
				cfg.AdminRateLimitRPS = 0
				cfg.AdminRateLimitBurst = 10
			},
			expectedError: "ORCHESTRATOR_ADMIN_RATE_LIMIT_RPS must be > 0 when burst is set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := validTestConfig()
			tt.mutate(cfg)

			err := cfg.Validate()
			if err == nil {
				t.Fatalf("expected validation error, got nil")
			}

			if !strings.Contains(err.Error(), tt.expectedError) {
				t.Fatalf("expected error to contain %q, got %q", tt.expectedError, err.Error())
			}
		})
	}
}

func TestConfigValidateAllowsDisabledRetentionWithZeroPruneSettings(t *testing.T) {
	t.Parallel()

	cfg := validTestConfig()
	cfg.JobEventsRetentionDays = 0
	cfg.JobEventsPruneIntervalMinutes = 0
	cfg.JobEventsPruneBatchSize = 0

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected config to validate when retention is disabled, got %v", err)
	}
}

func validTestConfig() *Config {
	return &Config{
		NATS: sharedconfig.NATSConfig{
			URL: "nats://nats:4222",
		},
		MinIO: sharedconfig.MinIOConfig{
			Endpoint:  "minio:9000",
			AccessKey: "minio",
			SecretKey: "password",
		},
		PodmanSocket:                  "/run/podman/podman.sock",
		PodmanRequestTimeout:          120 * time.Second,
		PodmanResponseHeaderTimeout:   60 * time.Second,
		PodmanDialTimeout:             5 * time.Second,
		DatabaseURL:                   "postgres://stageflow:stageflow@localhost:5432/stageflow?sslmode=disable",
		ExtractionImage:               "localhost/stageflow/extractor:latest",
		ScannerImage:                  "localhost/stageflow/scanner-runner:latest",
		APIPort:                       "8080",
		APIToken:                      "test-token",
		NatsHost:                      "nats",
		MinioHost:                     "minio",
		PageLoadTimeout:               15000,
		ScrollTimeout:                 300,
		JobEventsRetentionDays:        30,
		JobEventsPruneIntervalMinutes: 60,
		JobEventsPruneBatchSize:       1000,
	}
}

func TestValidatePodmanConfigRejectsAnUnreachableHeaderTimeout(t *testing.T) {
	// RequestTimeout caps the whole exchange, so a larger header timeout can never
	// be reached. Accepting it would give an operator a knob that silently does
	// nothing -- which is how the 10s default went unnoticed in production.
	cfg := &Config{
		PodmanRequestTimeout:        30 * time.Second,
		PodmanResponseHeaderTimeout: 60 * time.Second,
		PodmanDialTimeout:           5 * time.Second,
	}

	errs := cfg.validatePodmanConfig()
	if len(errs) != 1 {
		t.Fatalf("errors = %v, want exactly one", errs)
	}

	if !strings.Contains(errs[0].Error(), "must be <= PODMAN_REQUEST_TIMEOUT") {
		t.Fatalf("error = %v, want it to name the relationship", errs[0])
	}
}

func TestValidatePodmanConfigRequiresPositiveTimeouts(t *testing.T) {
	errs := (&Config{}).validatePodmanConfig()
	if len(errs) != 3 {
		t.Fatalf("errors = %v, want one per zero-valued timeout", errs)
	}
}

func TestValidatePodmanConfigAcceptsTheDefaults(t *testing.T) {
	cfg := &Config{
		PodmanRequestTimeout:        120 * time.Second,
		PodmanResponseHeaderTimeout: 60 * time.Second,
		PodmanDialTimeout:           5 * time.Second,
	}

	if errs := cfg.validatePodmanConfig(); len(errs) != 0 {
		t.Fatalf("errors = %v, want none", errs)
	}
}
