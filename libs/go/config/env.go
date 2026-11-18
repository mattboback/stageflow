// Package config provides utilities for loading configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// GetEnv returns an env var, treating empty values as unset for predictable defaults.
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return defaultValue
}

// GetEnvInt parses an integer env var and falls back on parse failures to keep startup resilient.
func GetEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}

	return defaultValue
}

// GetEnvBool reads a boolean env var using a fixed allowlist to avoid surprising coercions.
// Recognized true values: "1", "true", "TRUE", "True".
// Recognized false values: "0", "false", "FALSE", "False".
func GetEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		switch value {
		case "1", "true", "TRUE", "True":
			return true
		case "0", "false", "FALSE", "False":
			return false
		}
	}

	return defaultValue
}

func GetEnvDurationStrict(key string, defaultValue time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue, nil
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a duration: %w", key, err)
	}

	return duration, nil
}

func MustGetEnvDuration(key string, defaultValue time.Duration) time.Duration {
	value, err := GetEnvDurationStrict(key, defaultValue)
	if err != nil {
		panic(err)
	}

	return value
}
