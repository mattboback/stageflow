// Package config provides utilities for loading configuration from environment variables.
package config

import (
	"fmt"
	"os"
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
		var i int
		if _, err := fmt.Sscanf(value, "%d", &i); err == nil {
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

// GetEnvDuration parses a duration env var, defaulting on parse errors to keep services configurable but robust.
// Duration strings follow time.ParseDuration (e.g., "5m", "30s", "1h").
func GetEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}

	return defaultValue
}
