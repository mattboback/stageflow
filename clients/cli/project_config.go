package main

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/mattboback/stageflow/clients/cli/internal/manifesttmpl"
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

const scaffoldDevStartCommandPlaceholder = manifesttmpl.DevStartCommandPlaceholder

func (e missingProjectConfigError) Error() string {
	return fmt.Sprintf("no .stageflow/config.yaml found under %s", e.ProjectRoot)
}

func readProjectConfig(projectRoot string) (projectConfig, string, error) {
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

	return cfg, cfgPath, nil
}

func loadProjectConfig(projectRoot string) (projectConfig, string, error) {
	cfg, cfgPath, err := readProjectConfig(projectRoot)
	if err != nil {
		return projectConfig{}, "", err
	}

	if validationErr := validateProjectConfig(cfg); validationErr != nil {
		return projectConfig{}, "", fmt.Errorf("invalid %s: %w", cfgPath, validationErr)
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
	return manifesttmpl.ConfigYAML(manifesttmpl.ConfigParams{
		APIURL:   apiURL,
		Scanners: defaultScanScanners,
		Suggestion: manifesttmpl.Suggestion{
			Command:       suggestion.Command,
			Cwd:           suggestion.Cwd,
			CommandSource: suggestion.CommandSource,
			URL:           suggestion.URL,
		},
	})
}

func defaultProjectGuideTemplate() string {
	return manifesttmpl.GuideMarkdown()
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
