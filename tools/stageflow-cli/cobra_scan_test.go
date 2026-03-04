package main

import (
	"errors"
	"strings"
	"testing"
)

func TestEnhanceSubmitJobError_PrivateTargetHint(t *testing.T) {
	baseErr := errors.New(
		"API request failed with status 400: " +
			"{\"error\":\"hostname localhost resolves to disallowed address ::1\"}",
	)

	err := enhanceSubmitJobError(baseErr, SubmitJobRequest{AllowPrivateTargets: false})
	if !strings.Contains(err.Error(), "--allow-private-targets") {
		t.Fatalf("expected private target hint, got %q", err.Error())
	}
}

func TestEnhanceSubmitJobError_LocalStackHint(t *testing.T) {
	baseErr := errors.New(
		"API request failed with status 400: " +
			"{\"error\":{\"field\":\"allow_private_targets\"}}",
	)

	err := enhanceSubmitJobError(baseErr, SubmitJobRequest{AllowPrivateTargets: true})
	if !strings.Contains(err.Error(), "just dev up local") {
		t.Fatalf("expected local stack hint, got %q", err.Error())
	}
}
