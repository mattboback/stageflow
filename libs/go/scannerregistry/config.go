package scannerregistry

import (
	"fmt"

	"github.com/mattboback/stageflow/libs/go/scannercatalog"
)

// Config is the in-memory scanner registry configuration, built from the
// embedded manifests in libs/go/scannercatalog.
type Config struct {
	DefaultImage string
	Scanners     map[string]*Definition
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
