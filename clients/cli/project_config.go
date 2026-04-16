package main

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type projectConfig struct {
	Version   int                 `yaml:"version"`
	Stageflow projectStageflowCfg `yaml:"stageflow"`
	Scan      projectScanCfg      `yaml:"scan"`
	Dev       projectDevCfg       `yaml:"dev"`
}

type projectStageflowCfg struct {
	APIURL        string `yaml:"api_url"`
	APIKeyEnv     string `yaml:"api_key_env"`
	RemoteProject string `yaml:"remote_project"`
	RemoteAPIURL  string `yaml:"remote_api_url"`
}

type projectScanCfg struct {
	URLs                []string `yaml:"urls"`
	Scanners            string   `yaml:"scanners"`
	Screenshot          *bool    `yaml:"screenshot"`
	AllowPrivateTargets *bool    `yaml:"allow_private_targets"`
	Timeout             string   `yaml:"timeout"`
}

type projectDevCfg struct {
	Up    [][]string         `yaml:"up"`
	Start projectDevStartCfg `yaml:"start"`
	Ready projectDevReadyCfg `yaml:"ready"`
	Down  [][]string         `yaml:"down"`
	Stop  projectDevStopCfg  `yaml:"stop"`
}

type projectDevStartCfg struct {
	Cmd []string          `yaml:"cmd"`
	Cwd string            `yaml:"cwd"`
	Env map[string]string `yaml:"env"`
}

type projectDevReadyCfg struct {
	URL      string `yaml:"url"`
	Timeout  string `yaml:"timeout"`
	Interval string `yaml:"interval"`
}

type projectDevStopCfg struct {
	Signal  string `yaml:"signal"`
	Timeout string `yaml:"timeout"`
}

type missingProjectConfigError struct {
	ProjectRoot string
}

const scaffoldDevStartCommandPlaceholder = "__STAGEFLOW_SET_DEV_START_CMD__"

func (e missingProjectConfigError) Error() string {
	return fmt.Sprintf("no .stageflow/config.yaml found under %s", e.ProjectRoot)
}

func loadProjectConfig(projectRoot string) (projectConfig, string, error) {
	configDir := filepath.Join(projectRoot, ".stageflow")

	candidates := []string{
		filepath.Join(configDir, "config.yaml"),
		filepath.Join(configDir, "config.yml"),
	}

	var (
		raw      []byte
		cfgPath  string
		readErrs []error
	)

	for _, candidate := range candidates {
		data, err := os.ReadFile(candidate)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}

			readErrs = append(readErrs, fmt.Errorf("read %s: %w", candidate, err))

			continue
		}

		raw = data
		cfgPath = candidate

		break
	}

	if cfgPath == "" {
		if len(readErrs) > 0 {
			return projectConfig{}, "", fmt.Errorf("failed to read .stageflow config: %w", errors.Join(readErrs...))
		}

		return projectConfig{}, "", missingProjectConfigError{ProjectRoot: projectRoot}
	}

	var cfg projectConfig

	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)

	if err := decoder.Decode(&cfg); err != nil {
		return projectConfig{}, "", fmt.Errorf("failed to parse %s: %w", cfgPath, err)
	}

	if err := validateProjectConfig(cfg); err != nil {
		return projectConfig{}, "", fmt.Errorf("invalid %s: %w", cfgPath, err)
	}

	return cfg, cfgPath, nil
}

func scaffoldProjectConfig(projectRoot string, apiURL string) (string, error) {
	configDir := filepath.Join(projectRoot, ".stageflow")
	if err := os.MkdirAll(configDir, 0o750); err != nil {
		return "", fmt.Errorf("create %s: %w", configDir, err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	if _, err := os.Stat(configPath); err == nil {
		return configPath, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("stat %s: %w", configPath, err)
	}

	suggestion, err := detectProjectBootstrapSuggestion(projectRoot)
	if err != nil {
		return "", err
	}

	template := defaultProjectConfigTemplate(apiURL, suggestion)

	writeErr := os.WriteFile(configPath, []byte(template), 0o600)
	if writeErr != nil {
		return "", fmt.Errorf("write %s: %w", configPath, writeErr)
	}

	return configPath, nil
}

func scaffoldProjectGuide(projectRoot string) (string, error) {
	configDir := filepath.Join(projectRoot, ".stageflow")
	if err := os.MkdirAll(configDir, 0o750); err != nil {
		return "", fmt.Errorf("create %s: %w", configDir, err)
	}

	guidePath := filepath.Join(configDir, "README.md")
	if _, err := os.Stat(guidePath); err == nil {
		return guidePath, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("stat %s: %w", guidePath, err)
	}

	template := defaultProjectGuideTemplate()
	if err := os.WriteFile(guidePath, []byte(template), 0o600); err != nil {
		return "", fmt.Errorf("write %s: %w", guidePath, err)
	}

	return guidePath, nil
}

func defaultProjectConfigTemplate(apiURL string, suggestion projectBootstrapSuggestion) string {
	baseAPI := strings.TrimSpace(apiURL)
	if baseAPI == "" {
		baseAPI = "http://localhost:8080"
	}

	devURL := strings.TrimSpace(suggestion.URL)
	if devURL == "" {
		devURL = "http://127.0.0.1:3000"
	}

	devCommand := strings.TrimSpace(suggestion.Command)
	if devCommand == "" {
		devCommand = scaffoldDevStartCommandPlaceholder
	}

	parts := strings.Fields(devCommand)

	var quotedParts []string

	for _, part := range parts {
		quotedParts = append(quotedParts, strconv.Quote(part))
	}

	devCommandFormatted := strings.Join(quotedParts, ", ")

	devCwd := strings.TrimSpace(suggestion.Cwd)
	if devCwd == "" {
		devCwd = "."
	}

	commandComment := "Replace this placeholder with your real dev command."
	if source := strings.TrimSpace(suggestion.CommandSource); source != "" {
		commandComment = source
	}

	return strings.TrimSpace(fmt.Sprintf(`
version: 1

stageflow:
  api_url: %s
  # Optional: link this local project loop to a hosted StageFlow project for regression memory.
  # remote_project: your-hosted-project-slug
  # remote_api_url: https://api.stageflow.example

scan:
  # Set this to the page URL your dev server serves.
  urls:
    - %s
  scanners: %s
  allow_private_targets: true

dev:
  start:
    # %s
    cmd: [%s]
    cwd: %s
  ready:
    # Match this to your dev server URL.
    url: %s
`, strconv.Quote(baseAPI), devURL, defaultScanScanners, commandComment, devCommandFormatted, devCwd, devURL)) + "\n"
}

func defaultProjectGuideTemplate() string {
	return strings.TrimSpace(`
# StageFlow project setup

This folder configures `+"`stageflow project`"+` for this repository.

## Quick setup

1. Open `+"`config.yaml`"+` in this folder.
2. Set `+"`dev.start.cmd`"+` to the command that starts your app.
3. Set `+"`dev.ready.url`"+` to the URL that returns HTTP 2xx or 3xx when your app is ready.
4. Set `+"`scan.urls`"+` to the page URLs you want scanned.
5. Optional: set `+"`stageflow.remote_project`"+` (and `+"`stageflow.remote_api_url`"+` if needed) to link hosted regression memory.
6. Run `+"`stageflow project`"+` again.

## Example dev commands

- npm: `+"`cmd: [\"npm\", \"run\", \"dev\"]`"+`
- bun: `+"`cmd: [\"bun\", \"run\", \"dev\"]`"+`
- pnpm: `+"`cmd: [\"pnpm\", \"dev\"]`"+`
- yarn: `+"`cmd: [\"yarn\", \"dev\"]`"+`

## Localhost/private scans

For local targets like `+"`localhost`"+` and `+"`127.0.0.1`"+`:

1. Start the StageFlow local overlay:
   - `+"`just dev up local`"+`
   - `+"`just dev init local`"+`
2. Re-run `+"`stageflow project`"+`.

## Hosted regression memory (optional)

After your local localhost loop passes, you can follow up against a hosted project baseline:

- Set `+"`stageflow.remote_project`"+` to the hosted project slug.
- Set `+"`stageflow.remote_api_url`"+` if the hosted project uses a different API base URL.
- Run `+"`stageflow project doctor --format json`"+` to discover the exact hosted follow-up command agents should run next.

## Troubleshooting

- If you see "ENOENT" for your dev command, verify `+"`dev.start.cmd`"+` and `+"`dev.start.cwd`"+`.
- If readiness times out, verify `+"`dev.ready.url`"+` responds while your app is running.

For full documentation, see the [Project Mode guide](https://github.com/mattboback/stageflow/blob/main/docs/PROJECT_MODE.md).
`) + "\n"
}

func validateProjectConfig(cfg projectConfig) error {
	if cfg.Version != 1 {
		return fmt.Errorf("version must be 1 (got %d)", cfg.Version)
	}

	if len(cfg.Scan.URLs) == 0 {
		return errors.New("scan.urls must contain at least one URL")
	}

	for _, raw := range cfg.Scan.URLs {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			return errors.New("scan.urls contains an empty URL")
		}

		u, err := url.Parse(trimmed)
		if err != nil {
			return fmt.Errorf("scan.urls contains invalid URL %q: %w", trimmed, err)
		}

		if u.Scheme != "http" && u.Scheme != "https" {
			return fmt.Errorf(
				"scan.urls contains unsupported scheme %q in %q (expected http or https)",
				u.Scheme,
				trimmed,
			)
		}

		if u.Host == "" {
			return fmt.Errorf("scan.urls contains invalid URL %q: missing host", trimmed)
		}
	}

	if len(cfg.Dev.Start.Cmd) == 0 {
		return errors.New("dev.start.cmd is required")
	}

	if strings.TrimSpace(cfg.Dev.Ready.URL) == "" {
		return errors.New("dev.ready.url is required")
	}

	if err := validateOptionalHTTPURL("stageflow.remote_api_url", cfg.Stageflow.RemoteAPIURL); err != nil {
		return err
	}

	return nil
}

func validateOptionalHTTPURL(fieldName string, raw string) error {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}

	u, err := url.Parse(trimmed)
	if err != nil {
		return fmt.Errorf("%s is invalid: %w", fieldName, err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%s has unsupported scheme %q (expected http or https)", fieldName, u.Scheme)
	}

	if u.Host == "" {
		return fmt.Errorf("%s is invalid: missing host", fieldName)
	}

	return nil
}

func configDuration(raw string) (time.Duration, bool, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, false, nil
	}

	d, err := time.ParseDuration(trimmed)
	if err != nil {
		return 0, true, fmt.Errorf("invalid duration %q: %w", trimmed, err)
	}

	return d, true, nil
}
