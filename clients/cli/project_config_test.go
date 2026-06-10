package main

import (
	"errors"
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

	configYAML := `version: 2
stageflow:
  api_url: http://localhost:8080
  api_key_env: STAGEFLOW_API_KEY
  project: hosted-demo
scan:
  urls:
    - http://localhost:3000
  scanners: [axe, lighthouse]
  screenshot: true
  allow_private_targets: true
  timeout: 2s
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

	requireEqual(t, cfg.Version, 2, "cfg.Version")

	requireDeepEqual(t, cfg.Scan.URLs, []string{"http://localhost:3000"}, "cfg.Scan.URLs")

	requireDeepEqual(t, cfg.Scan.Scanners, []string{"axe", "lighthouse"}, "cfg.Scan.Scanners")

	requireBoolPtr(t, cfg.Scan.Screenshot, true, "cfg.Scan.Screenshot")

	requireBoolPtr(t, cfg.Scan.AllowPrivateTargets, true, "cfg.Scan.AllowPrivateTargets")

	requireDeepEqual(t, cfg.Dev.Start.Cmd, []string{"npm", "run", "dev"}, "cfg.Dev.Start.Cmd")

	requireEqual(t, cfg.Dev.Ready.URL, "http://localhost:3000/health", "cfg.Dev.Ready.URL")
	requireEqual(t, cfg.Stageflow.Project, "hosted-demo", "cfg.Stageflow.Project")
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

func TestLoadProjectConfig_MissingFileTypedError(t *testing.T) {
	root := t.TempDir()

	_, _, err := loadProjectConfig(root)
	if err == nil {
		t.Fatalf("loadProjectConfig err = nil, want non-nil")
	}

	var missingErr missingProjectConfigError
	if !errors.As(err, &missingErr) {
		t.Fatalf("loadProjectConfig err = %T, want missingProjectConfigError", err)
	}

	requireEqual(t, missingErr.ProjectRoot, root, "missingErr.ProjectRoot")
}

func TestLoadProjectScanConfig_Minimal(t *testing.T) {
	root := t.TempDir()

	configPath := writeProjectConfig(t, root, `version: 2
stageflow:
  project: hosted-demo
  api_url: https://hosted.stageflow.example
`)

	cfg, gotPath, err := loadProjectScanConfig(root)
	requireNoErr(t, err)

	requireEqual(t, gotPath, configPath, "config path")
	requireEqual(t, cfg.Version, 2, "cfg.Version")
	requireEqual(t, cfg.Stageflow.Project, "hosted-demo", "cfg.Stageflow.Project")
	requireEqual(t, cfg.Stageflow.APIURL, "https://hosted.stageflow.example", "cfg.Stageflow.APIURL")
}

func TestLoadProjectScanConfig_RequiresProject(t *testing.T) {
	root := t.TempDir()

	writeProjectConfig(t, root, `version: 2
stageflow:
  api_url: https://hosted.stageflow.example
`)

	_, _, err := loadProjectScanConfig(root)
	if err == nil {
		t.Fatalf("loadProjectScanConfig err = nil, want non-nil")
	}

	if !strings.Contains(err.Error(), "stageflow.project is not set") {
		t.Fatalf("loadProjectScanConfig err = %q, want stageflow.project guidance", err.Error())
	}
}

func TestLoadProjectScanConfig_MissingConfigSuggestsDevInit(t *testing.T) {
	root := t.TempDir()

	_, _, err := loadProjectScanConfig(root)
	if err == nil {
		t.Fatalf("loadProjectScanConfig err = nil, want non-nil")
	}

	if !strings.Contains(err.Error(), "stageflow dev init") {
		t.Fatalf("loadProjectScanConfig err = %q, want dev init hint", err.Error())
	}
}

func TestScaffoldProjectConfig_CreatesConfig(t *testing.T) {
	root := t.TempDir()

	configPath, err := scaffoldProjectConfig(root, "http://localhost:8080")
	requireNoErr(t, err)

	expectedPath := filepath.Join(root, ".stageflow", "config.yaml")
	requireEqual(t, configPath, expectedPath, "config path")

	data, err := os.ReadFile(configPath)
	requireNoErr(t, err)

	text := string(data)
	if !strings.Contains(text, "version: 2") {
		t.Fatalf("scaffolded config missing version field")
	}

	if !strings.Contains(text, "scanners: [axe, lighthouse, seo, link-checker]") {
		t.Fatalf("scaffolded config missing default scanner list")
	}

	if !strings.Contains(text, "allow_private_targets: true") {
		t.Fatalf("scaffolded config missing allow_private_targets default")
	}

	if !strings.Contains(text, scaffoldDevStartCommandPlaceholder) {
		t.Fatalf("scaffolded config missing dev command placeholder")
	}

	if !strings.Contains(text, "# project: your-project-slug") {
		t.Fatalf("scaffolded config missing project guidance")
	}

	cfg, gotPath, err := loadProjectConfig(root)
	requireNoErr(t, err)
	requireEqual(t, gotPath, configPath, "loaded config path")
	requireEqual(t, cfg.Version, 2, "cfg.Version")
}

func TestScaffoldProjectGuide_CreatesReadme(t *testing.T) {
	root := t.TempDir()

	guidePath, err := scaffoldProjectGuide(root)
	requireNoErr(t, err)

	expectedPath := filepath.Join(root, ".stageflow", "README.md")
	requireEqual(t, guidePath, expectedPath, "guide path")

	data, err := os.ReadFile(guidePath)
	requireNoErr(t, err)

	text := string(data)
	if !strings.Contains(text, "StageFlow dev setup") {
		t.Fatalf("guide missing heading")
	}

	if !strings.Contains(text, "dev.start.cmd") {
		t.Fatalf("guide missing dev.start.cmd guidance")
	}

	if !strings.Contains(text, "just dev up local") {
		t.Fatalf("guide missing localhost setup guidance")
	}

	if !strings.Contains(text, "stageflow.project") {
		t.Fatalf("guide missing remote project guidance")
	}

	if !strings.Contains(text, "stageflow dev doctor --format json") {
		t.Fatalf("guide missing doctor json guidance")
	}
}

func TestLoadProjectConfig_UnknownFieldReturnsError(t *testing.T) {
	root := t.TempDir()

	configYAML := `version: 2
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

func TestLoadProjectConfig_RejectsRemovedV1Keys(t *testing.T) {
	root := t.TempDir()

	configYAML := `version: 2
stageflow:
  remote_project: hosted-demo
  remote_api_url: https://hosted.stageflow.example
scan:
  urls: ["https://example.com"]
dev:
  start:
    cmd: ["true"]
  ready:
    url: "http://localhost:1234"
`

	_ = writeProjectConfig(t, root, configYAML)

	_, _, err := loadProjectConfig(root)
	if err == nil {
		t.Fatalf("loadProjectConfig err = nil, want parse error for removed v1 keys")
	}
}

func TestLoadProjectConfig_ValidationFailures(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		errContains string
	}{
		{
			name: "version != 2",
			yaml: `version: 1
scan:
  urls: ["https://example.com"]
dev:
  start:
    cmd: ["true"]
  ready:
    url: "http://localhost:1234"
`,
			errContains: "version must be 2",
		},
		{
			name: "scan.urls empty",
			yaml: `version: 2
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
			yaml: `version: 2
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
			name: "scan.urls unsupported scheme",
			yaml: `version: 2
scan:
  urls: ["ftp://example.com"]
dev:
  start:
    cmd: ["true"]
  ready:
    url: "http://localhost:1234"
`,
			errContains: "unsupported scheme",
		},
		{
			name: "scan.urls missing host",
			yaml: `version: 2
scan:
  urls: ["http:///nope"]
dev:
  start:
    cmd: ["true"]
  ready:
    url: "http://localhost:1234"
`,
			errContains: "missing host",
		},
		{
			name: "dev.start.cmd missing/empty",
			yaml: `version: 2
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
			yaml: `version: 2
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
		{
			name: "stageflow.api_url unsupported scheme",
			yaml: `version: 2
stageflow:
  api_url: ftp://example.com
scan:
  urls: ["https://example.com"]
dev:
  start:
    cmd: ["true"]
  ready:
    url: "http://localhost:1234"
`,
			errContains: "stageflow.api_url has unsupported scheme",
		},
		{
			name: "stageflow.api_url missing host",
			yaml: `version: 2
stageflow:
  api_url: https:///missing-host
scan:
  urls: ["https://example.com"]
dev:
  start:
    cmd: ["true"]
  ready:
    url: "http://localhost:1234"
`,
			errContains: "stageflow.api_url is invalid: missing host",
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

func TestLoadProjectConfigRejectsNonPositiveDurations(t *testing.T) {
	tests := []struct {
		name        string
		configYAML  string
		errContains string
	}{
		{
			name: "ready interval zero",
			configYAML: `version: 2
stageflow:
  api_url: http://localhost:8080
scan:
  urls:
    - http://localhost:3000
dev:
  start:
    cmd: ["npm", "run", "dev"]
  ready:
    url: http://localhost:3000/health
    interval: 0s
`,
			errContains: "dev.ready.interval must be > 0",
		},
		{
			name: "scan timeout negative",
			configYAML: `version: 2
stageflow:
  api_url: http://localhost:8080
scan:
  urls:
    - http://localhost:3000
  timeout: -1s
dev:
  start:
    cmd: ["npm", "run", "dev"]
  ready:
    url: http://localhost:3000/health
`,
			errContains: "scan.timeout must be > 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writeProjectConfig(t, root, tt.configYAML)

			_, _, err := loadProjectConfig(root)
			if err == nil {
				t.Fatalf("loadProjectConfig err = nil, want non-nil")
			}

			if !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("loadProjectConfig err = %q, want %q", err.Error(), tt.errContains)
			}
		})
	}
}
