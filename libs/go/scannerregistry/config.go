package scannerregistry

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/mattboback/stageflow/libs/go/scannercatalog"
)

// Config is the in-memory scanner registry configuration (built-ins + overrides).
type Config struct {
	DefaultImage string                 `yaml:"defaultImage"`
	Scanners     map[string]*Definition `yaml:"scanners"`
}

// Overrides describes partial scanner overrides loaded from YAML.
// Built-in scanner metadata comes from the embedded manifests in libs/go/scannercatalog.
type Overrides struct {
	DefaultImage string                    `yaml:"defaultImage"`
	Scanners     map[string]*ScannerConfig `yaml:"scanners"`
}

// ScannerConfig contains per-scanner overrides.
type ScannerConfig struct {
	Enabled      *bool                 `yaml:"enabled,omitempty"`
	Image        string                `yaml:"image,omitempty"`
	Capabilities *CapabilitiesOverride `yaml:"capabilities,omitempty"`
	Requirements *RequirementsOverride `yaml:"requirements,omitempty"`
}

// CapabilitiesOverride allows overriding runtime limits.
type CapabilitiesOverride struct {
	MaxConcurrency       *int `yaml:"maxConcurrency,omitempty"`
	EstimatedTimePerPage *int `yaml:"estimatedTimePerPage,omitempty"`
}

// RequirementsOverride allows overriding runtime resource constraints.
type RequirementsOverride struct {
	MaxMemoryMB  *int `yaml:"maxMemoryMB,omitempty"`
	MaxTimeoutMs *int `yaml:"maxTimeoutMs,omitempty"`
}

// LoadOverrides loads scanner overrides from a YAML file.
func LoadOverrides(configPath string) (*Overrides, error) {
	cleanPath := filepath.Clean(configPath)
	data, err := os.ReadFile(cleanPath) // #nosec G304 -- path comes from explicit user/service config, cleaned before read
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var overrides Overrides
	if unmarshalErr := yaml.Unmarshal(data, &overrides); unmarshalErr != nil {
		return nil, fmt.Errorf("failed to parse config: %w", unmarshalErr)
	}

	return &overrides, nil
}

// LoadOverridesFromDir loads overrides from a directory.
// Looks for scanners.yaml, scanners.yml, or config/scanners.yaml.
func LoadOverridesFromDir(dir string) (*Overrides, error) {
	candidates := []string{
		filepath.Join(dir, "scanners.yaml"),
		filepath.Join(dir, "scanners.yml"),
		filepath.Join(dir, "config", "scanners.yaml"),
		filepath.Join(dir, "config", "scanners.yml"),
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return LoadOverrides(path)
		}
	}

	return nil, fmt.Errorf("no scanner overrides found in %s", dir)
}

func applyCapabilitiesOverride(def *Definition, override *CapabilitiesOverride) {
	if override == nil {
		return
	}

	if override.MaxConcurrency != nil {
		def.Capabilities.MaxConcurrency = *override.MaxConcurrency
	}

	if override.EstimatedTimePerPage != nil {
		def.Capabilities.EstimatedTimePerPage = *override.EstimatedTimePerPage
	}
}

func applyRequirementsOverride(def *Definition, override *RequirementsOverride) {
	if override == nil {
		return
	}

	if override.MaxMemoryMB != nil {
		def.Requirements.MaxMemoryMB = *override.MaxMemoryMB
	}

	if override.MaxTimeoutMs != nil {
		def.Requirements.MaxTimeoutMs = *override.MaxTimeoutMs
	}
}

func applyScannerOverride(def *Definition, override *ScannerConfig) {
	if def == nil || override == nil {
		return
	}

	if override.Enabled != nil {
		def.Enabled = *override.Enabled
	}

	if override.Image != "" {
		def.Image = override.Image
	}

	applyCapabilitiesOverride(def, override.Capabilities)
	applyRequirementsOverride(def, override.Requirements)
}

func boolOrFalse(value *bool) bool {
	if value == nil {
		return false
	}

	return *value
}

func intOrZero(value *int) int {
	if value == nil {
		return 0
	}

	return *value
}

func stringOrEmpty(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

func toStringSlice[T ~string](values []T) []string {
	if len(values) == 0 {
		return nil
	}

	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, string(value))
	}

	return out
}

// ApplyOverrides applies YAML overrides to a registry config in place.
func ApplyOverrides(config *Config, overrides *Overrides) *Config {
	if config == nil || overrides == nil {
		return config
	}

	if overrides.DefaultImage != "" {
		config.DefaultImage = overrides.DefaultImage
	}

	for id, scanner := range overrides.Scanners {
		def, ok := config.Scanners[id]
		if !ok {
			continue
		}

		applyScannerOverride(def, scanner)
	}

	return config
}

// InitializeRegistry creates and populates a registry from config.
func InitializeRegistry(config *Config) (*Registry, error) {
	defaultImage := config.DefaultImage
	if defaultImage == "" {
		defaultImage = "localhost/stageflow/scanner-runner:latest"
	}

	registry := NewRegistry(defaultImage)

	for _, def := range config.Scanners {
		if err := registry.Register(def); err != nil {
			return nil, fmt.Errorf("failed to register scanner %s: %w", def.ID, err)
		}
	}

	return registry, nil
}

// DefaultConfig returns the built-in scanner configuration.
func DefaultConfig() (*Config, error) {
	manifests, err := scannercatalog.BuiltinManifests()
	if err != nil {
		return nil, fmt.Errorf("load builtin manifests: %w", err)
	}

	scanners := make(map[string]*Definition, len(manifests))

	for _, m := range manifests {
		def := &Definition{
			ID:          m.Id,
			Name:        m.Name,
			Version:     m.Version,
			Description: stringOrEmpty(m.Description),
			Categories:  toStringSlice(m.Capabilities.Categories),
			Aliases:     append([]string(nil), m.Aliases...),
			Enabled:     true,
			BuiltIn:     true,
			Capabilities: Capabilities{
				OutputFormats:        toStringSlice(m.Capabilities.OutputFormats),
				SupportsScreenshots:  m.Capabilities.SupportsScreenshots,
				SupportsConcurrency:  m.Capabilities.SupportsConcurrency,
				RequiresBrowser:      m.Capabilities.RequiresBrowser,
				SupportsOffline:      boolOrFalse(m.Capabilities.SupportsOffline),
				MaxConcurrency:       intOrZero(m.Capabilities.MaxConcurrency),
				EstimatedTimePerPage: intOrZero(m.Capabilities.EstimatedTimePerPage),
			},
		}

		if m.Requirements != nil {
			def.Requirements.NodeVersion = stringOrEmpty(m.Requirements.NodeVersion)

			if m.Requirements.Browser != nil {
				def.Requirements.Browser = &BrowserRequirement{
					Type:     string(m.Requirements.Browser.Type),
					Headless: boolOrFalse(m.Requirements.Browser.Headless),
					Args:     append([]string(nil), m.Requirements.Browser.Args...),
				}
			}

			if m.Requirements.Resources != nil {
				def.Requirements.MaxMemoryMB = intOrZero(m.Requirements.Resources.MaxMemoryMB)
				def.Requirements.MaxTimeoutMs = intOrZero(m.Requirements.Resources.MaxTimeoutMs)
			}
		}

		scanners[m.Id] = def
	}

	return &Config{
		DefaultImage: "localhost/stageflow/scanner-runner:latest",
		Scanners:     scanners,
	}, nil
}
