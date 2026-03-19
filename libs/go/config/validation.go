package config

import (
	"errors"
	"fmt"
)

// RequireNonEmpty standardizes validation errors for required string fields.
func RequireNonEmpty(name, value string) error {
	if value == "" {
		return fmt.Errorf("%s is required", name)
	}

	return nil
}

// RequirePositive standardizes validation errors for positive numeric fields.
func RequirePositive(name string, value int) error {
	if value <= 0 {
		return fmt.Errorf("%s must be positive, got %d", name, value)
	}

	return nil
}

// ValidateAll aggregates non-nil validation errors for a single startup failure.
func ValidateAll(errs ...error) error {
	var combined []error

	for _, err := range errs {
		if err != nil {
			combined = append(combined, err)
		}
	}

	if len(combined) == 0 {
		return nil
	}

	return errors.Join(combined...)
}
