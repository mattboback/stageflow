// Package scannercatalog is the one description of StageFlow's built-in
// scanners. It reads their embedded manifests once, when the program starts,
// and the result never changes. The package tests check every manifest against
// the shared schema.
package scannercatalog

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"

	scannermanifest "github.com/mattboback/stageflow/libs/contracts/scanner-manifest"
)

//go:embed manifests/*/manifest.json
var manifestsFS embed.FS

// Scanner is one built-in scanner as its manifest describes it.
type Scanner struct {
	ID           string
	Name         string
	Version      string
	Description  string
	Categories   []string
	Aliases      []string
	Capabilities Capabilities
	// MaxMemoryMB is the container memory limit for one run of the scanner.
	MaxMemoryMB int
}

// Capabilities is the manifest's capability block, in the shape
// GET /api/v1/scanners returns it.
type Capabilities struct {
	OutputFormats        []string `json:"outputFormats,omitempty"`
	SupportsScreenshots  bool     `json:"supportsScreenshots"`
	SupportsConcurrency  bool     `json:"supportsConcurrency"`
	RequiresBrowser      bool     `json:"requiresBrowser"`
	SupportsOffline      bool     `json:"supportsOffline"`
	MaxConcurrency       int      `json:"maxConcurrency,omitempty"`
	EstimatedTimePerPage int      `json:"estimatedTimePerPage,omitempty"`
}

// ModuleError reports a requested module that names no built-in scanner.
// Module is empty when the request named no module at all.
type ModuleError struct {
	Module  string
	message string
}

func (e *ModuleError) Error() string { return e.message }

type catalog struct {
	scanners   []Scanner           // sorted by ID
	byName     map[string]int      // lowercased ID or alias -> index into scanners
	byCategory map[string][]string // category -> scanner IDs, sorted
}

// builtin is read at program start. The manifests are embedded, so a load
// error is a build defect, and the package tests catch it first.
var builtin = mustLoad(manifestsFS)

func mustLoad(fsys fs.FS) catalog {
	c, err := load(fsys)
	if err != nil {
		panic("scannercatalog: " + err.Error())
	}

	return c
}

func load(fsys fs.FS) (catalog, error) {
	paths, err := fs.Glob(fsys, "manifests/*/manifest.json")
	if err != nil {
		return catalog{}, err
	}

	if len(paths) == 0 {
		return catalog{}, errors.New("no manifests embedded")
	}

	c := catalog{byName: map[string]int{}, byCategory: map[string][]string{}}

	for _, path := range paths {
		data, readErr := fs.ReadFile(fsys, path)
		if readErr != nil {
			return catalog{}, readErr
		}

		var manifest scannermanifest.ScannerManifest
		if decodeErr := json.Unmarshal(data, &manifest); decodeErr != nil {
			return catalog{}, fmt.Errorf("%s: %w", path, decodeErr)
		}

		c.scanners = append(c.scanners, fromManifest(&manifest))
	}

	slices.SortFunc(c.scanners, func(a, b Scanner) int { return strings.Compare(a.ID, b.ID) })

	for i, scanner := range c.scanners {
		for _, name := range append([]string{scanner.ID}, scanner.Aliases...) {
			key := strings.ToLower(name)
			if other, taken := c.byName[key]; taken {
				return catalog{}, fmt.Errorf("%q names both %s and %s", name, c.scanners[other].ID, scanner.ID)
			}

			c.byName[key] = i
		}

		for _, category := range scanner.Categories {
			c.byCategory[category] = append(c.byCategory[category], scanner.ID)
		}
	}

	return c, nil
}

func fromManifest(m *scannermanifest.ScannerManifest) Scanner {
	scanner := Scanner{
		ID:          m.Id,
		Name:        m.Name,
		Version:     m.Version,
		Description: value(m.Description),
		Categories:  toStrings(m.Capabilities.Categories),
		Aliases:     slices.Clone(m.Aliases),
		Capabilities: Capabilities{
			OutputFormats:        toStrings(m.Capabilities.OutputFormats),
			SupportsScreenshots:  m.Capabilities.SupportsScreenshots,
			SupportsConcurrency:  m.Capabilities.SupportsConcurrency,
			RequiresBrowser:      m.Capabilities.RequiresBrowser,
			SupportsOffline:      value(m.Capabilities.SupportsOffline),
			MaxConcurrency:       value(m.Capabilities.MaxConcurrency),
			EstimatedTimePerPage: value(m.Capabilities.EstimatedTimePerPage),
		},
	}

	if m.Requirements != nil && m.Requirements.Resources != nil {
		scanner.MaxMemoryMB = value(m.Requirements.Resources.MaxMemoryMB)
	}

	return scanner
}

func value[T any](pointer *T) T {
	var zero T
	if pointer == nil {
		return zero
	}

	return *pointer
}

// toStrings converts an enum slice, keeping an empty one nil.
func toStrings[T ~string](values []T) []string {
	if len(values) == 0 {
		return nil
	}

	out := make([]string, len(values))
	for i, v := range values {
		out[i] = string(v)
	}

	return out
}

func (s Scanner) clone() Scanner {
	s.Categories = slices.Clone(s.Categories)
	s.Aliases = slices.Clone(s.Aliases)
	s.Capabilities.OutputFormats = slices.Clone(s.Capabilities.OutputFormats)

	return s
}

// All returns every built-in scanner, sorted by ID.
func All() []Scanner {
	scanners := make([]Scanner, len(builtin.scanners))
	for i := range builtin.scanners {
		scanners[i] = builtin.scanners[i].clone()
	}

	return scanners
}

// IDs returns every built-in scanner ID, sorted.
func IDs() []string {
	ids := make([]string, len(builtin.scanners))
	for i := range builtin.scanners {
		ids[i] = builtin.scanners[i].ID
	}

	return ids
}

// HasID reports whether id is exactly the ID of a built-in scanner.
func HasID(id string) bool {
	i, ok := builtin.byName[id]

	return ok && builtin.scanners[i].ID == id
}

// Lookup returns the scanner whose ID or alias is name, ignoring case.
func Lookup(name string) (Scanner, bool) {
	i, ok := builtin.byName[strings.ToLower(name)]
	if !ok {
		return Scanner{}, false
	}

	return builtin.scanners[i].clone(), true
}

// Categories returns every category a built-in scanner declares, sorted.
func Categories() []string {
	categories := make([]string, 0, len(builtin.byCategory))
	for category := range builtin.byCategory {
		categories = append(categories, category)
	}

	slices.Sort(categories)

	return categories
}

// Select resolves requested modules to scanner IDs, sorted and without
// duplicates. A module is a scanner ID, an alias or a category, ignoring case
// and surrounding space. An ID or alias wins over a category of the same name,
// so "seo" selects only the SEO scanner. No modules selects every scanner.
func Select(modules []string) ([]string, error) {
	if len(modules) == 0 {
		return IDs(), nil
	}

	var selected []string

	for _, module := range modules {
		token := strings.ToLower(strings.TrimSpace(module))
		if token == "" {
			continue
		}

		if i, ok := builtin.byName[token]; ok {
			selected = append(selected, builtin.scanners[i].ID)

			continue
		}

		ids, ok := builtin.byCategory[token]
		if !ok {
			return nil, &ModuleError{Module: token, message: fmt.Sprintf("unknown scanner module '%s'", token)}
		}

		selected = append(selected, ids...)
	}

	if len(selected) == 0 {
		return nil, &ModuleError{message: "no scanners selected"}
	}

	slices.Sort(selected)

	return slices.Compact(selected), nil
}
