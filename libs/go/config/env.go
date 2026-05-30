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

func GetEnvIntStrict(key string, defaultValue int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue, nil
	}

	i, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}

	return i, nil
}

func MustGetEnvInt(key string, defaultValue int) int {
	value, err := GetEnvIntStrict(key, defaultValue)
	if err != nil {
		panic(err)
	}

	return value
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

func GetEnvBoolStrict(key string, defaultValue bool) (bool, error) {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue, nil
	}

	switch value {
	case "1", "true", "TRUE", "True":
		return true, nil
	case "0", "false", "FALSE", "False":
		return false, nil
	default:
		return false, fmt.Errorf("%s must be a boolean (true/false or 1/0)", key)
	}
}

func MustGetEnvBool(key string, defaultValue bool) bool {
	value, err := GetEnvBoolStrict(key, defaultValue)
	if err != nil {
		panic(err)
	}

	return value
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
