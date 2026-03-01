package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	APIURL    string `yaml:"api_url"`
	APIKeyEnv string `yaml:"api_key_env"`
}

type projectScanCfg struct {
	URLs                []string             `yaml:"urls"`
	Scanners            string               `yaml:"scanners"`
	Screenshot          *bool                `yaml:"screenshot"`
	AllowPrivateTargets *bool                `yaml:"allow_private_targets"`
	Timeout             string               `yaml:"timeout"`
	Format              string               `yaml:"format"`
	Severity            string               `yaml:"severity"`
	MaxIssues           *int                 `yaml:"max_issues"`
	Thresholds          projectThresholdsCfg `yaml:"thresholds"`
}

type projectThresholdsCfg struct {
	Critical *int `yaml:"critical"`
	Serious  *int `yaml:"serious"`
	Total    *int `yaml:"total"`
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

		return projectConfig{}, "", fmt.Errorf("no .stageflow/config.yaml found under %s", projectRoot)
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

func validateProjectConfig(cfg projectConfig) error {
	if cfg.Version != 1 {
		return fmt.Errorf("version must be 1 (got %d)", cfg.Version)
	}

	if len(cfg.Scan.URLs) == 0 {
		return errors.New("scan.urls must contain at least one URL")
	}

	for _, raw := range cfg.Scan.URLs {
		if strings.TrimSpace(raw) == "" {
			return errors.New("scan.urls contains an empty URL")
		}
	}

	if len(cfg.Dev.Start.Cmd) == 0 {
		return errors.New("dev.start.cmd is required")
	}

	if strings.TrimSpace(cfg.Dev.Ready.URL) == "" {
		return errors.New("dev.ready.url is required")
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
