package config

import (
	"testing"
	"time"
)

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		want         string
	}{
		{
			name:         "returns environment value when set",
			key:          "TEST_VAR",
			defaultValue: "default",
			envValue:     "custom",
			want:         "custom",
		},
		{
			name:         "returns default when not set",
			key:          "TEST_VAR",
			defaultValue: "default",
			envValue:     "",
			want:         "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.key, tt.envValue)

			if got := GetEnv(tt.key, tt.defaultValue); got != tt.want {
				t.Errorf("GetEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetEnvInt(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue int
		envValue     string
		want         int
	}{
		{
			name:         "returns parsed integer when valid",
			key:          "TEST_INT",
			defaultValue: 10,
			envValue:     "42",
			want:         42,
		},
		{
			name:         "returns default when not set",
			key:          "TEST_INT",
			defaultValue: 10,
			envValue:     "",
			want:         10,
		},
		{
			name:         "returns default when invalid",
			key:          "TEST_INT",
			defaultValue: 10,
			envValue:     "invalid",
			want:         10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.key, tt.envValue)

			if got := GetEnvInt(tt.key, tt.defaultValue); got != tt.want {
				t.Errorf("GetEnvInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue bool
		envValue     string
		want         bool
	}{
		{
			name:         "returns true for '1'",
			key:          "TEST_BOOL",
			defaultValue: false,
			envValue:     "1",
			want:         true,
		},
		{
			name:         "returns true for 'true'",
			key:          "TEST_BOOL",
			defaultValue: false,
			envValue:     "true",
			want:         true,
		},
		{
			name:         "returns true for 'TRUE'",
			key:          "TEST_BOOL",
			defaultValue: false,
			envValue:     "TRUE",
			want:         true,
		},
		{
			name:         "returns true for 'True'",
			key:          "TEST_BOOL",
			defaultValue: false,
			envValue:     "True",
			want:         true,
		},
		{
			name:         "returns false for '0'",
			key:          "TEST_BOOL",
			defaultValue: true,
			envValue:     "0",
			want:         false,
		},
		{
			name:         "returns false for 'false'",
			key:          "TEST_BOOL",
			defaultValue: true,
			envValue:     "false",
			want:         false,
		},
		{
			name:         "returns false for 'FALSE'",
			key:          "TEST_BOOL",
			defaultValue: true,
			envValue:     "FALSE",
			want:         false,
		},
		{
			name:         "returns false for 'False'",
			key:          "TEST_BOOL",
			defaultValue: true,
			envValue:     "False",
			want:         false,
		},
		{
			name:         "returns default when not set",
			key:          "TEST_BOOL",
			defaultValue: true,
			envValue:     "",
			want:         true,
		},
		{
			name:         "returns default when invalid",
			key:          "TEST_BOOL",
			defaultValue: true,
			envValue:     "invalid",
			want:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.key, tt.envValue)

			if got := GetEnvBool(tt.key, tt.defaultValue); got != tt.want {
				t.Errorf("GetEnvBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetEnvDuration(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue time.Duration
		envValue     string
		want         time.Duration
	}{
		{
			name:         "returns parsed duration when valid",
			key:          "TEST_DURATION",
			defaultValue: 5 * time.Minute,
			envValue:     "10m",
			want:         10 * time.Minute,
		},
		{
			name:         "returns default when not set",
			key:          "TEST_DURATION",
			defaultValue: 5 * time.Minute,
			envValue:     "",
			want:         5 * time.Minute,
		},
		{
			name:         "returns default when invalid",
			key:          "TEST_DURATION",
			defaultValue: 5 * time.Minute,
			envValue:     "invalid",
			want:         5 * time.Minute,
		},
		{
			name:         "parses complex duration",
			key:          "TEST_DURATION",
			defaultValue: 1 * time.Second,
			envValue:     "1h30m45s",
			want:         1*time.Hour + 30*time.Minute + 45*time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.key, tt.envValue)

			if got := GetEnvDuration(tt.key, tt.defaultValue); got != tt.want {
				t.Errorf("GetEnvDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}
