package config

import (
	"errors"
	"strings"
	"testing"
)

func TestRequireNonEmpty(t *testing.T) {
	err := RequireNonEmpty("name", "value")
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	err = RequireNonEmpty("name", "")
	if err == nil {
		t.Error("expected error for empty value")
	}

	if !strings.Contains(err.Error(), "name") {
		t.Errorf("error should contain field name: %v", err)
	}
}

func TestRequirePositive(t *testing.T) {
	err := RequirePositive("count", 5)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	err = RequirePositive("count", 0)
	if err == nil {
		t.Error("expected error for zero value")
	}

	err = RequirePositive("count", -1)
	if err == nil {
		t.Error("expected error for negative value")
	}
}

func TestValidateAll(t *testing.T) {
	// All nil
	err := ValidateAll(nil, nil, nil)
	if err != nil {
		t.Errorf("expected nil when all errors are nil, got %v", err)
	}

	// Some errors
	e1 := errors.New("error 1")
	e2 := errors.New("error 2")

	err = ValidateAll(nil, e1, nil, e2)
	if err == nil {
		t.Error("expected combined error")
	}

	if !strings.Contains(err.Error(), "error 1") || !strings.Contains(err.Error(), "error 2") {
		t.Errorf("combined error should contain both messages: %v", err)
	}
}
