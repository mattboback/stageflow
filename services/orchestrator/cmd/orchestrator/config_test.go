package main

import (
	"strings"
	"testing"
	"time"
)

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

func TestValidateRequiresANamedNetworkForBridgeModeJobPods(t *testing.T) {
	valid := func(mode, network string) *Config {
		return &Config{
			PodmanSocket:                "/run/podman/podman.sock",
			PodmanRequestTimeout:        120 * time.Second,
			PodmanResponseHeaderTimeout: 60 * time.Second,
			PodmanDialTimeout:           5 * time.Second,
			DatabaseURL:                 "postgres://stageflow@postgres/stageflow",
			ExtractionImage:             "localhost/stageflow/extractor:test",
			APIPort:                     "8081",
			PodNetnsMode:                mode,
			PodNetwork:                  network,
		}
	}

	tests := []struct {
		name, mode, network, wantErr string
	}{
		{name: "bridge mode on a named network", mode: "bridge", network: "stageflow_jobs"},
		{name: "host mode without a network", mode: "host"},
		{
			// Podman's default bridge can reach every other bridge by address.
			name: "bridge mode without a network", mode: "bridge",
			wantErr: "POD_NETWORK is required when POD_NETNS_MODE is bridge",
		},
		{name: "an unset mode is bridge mode", wantErr: "POD_NETWORK is required when POD_NETNS_MODE is bridge"},
		{
			name: "host mode with a network", mode: "host", network: "stageflow_jobs",
			wantErr: "POD_NETNS_MODE must be bridge when POD_NETWORK is set",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := valid(test.mode, test.network).Validate()
			if test.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() error = %v, want nil", err)
				}

				return
			}

			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("Validate() error = %v, want it to contain %q", err, test.wantErr)
			}
		})
	}
}
