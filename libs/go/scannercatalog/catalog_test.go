package scannercatalog

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

func TestBuiltinScanners(t *testing.T) {
	want := []string{"axe", "lighthouse", "link-checker", "security-headers", "seo"}
	if got := IDs(); !slices.Equal(got, want) {
		t.Fatalf("IDs() = %v, want %v", got, want)
	}

	for _, scanner := range All() {
		if scanner.Name == "" || scanner.Version == "" || len(scanner.Categories) == 0 {
			t.Errorf("%s: incomplete scanner %+v", scanner.ID, scanner)
		}
	}
}

// The manifests are embedded at build time, so they are checked here rather
// than when a service starts.
func TestManifestsMatchSchema(t *testing.T) {
	schema := compileManifestSchema(t)

	paths, err := fs.Glob(manifestsFS, "manifests/*/manifest.json")
	if err != nil || len(paths) == 0 {
		t.Fatalf("no embedded manifests: %v", err)
	}

	for _, path := range paths {
		data, readErr := fs.ReadFile(manifestsFS, path)
		if readErr != nil {
			t.Fatal(readErr)
		}

		if problem := manifestProblem(schema, data); problem != nil {
			t.Errorf("%s: %v", path, problem)
		}
	}
}

func TestManifestProblemCatchesRuleViolations(t *testing.T) {
	schema := compileManifestSchema(t)
	valid := `{"id":"example","name":"Example","version":"1.0.0",` +
		`"capabilities":{"categories":["quality"],"outputFormats":["json"],` +
		`"supportsScreenshots":false,"supportsConcurrency":true,"requiresBrowser":false},` +
		`"entry":{"module":"./index.js"}`

	if problem := manifestProblem(schema, []byte(valid+`}`)); problem != nil {
		t.Fatalf("valid manifest rejected: %v", problem)
	}

	screenshots := strings.Replace(valid, `"supportsScreenshots":false`, `"supportsScreenshots":true`, 1)

	for name, data := range map[string]string{
		"missing fields":            `{"id":"axe"}`,
		"screenshots, no browser":   screenshots + `}`,
		"configSchema not a schema": valid + `,"configSchema":{"type":42}}`,
	} {
		if manifestProblem(schema, []byte(data)) == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func compileManifestSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()

	data, err := os.ReadFile("../../contracts/scanner-manifest/schema/scanner-manifest.schema.json")
	if err != nil {
		t.Fatal(err)
	}

	compiler := jsonschema.NewCompiler()
	if addErr := compiler.AddResource("scanner-manifest.schema.json", bytes.NewReader(data)); addErr != nil {
		t.Fatal(addErr)
	}

	schema, err := compiler.Compile("scanner-manifest.schema.json")
	if err != nil {
		t.Fatal(err)
	}

	return schema
}

// manifestProblem applies the schema, then the two rules it does not express.
func manifestProblem(schema *jsonschema.Schema, data []byte) error {
	var document any
	if err := json.Unmarshal(data, &document); err != nil {
		return err
	}

	if err := schema.Validate(document); err != nil {
		return err
	}

	var manifest struct {
		ConfigSchema json.RawMessage `json:"configSchema"`
		Capabilities struct {
			SupportsScreenshots bool `json:"supportsScreenshots"`
			RequiresBrowser     bool `json:"requiresBrowser"`
		} `json:"capabilities"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return err
	}

	if manifest.Capabilities.SupportsScreenshots && !manifest.Capabilities.RequiresBrowser {
		return errors.New("supportsScreenshots requires requiresBrowser")
	}

	// The runner validates SCANNER_OPTIONS against configSchema, so it must compile.
	if len(manifest.ConfigSchema) == 0 {
		return nil
	}

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("configSchema.json", bytes.NewReader(manifest.ConfigSchema)); err != nil {
		return fmt.Errorf("configSchema: %w", err)
	}

	if _, err := compiler.Compile("configSchema.json"); err != nil {
		return fmt.Errorf("configSchema: %w", err)
	}

	return nil
}

// The orchestrator applies these limits to every scanner container, and its
// TestJobMemoryFitsBudget sums them.
func TestManifestsSetResourceLimits(t *testing.T) {
	for _, scanner := range All() {
		if scanner.MaxMemoryMB <= 0 || scanner.CPUs <= 0 {
			t.Fatalf("%s: manifest must set requirements.resources.maxMemoryMB and cpus", scanner.ID)
		}
	}
}

func TestLookup(t *testing.T) {
	for name, want := range map[string]string{
		"axe":        "axe",
		"A11Y":       "axe",
		"keyboard":   "axe",
		"404":        "link-checker",
		"csp":        "security-headers",
		"Lighthouse": "lighthouse",
	} {
		if scanner, ok := Lookup(name); !ok || scanner.ID != want {
			t.Errorf("Lookup(%q) = %q, %v; want %q", name, scanner.ID, ok, want)
		}
	}

	if _, ok := Lookup("quality"); ok {
		t.Error("a category resolved as a scanner")
	}

	scanner, _ := Lookup("axe")
	scanner.Aliases[0] = "changed"
	scanner.Capabilities.OutputFormats[0] = "changed"

	if again, _ := Lookup("axe"); again.Aliases[0] == "changed" || again.Capabilities.OutputFormats[0] == "changed" {
		t.Fatal("Lookup exposed the catalog's own slices")
	}
}

func TestHasIDMatchesIDsExactly(t *testing.T) {
	if !HasID("security-headers") {
		t.Error("HasID rejected a scanner ID")
	}

	for _, name := range []string{"a11y", "AXE", "quality", ""} {
		if HasID(name) {
			t.Errorf("HasID(%q) = true", name)
		}
	}
}

func TestCategories(t *testing.T) {
	want := []string{"accessibility", "quality", "security", "seo"}
	if got := Categories(); !slices.Equal(got, want) {
		t.Fatalf("Categories() = %v, want %v", got, want)
	}
}

func TestSelect(t *testing.T) {
	for _, tt := range []struct {
		modules []string
		want    string
	}{
		{nil, "axe,lighthouse,link-checker,security-headers,seo"},
		{[]string{}, "axe,lighthouse,link-checker,security-headers,seo"},
		{[]string{"lighthouse"}, "lighthouse"},
		{[]string{" A11Y "}, "axe"},
		{[]string{"quality"}, "lighthouse,link-checker"},
		// An ID or alias wins over the category of the same name.
		{[]string{"seo"}, "seo"},
		{[]string{"accessibility"}, "axe"},
		{[]string{"wcag", "axe", "keyboard", ""}, "axe"},
		{[]string{"seo", "csp", "axe"}, "axe,security-headers,seo"},
	} {
		got, err := Select(tt.modules)
		if err != nil || strings.Join(got, ",") != tt.want {
			t.Errorf("Select(%q) = %v, %v; want %s", tt.modules, got, err, tt.want)
		}
	}
}

func TestSelectRejectsUnknownAndEmptyModules(t *testing.T) {
	_, err := Select([]string{"axe", "Performance"})

	var moduleErr *ModuleError
	if !errors.As(err, &moduleErr) || moduleErr.Module != "performance" ||
		err.Error() != "unknown scanner module 'performance'" {
		t.Fatalf("unknown module error = %v", err)
	}

	_, err = Select([]string{" ", ""})
	if !errors.As(err, &moduleErr) || moduleErr.Module != "" || err.Error() != "no scanners selected" {
		t.Fatalf("empty modules error = %v", err)
	}
}

func TestLoadRejectsAmbiguousNames(t *testing.T) {
	manifest := func(id string, aliases ...string) *fstest.MapFile {
		data, err := json.Marshal(map[string]any{
			"id":      id,
			"name":    id,
			"version": "1.0.0",
			"aliases": aliases,
			"capabilities": map[string]any{
				"categories":          []string{"quality"},
				"outputFormats":       []string{"json"},
				"supportsScreenshots": false,
				"supportsConcurrency": true,
				"requiresBrowser":     false,
			},
			"entry": map[string]string{"module": "./index.js"},
		})
		if err != nil {
			t.Fatal(err)
		}

		return &fstest.MapFile{Data: data}
	}

	if _, err := load(fstest.MapFS{
		"manifests/one/manifest.json": manifest("one", "shared"),
		"manifests/two/manifest.json": manifest("two", "Shared"),
	}); err == nil || !strings.Contains(err.Error(), "names both") {
		t.Errorf("alias shared by two scanners: %v", err)
	}

	if _, err := load(fstest.MapFS{
		"manifests/one/manifest.json":  manifest("one"),
		"manifests/copy/manifest.json": manifest("one"),
	}); err == nil {
		t.Error("loaded two scanners with one ID")
	}

	if _, err := load(fstest.MapFS{}); err == nil {
		t.Error("loaded an empty catalog")
	}
}
