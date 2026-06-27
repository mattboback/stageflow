package scannerregistry

import "time"

// Definition is the registry contract for a scanner, used for selection and scheduling.
type Definition struct {
	ID           string         `json:"id"                     yaml:"id"`
	Name         string         `json:"name"                   yaml:"name"`
	Version      string         `json:"version"                yaml:"version"`
	Description  string         `json:"description,omitempty"  yaml:"description,omitempty"`
	Categories   []string       `json:"categories"             yaml:"categories"`
	Aliases      []string       `json:"aliases,omitempty"      yaml:"aliases,omitempty"`
	Image        string         `json:"image,omitempty"        yaml:"image,omitempty"`
	Enabled      bool           `json:"enabled"                yaml:"enabled"`
	BuiltIn      bool           `json:"builtIn"                yaml:"builtIn"`
	Config       map[string]any `json:"config,omitempty"       yaml:"config,omitempty"`
	Capabilities Capabilities   `json:"capabilities"           yaml:"capabilities"`
	Requirements Requirements   `json:"requirements,omitempty" yaml:"requirements,omitempty"`
}

// Capabilities capture scheduling-relevant features and limits for a scanner.
type Capabilities struct {
	OutputFormats        []string `json:"outputFormats,omitempty"        yaml:"outputFormats,omitempty"`
	SupportsScreenshots  bool     `json:"supportsScreenshots"            yaml:"supportsScreenshots"`
	SupportsConcurrency  bool     `json:"supportsConcurrency"            yaml:"supportsConcurrency"`
	RequiresBrowser      bool     `json:"requiresBrowser"                yaml:"requiresBrowser"`
	SupportsOffline      bool     `json:"supportsOffline"                yaml:"supportsOffline"`
	MaxConcurrency       int      `json:"maxConcurrency,omitempty"       yaml:"maxConcurrency,omitempty"`
	EstimatedTimePerPage int      `json:"estimatedTimePerPage,omitempty" yaml:"estimatedTimePerPage,omitempty"`
}

// Requirements describe runtime constraints the orchestrator must honor when starting a scanner.
type Requirements struct {
	Browser      *BrowserRequirement `json:"browser,omitempty"      yaml:"browser,omitempty"`
	NodeVersion  string              `json:"nodeVersion,omitempty"  yaml:"nodeVersion,omitempty"`
	MaxMemoryMB  int                 `json:"maxMemoryMB,omitempty"  yaml:"maxMemoryMB,omitempty"`
	MaxTimeoutMs int                 `json:"maxTimeoutMs,omitempty" yaml:"maxTimeoutMs,omitempty"`
}

// BrowserRequirement declares the browser profile required by a scanner.
type BrowserRequirement struct {
	Type     string   `json:"type"           yaml:"type"`
	Headless bool     `json:"headless"       yaml:"headless"`
	Args     []string `json:"args,omitempty" yaml:"args,omitempty"`
}

// Stats aggregates historical usage for operator visibility.
type Stats struct {
	TotalScans      int64         `json:"totalScans"`
	SuccessfulScans int64         `json:"successfulScans"`
	FailedScans     int64         `json:"failedScans"`
	AverageDuration time.Duration `json:"averageDuration"`
	LastUsed        *time.Time    `json:"lastUsed,omitempty"`
}

// Info is the public API projection of a scanner Definition.
type Info struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Version      string       `json:"version"`
	Description  string       `json:"description,omitempty"`
	Categories   []string     `json:"categories"`
	Enabled      bool         `json:"enabled"`
	BuiltIn      bool         `json:"builtIn"`
	Capabilities Capabilities `json:"capabilities"`
}

// ToInfo returns the public projection for API responses.
func (d *Definition) ToInfo() Info {
	return Info{
		ID:           d.ID,
		Name:         d.Name,
		Version:      d.Version,
		Description:  d.Description,
		Categories:   append([]string(nil), d.Categories...),
		Enabled:      d.Enabled,
		BuiltIn:      d.BuiltIn,
		Capabilities: d.Capabilities.clone(),
	}
}

func (d *Definition) clone() *Definition {
	if d == nil {
		return nil
	}

	copy := *d
	copy.Categories = append([]string(nil), d.Categories...)
	copy.Aliases = append([]string(nil), d.Aliases...)
	copy.Capabilities = d.Capabilities.clone()
	copy.Requirements = d.Requirements.clone()
	if d.Config != nil {
		copy.Config = make(map[string]any, len(d.Config))
		for key, value := range d.Config {
			copy.Config[key] = cloneConfigValue(value)
		}
	}

	return &copy
}

func (c Capabilities) clone() Capabilities {
	c.OutputFormats = append([]string(nil), c.OutputFormats...)
	return c
}

func (r Requirements) clone() Requirements {
	if r.Browser != nil {
		browser := *r.Browser
		browser.Args = append([]string(nil), r.Browser.Args...)
		r.Browser = &browser
	}

	return r
}

func cloneConfigValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		copy := make(map[string]any, len(typed))
		for key, item := range typed {
			copy[key] = cloneConfigValue(item)
		}
		return copy
	case []any:
		copy := make([]any, len(typed))
		for i, item := range typed {
			copy[i] = cloneConfigValue(item)
		}
		return copy
	case []string:
		return append([]string(nil), typed...)
	default:
		return value
	}
}
