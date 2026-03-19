package main

import "testing"

func TestApplyDefaults(t *testing.T) {
	suite := Suite{}

	applyDefaults(&suite)

	if suite.TimeoutSec == 0 {
		t.Fatalf("expected default timeout_seconds to be set")
	}

	if suite.StreamRetry == 0 {
		t.Fatalf("expected default stream_retry_seconds to be set")
	}

	if len(suite.Modules) == 0 {
		t.Fatalf("expected default modules to be set")
	}

	if suite.Modules[0] != "axe" {
		t.Fatalf("expected default modules[0] to be axe, got %q", suite.Modules[0])
	}
}
