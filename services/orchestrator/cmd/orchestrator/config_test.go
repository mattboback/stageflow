package main

import (
	"strings"
	"testing"

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
		DatabaseURL:                   "postgres://stageflow:stageflow@localhost:5432/stageflow?sslmode=disable",
		ExtractionImage:               "localhost/stageflow/extractor:latest",
		ScannerImage:                  "localhost/stageflow/scanner-runner:latest",
		APIPort:                       "8080",
		NatsHost:                      "nats",
		MinioHost:                     "minio",
		PageLoadTimeout:               15000,
		ScrollTimeout:                 300,
		JobEventsRetentionDays:        30,
		JobEventsPruneIntervalMinutes: 60,
		JobEventsPruneBatchSize:       1000,
	}
}
