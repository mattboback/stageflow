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
