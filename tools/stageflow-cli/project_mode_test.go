package main

import (
	"strings"
	"testing"
	"time"
)

func expectedAPIURL(setFlags map[string]bool, flagOptions runOptions, cfg projectConfig) string {
	if setFlags["api"] {
		return flagOptions.apiURL
	}

	return cfg.Stageflow.APIURL
}

func expectedAPIKey(setFlags map[string]bool, flagOptions runOptions, cfg projectConfig, getenv getenvFunc) string {
	if setFlags["api-key"] {
		return flagOptions.apiKey
	}

	return strings.TrimSpace(getenv(strings.TrimSpace(cfg.Stageflow.APIKeyEnv)))
}

func expectedModules(setFlags map[string]bool, flagOptions runOptions, configModules []string) []string {
	if setFlags["scanners"] {
		return flagOptions.modules
	}

	return configModules
}

func expectedScreenshot(setFlags map[string]bool, flagOptions runOptions, cfg projectConfig) bool {
	if setFlags["screenshot"] {
		return flagOptions.screenshot
	}

	return *cfg.Scan.Screenshot
}

func expectedAllowPrivate(setFlags map[string]bool, flagOptions runOptions, cfg projectConfig) bool {
	if setFlags["allow-private-targets"] {
		return flagOptions.allowPrivate
	}

	return *cfg.Scan.AllowPrivateTargets
}

func expectedTimeout(setFlags map[string]bool, flagOptions runOptions, configTimeout time.Duration) time.Duration {
	if setFlags["timeout"] {
		return flagOptions.timeout
	}

	return configTimeout
}

func expectedFormat(setFlags map[string]bool, flagOptions runOptions, configFormat string) string {
	if setFlags["format"] {
		return flagOptions.format
	}

	return configFormat
}

func expectedMaxIssues(setFlags map[string]bool, flagOptions runOptions, configMax int) int {
	if setFlags["max"] {
		return flagOptions.maxIssues
	}

	return configMax
}

func expectedMinSeverity(setFlags map[string]bool, flagOptions runOptions, configMin severityLevel) severityLevel {
	if setFlags["severity"] {
		return flagOptions.minSeverity
	}

	return configMin
}

func expectedThresholdCritical(setFlags map[string]bool, flagOptions runOptions, configVal int) int {
	if setFlags["threshold-critical"] {
		return flagOptions.thresholdCritical
	}

	return configVal
}

func expectedThresholdSerious(setFlags map[string]bool, flagOptions runOptions, configVal int) int {
	if setFlags["threshold-serious"] {
		return flagOptions.thresholdSerious
	}

	return configVal
}

func expectedThresholdTotal(setFlags map[string]bool, flagOptions runOptions, configVal int) int {
	if setFlags["threshold-total"] {
		return flagOptions.thresholdTotal
	}

	return configVal
}

func TestApplyProjectConfig_Precedence(t *testing.T) {
	boolPtr := func(v bool) *bool { return &v }
	intPtr := func(v int) *int { return &v }

	flagOptions := runOptions{
		urls:              []string{"https://flag.example"},
		modules:           []string{"flag-module"},
		screenshot:        true,
		allowPrivate:      true,
		apiURL:            "http://flag-api:8080",
		apiKey:            "flag-key",
		timeout:           123 * time.Second,
		format:            outputFormatSummary,
		outPath:           "flag.out",
		maxIssues:         7,
		minSeverity:       severityModerate,
		thresholdCritical: 10,
		thresholdSerious:  11,
		thresholdTotal:    12,
		noStream:          true,
	}

	cfg := projectConfig{
		Version: 1,
		Stageflow: projectStageflowCfg{
			APIURL:    "http://cfg-api:8080",
			APIKeyEnv: "CFG_API_KEY",
		},
		Scan: projectScanCfg{
			URLs:                []string{"http://cfg-url-1", "http://cfg-url-2"},
			Scanners:            "axe,lighthouse",
			Screenshot:          boolPtr(false),
			AllowPrivateTargets: boolPtr(false),
			Timeout:             "2s",
			Format:              outputFormatJSON,
			Severity:            "critical",
			MaxIssues:           intPtr(42),
			Thresholds: projectThresholdsCfg{
				Critical: intPtr(0),
				Serious:  intPtr(1),
				Total:    intPtr(2),
			},
		},
	}

	getenv := func(key string) string {
		if key == "CFG_API_KEY" {
			return "cfg-key"
		}

		return ""
	}

	configModules, modulesErr := parseModules(cfg.Scan.Scanners)
	requireNoErr(t, modulesErr)

	configSeverity, severityErr := parseMinimumSeverity(cfg.Scan.Severity)
	requireNoErr(t, severityErr)

	configTimeout := 2 * time.Second
	configFormat := outputFormatJSON
	configMax := 42
	configThresholdCritical := 0
	configThresholdSerious := 1
	configThresholdTotal := 2

	tests := []struct {
		name     string
		setFlags map[string]bool
	}{
		{
			name:     "flag not set uses config",
			setFlags: map[string]bool{},
		},
		{
			name:     "api set",
			setFlags: map[string]bool{"api": true},
		},
		{
			name:     "api-key set",
			setFlags: map[string]bool{"api-key": true},
		},
		{
			name:     "scanners set",
			setFlags: map[string]bool{"scanners": true},
		},
		{
			name:     "screenshot set",
			setFlags: map[string]bool{"screenshot": true},
		},
		{
			name:     "allow-private-targets set",
			setFlags: map[string]bool{"allow-private-targets": true},
		},
		{
			name:     "timeout set",
			setFlags: map[string]bool{"timeout": true},
		},
		{
			name:     "format set",
			setFlags: map[string]bool{"format": true},
		},
		{
			name:     "max set",
			setFlags: map[string]bool{"max": true},
		},
		{
			name:     "severity set",
			setFlags: map[string]bool{"severity": true},
		},
		{
			name:     "threshold-critical set",
			setFlags: map[string]bool{"threshold-critical": true},
		},
		{
			name:     "threshold-serious set",
			setFlags: map[string]bool{"threshold-serious": true},
		},
		{
			name:     "threshold-total set",
			setFlags: map[string]bool{"threshold-total": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, applyErr := applyProjectConfig(flagOptions, tt.setFlags, cfg, getenv)
			requireNoErr(t, applyErr)

			requireDeepEqual(t, got.urls, cfg.Scan.URLs, "options.urls")
			requireEqual(t, got.apiURL, expectedAPIURL(tt.setFlags, flagOptions, cfg), "options.apiURL")
			requireEqual(t, got.apiKey, expectedAPIKey(tt.setFlags, flagOptions, cfg, getenv), "options.apiKey")
			requireDeepEqual(
				t,
				got.modules,
				expectedModules(tt.setFlags, flagOptions, configModules),
				"options.modules",
			)
			requireEqual(t, got.screenshot, expectedScreenshot(tt.setFlags, flagOptions, cfg), "options.screenshot")
			requireEqual(
				t,
				got.allowPrivate,
				expectedAllowPrivate(tt.setFlags, flagOptions, cfg),
				"options.allowPrivate",
			)
			requireEqual(t, got.timeout, expectedTimeout(tt.setFlags, flagOptions, configTimeout), "options.timeout")
			requireEqual(t, got.format, expectedFormat(tt.setFlags, flagOptions, configFormat), "options.format")
			requireEqual(t, got.maxIssues, expectedMaxIssues(tt.setFlags, flagOptions, configMax), "options.maxIssues")
			requireEqual(
				t,
				got.minSeverity,
				expectedMinSeverity(tt.setFlags, flagOptions, configSeverity),
				"options.minSeverity",
			)
			requireEqual(
				t,
				got.thresholdCritical,
				expectedThresholdCritical(tt.setFlags, flagOptions, configThresholdCritical),
				"options.thresholdCritical",
			)
			requireEqual(
				t,
				got.thresholdSerious,
				expectedThresholdSerious(tt.setFlags, flagOptions, configThresholdSerious),
				"options.thresholdSerious",
			)
			requireEqual(
				t,
				got.thresholdTotal,
				expectedThresholdTotal(tt.setFlags, flagOptions, configThresholdTotal),
				"options.thresholdTotal",
			)
		})
	}
}

func TestApplyProjectConfig_ErrorCases(t *testing.T) {
	intPtr := func(v int) *int { return &v }

	flagOptions := runOptions{
		apiURL:      "http://flag-api:8080",
		apiKey:      "flag-key",
		format:      outputFormatSummary,
		maxIssues:   1,
		minSeverity: severityMinor,
		modules:     []string{"flag-module"},
		timeout:     10 * time.Second,
	}

	getenv := func(string) string { return "" }

	t.Run("invalid scan.scanners", func(t *testing.T) {
		cfg := projectConfig{
			Stageflow: projectStageflowCfg{APIURL: "http://cfg-api:8080"},
			Scan: projectScanCfg{
				URLs:     []string{"https://example.com"},
				Scanners: "axe,,",
			},
		}

		_, err := applyProjectConfig(flagOptions, map[string]bool{}, cfg, getenv)
		if err == nil {
			t.Fatalf("applyProjectConfig err = nil, want non-nil")
		}

		if !strings.Contains(err.Error(), "invalid scan.scanners") {
			t.Fatalf("applyProjectConfig err = %q, want to contain %q", err.Error(), "invalid scan.scanners")
		}
	})

	t.Run("scan.max_issues must be >= 0", func(t *testing.T) {
		cfg := projectConfig{
			Stageflow: projectStageflowCfg{APIURL: "http://cfg-api:8080"},
			Scan: projectScanCfg{
				URLs:      []string{"https://example.com"},
				MaxIssues: intPtr(-1),
			},
		}

		_, err := applyProjectConfig(flagOptions, map[string]bool{}, cfg, getenv)
		if err == nil {
			t.Fatalf("applyProjectConfig err = nil, want non-nil")
		}

		if !strings.Contains(err.Error(), "scan.max_issues must be >= 0") {
			t.Fatalf("applyProjectConfig err = %q, want to contain %q", err.Error(), "scan.max_issues must be >= 0")
		}
	})

	t.Run("invalid scan.format", func(t *testing.T) {
		cfg := projectConfig{
			Stageflow: projectStageflowCfg{APIURL: "http://cfg-api:8080"},
			Scan: projectScanCfg{
				URLs:   []string{"https://example.com"},
				Format: "nope",
			},
		}

		_, err := applyProjectConfig(flagOptions, map[string]bool{}, cfg, getenv)
		if err == nil {
			t.Fatalf("applyProjectConfig err = nil, want non-nil")
		}

		if !strings.Contains(err.Error(), "invalid format") {
			t.Fatalf("applyProjectConfig err = %q, want to contain %q", err.Error(), "invalid format")
		}
	})
}
