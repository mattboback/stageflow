package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeProjectConfig(t *testing.T, root string, yaml string) string {
	t.Helper()

	configDir := filepath.Join(root, ".stageflow")
	requireNoErr(t, os.MkdirAll(configDir, 0o750))

	configPath := filepath.Join(configDir, "config.yaml")
	requireNoErr(t, os.WriteFile(configPath, []byte(yaml), 0o640))

	return configPath
}

func TestLoadProjectConfig_Success(t *testing.T) {
	root := t.TempDir()

	configYAML := `version: 1
stageflow:
  api_url: http://localhost:8080
  api_key_env: STAGEFLOW_API_KEY
scan:
  urls:
    - http://localhost:3000
  scanners: axe,lighthouse
  screenshot: true
  allow_private_targets: true
  timeout: 2s
  format: json
  severity: serious
  max_issues: 10
  thresholds:
    critical: 0
    serious: 1
    total: 2
dev:
  start:
    cmd: ["npm", "run", "dev"]
    env:
      FOO: bar
  ready:
    url: http://localhost:3000/health
    timeout: 1s
    interval: 10ms
`

	configPath := writeProjectConfig(t, root, configYAML)

	cfg, gotPath, err := loadProjectConfig(root)
	requireNoErr(t, err)

	requireEqual(t, gotPath, configPath, "config path")

	requireEqual(t, cfg.Version, 1, "cfg.Version")

	requireDeepEqual(t, cfg.Scan.URLs, []string{"http://localhost:3000"}, "cfg.Scan.URLs")

	requireEqual(t, cfg.Scan.Scanners, "axe,lighthouse", "cfg.Scan.Scanners")

	requireBoolPtr(t, cfg.Scan.Screenshot, true, "cfg.Scan.Screenshot")

	requireBoolPtr(t, cfg.Scan.AllowPrivateTargets, true, "cfg.Scan.AllowPrivateTargets")

	requireIntPtr(t, cfg.Scan.Thresholds.Critical, 0, "cfg.Scan.Thresholds.Critical")

	requireIntPtr(t, cfg.Scan.Thresholds.Serious, 1, "cfg.Scan.Thresholds.Serious")

	requireIntPtr(t, cfg.Scan.Thresholds.Total, 2, "cfg.Scan.Thresholds.Total")

	requireDeepEqual(t, cfg.Dev.Start.Cmd, []string{"npm", "run", "dev"}, "cfg.Dev.Start.Cmd")

	requireEqual(t, cfg.Dev.Ready.URL, "http://localhost:3000/health", "cfg.Dev.Ready.URL")
}

func TestLoadProjectConfig_MissingFile(t *testing.T) {
	root := t.TempDir()

	_, _, err := loadProjectConfig(root)
	if err == nil {
		t.Fatalf("loadProjectConfig err = nil, want non-nil")
	}

	if !strings.Contains(err.Error(), "no .stageflow/config.yaml") {
		t.Fatalf("loadProjectConfig err = %q, want to contain %q", err.Error(), "no .stageflow/config.yaml")
	}
}

func TestLoadProjectConfig_UnknownFieldReturnsError(t *testing.T) {
	root := t.TempDir()

	configYAML := `version: 1
scan:
  urls: ["https://example.com"]
dev:
  start:
    cmd: ["true"]
  ready:
    url: "http://localhost:1234"
nope: 1
`

	_ = writeProjectConfig(t, root, configYAML)

	_, _, err := loadProjectConfig(root)
	if err == nil {
		t.Fatalf("loadProjectConfig err = nil, want non-nil")
	}
}

func TestLoadProjectConfig_ValidationFailures(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		errContains string
	}{
		{
			name: "version != 1",
			yaml: `version: 2
scan:
  urls: ["https://example.com"]
dev:
  start:
    cmd: ["true"]
  ready:
    url: "http://localhost:1234"
`,
			errContains: "version must be 1",
		},
		{
			name: "scan.urls empty",
			yaml: `version: 1
scan:
  urls: []
dev:
  start:
    cmd: ["true"]
  ready:
    url: "http://localhost:1234"
`,
			errContains: "scan.urls must contain at least one URL",
		},
		{
			name: "scan.urls contains blank entry",
			yaml: `version: 1
scan:
  urls:
    - ""
    - "https://example.com"
dev:
  start:
    cmd: ["true"]
  ready:
    url: "http://localhost:1234"
`,
			errContains: "scan.urls contains an empty URL",
		},
		{
			name: "dev.start.cmd missing/empty",
			yaml: `version: 1
scan:
  urls: ["https://example.com"]
dev:
  start:
    cmd: []
  ready:
    url: "http://localhost:1234"
`,
			errContains: "dev.start.cmd is required",
		},
		{
			name: "dev.ready.url blank",
			yaml: `version: 1
scan:
  urls: ["https://example.com"]
dev:
  start:
    cmd: ["true"]
  ready:
    url: ""
`,
			errContains: "dev.ready.url is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()

			configDir := filepath.Join(root, ".stageflow")
			if err := os.MkdirAll(configDir, 0o750); err != nil {
				t.Fatalf("mkdir .stageflow: %v", err)
			}

			configPath := filepath.Join(configDir, "config.yaml")
			if err := os.WriteFile(configPath, []byte(tt.yaml), 0o640); err != nil {
				t.Fatalf("write config.yaml: %v", err)
			}

			_, _, err := loadProjectConfig(root)
			if err == nil {
				t.Fatalf("loadProjectConfig err = nil, want non-nil")
			}

			if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("loadProjectConfig err = %q, want to contain %q", err.Error(), tt.errContains)
			}
		})
	}
}

func TestConfigDuration(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want time.Duration
		ok   bool
		err  bool
	}{
		{
			name: "empty",
			raw:  "",
			want: 0,
			ok:   false,
			err:  false,
		},
		{
			name: "valid",
			raw:  "2s",
			want: 2 * time.Second,
			ok:   true,
			err:  false,
		},
		{
			name: "invalid",
			raw:  "nope",
			want: 0,
			ok:   true,
			err:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok, err := configDuration(tt.raw)
			if (err != nil) != tt.err {
				t.Fatalf("configDuration(%q) err = %v, want err=%v", tt.raw, err, tt.err)
			}

			if ok != tt.ok {
				t.Fatalf("configDuration(%q) ok = %v, want %v", tt.raw, ok, tt.ok)
			}

			if err == nil && got != tt.want {
				t.Fatalf("configDuration(%q) = %v, want %v", tt.raw, got, tt.want)
			}
		})
	}
}
