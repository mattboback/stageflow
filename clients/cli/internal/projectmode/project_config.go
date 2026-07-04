package projectmode

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

type Config struct {
	Version   int             `yaml:"version"`
	Stageflow StageflowConfig `yaml:"stageflow"`
	Scan      ScanConfig      `yaml:"scan"`
	Dev       DevConfig       `yaml:"dev"`
}

type StageflowConfig struct {
	APIURL    string `yaml:"api_url"`
	APIKeyEnv string `yaml:"api_key_env"`
	Project   string `yaml:"project"`
}

type ScanConfig struct {
	URLs                []string `yaml:"urls"`
	Scanners            []string `yaml:"scanners"`
	Screenshot          *bool    `yaml:"screenshot"`
	AllowPrivateTargets *bool    `yaml:"allow_private_targets"`
	Timeout             string   `yaml:"timeout"`
}

type DevConfig struct {
	Up    [][]string     `yaml:"up"`
	Start DevStartConfig `yaml:"start"`
	Ready DevReadyConfig `yaml:"ready"`
	Down  [][]string     `yaml:"down"`
	Stop  DevStopConfig  `yaml:"stop"`
}

type DevStartConfig struct {
	Cmd []string          `yaml:"cmd"`
	Cwd string            `yaml:"cwd"`
	Env map[string]string `yaml:"env"`
}

type DevReadyConfig struct {
	URL      string `yaml:"url"`
	Timeout  string `yaml:"timeout"`
	Interval string `yaml:"interval"`
}

type DevStopConfig struct {
	Signal  string `yaml:"signal"`
	Timeout string `yaml:"timeout"`
}

type MissingConfigError struct {
	ProjectRoot string
}

const ScaffoldDevStartCommandPlaceholder = manifesttmpl.DevStartCommandPlaceholder

func (e MissingConfigError) Error() string {
	return fmt.Sprintf("no .stageflow/config.yaml found under %s", e.ProjectRoot)
}

func ReadConfig(projectRoot string) (Config, string, error) {
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
			return Config{}, "", fmt.Errorf("failed to read .stageflow config: %w", errors.Join(readErrs...))
		}

		return Config{}, "", MissingConfigError{ProjectRoot: projectRoot}
	}

	var cfg Config

	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)

	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, "", fmt.Errorf("failed to parse %s: %w", cfgPath, err)
	}

	return cfg, cfgPath, nil
}

func LoadConfig(projectRoot string) (Config, string, error) {
	cfg, cfgPath, err := ReadConfig(projectRoot)
	if err != nil {
		return Config{}, "", err
	}

	if validationErr := ValidateConfig(cfg); validationErr != nil {
		return Config{}, "", fmt.Errorf("invalid %s: %w", cfgPath, validationErr)
	}

	return cfg, cfgPath, nil
}

func ScaffoldConfig(projectRoot string, apiURL string) (string, error) {
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

	suggestion, err := DetectBootstrapSuggestion(projectRoot)
	if err != nil {
		return "", err
	}

	template := DefaultConfigTemplate(apiURL, suggestion)

	writeErr := os.WriteFile(configPath, []byte(template), 0o600)
	if writeErr != nil {
		return "", fmt.Errorf("write %s: %w", configPath, writeErr)
	}

	return configPath, nil
}

func ScaffoldGuide(projectRoot string) (string, error) {
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

func DefaultConfigTemplate(apiURL string, suggestion BootstrapSuggestion) string {
	return manifesttmpl.ConfigYAML(manifesttmpl.ConfigParams{
		APIURL:   apiURL,
		Scanners: DefaultScanScanners,
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

func ValidateConfig(cfg Config) error {
	if cfg.Version != 2 {
		return fmt.Errorf("version must be 2 (got %d)", cfg.Version)
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

	for _, item := range []struct {
		field string
		raw   string
	}{
		{field: "scan.timeout", raw: cfg.Scan.Timeout},
		{field: "dev.ready.timeout", raw: cfg.Dev.Ready.Timeout},
		{field: "dev.ready.interval", raw: cfg.Dev.Ready.Interval},
		{field: "dev.stop.timeout", raw: cfg.Dev.Stop.Timeout},
	} {
		if err := validatePositiveConfigDuration(item.field, item.raw); err != nil {
			return err
		}
	}

	return validateOptionalHTTPURL("stageflow.api_url", cfg.Stageflow.APIURL)
}

// ValidateScanConfig checks only what `stageflow project scan` needs
// from .stageflow/config.yaml: a remote project slug. The dev-loop fields
// required by ValidateConfig may be absent in a remote-only config.
func ValidateScanConfig(cfg Config) error {
	if cfg.Version != 2 {
		return fmt.Errorf("version must be 2 (got %d)", cfg.Version)
	}

	if strings.TrimSpace(cfg.Stageflow.Project) == "" {
		return errors.New(
			"stageflow.project is not set; pass a slug (`stageflow project scan <slug>`) or set it",
		)
	}

	return validateOptionalHTTPURL("stageflow.api_url", cfg.Stageflow.APIURL)
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

func validatePositiveConfigDuration(fieldName, raw string) error {
	d, ok, err := ConfigDuration(raw)
	if err != nil {
		return fmt.Errorf("%s: %w", fieldName, err)
	}

	if ok && d <= 0 {
		return fmt.Errorf("%s must be > 0", fieldName)
	}

	return nil
}

func ConfigDuration(raw string) (time.Duration, bool, error) {
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

const DefaultScanScanners = "axe,lighthouse,seo,link-checker"

// DefaultScanScannerList is the slice form of DefaultScanScanners, used as the
// --scanner flag default.
func DefaultScanScannerList() []string {
	return strings.Split(DefaultScanScanners, ",")
}

func LoadScanConfig(projectRoot string) (Config, string, error) {
	cfg, cfgPath, err := ReadConfig(projectRoot)
	if err != nil {
		var missingErr MissingConfigError
		if errors.As(err, &missingErr) {
			return Config{}, "", fmt.Errorf(
				"no slug given and no .stageflow/config.yaml found under %s; "+
					"pass a slug (`stageflow project scan <slug>`) or run `stageflow dev init` "+
					"and set stageflow.project",
				projectRoot,
			)
		}

		return Config{}, "", err
	}

	if validationErr := ValidateScanConfig(cfg); validationErr != nil {
		return Config{}, "", fmt.Errorf("invalid %s: %w", cfgPath, validationErr)
	}

	return cfg, cfgPath, nil
}
