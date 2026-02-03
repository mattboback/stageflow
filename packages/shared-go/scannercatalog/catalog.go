package scannercatalog

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	scannermanifest "github.com/mattboback/stageflow/packages/contracts/scanner-manifest"
)

//go:embed manifests/*/*.json
var manifestsFS embed.FS

type ScannerManifest = scannermanifest.ScannerManifest

func validateManifest(m *ScannerManifest) error {
	if strings.TrimSpace(m.Id) == "" {
		return errors.New("manifest missing id")
	}

	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("manifest %q missing name", m.Id)
	}

	if strings.TrimSpace(m.Version) == "" {
		return fmt.Errorf("manifest %q missing version", m.Id)
	}

	if len(m.Capabilities.Categories) == 0 {
		return fmt.Errorf("manifest %q missing capabilities.categories", m.Id)
	}

	if len(m.Capabilities.OutputFormats) == 0 {
		return fmt.Errorf("manifest %q missing capabilities.outputFormats", m.Id)
	}

	if strings.TrimSpace(m.Entry.Module) == "" {
		return fmt.Errorf("manifest %q missing entry.module", m.Id)
	}

	return nil
}

// BuiltinManifests returns the embedded scanner manifests used as the single source of truth
// for built-in scanner metadata across services.
func BuiltinManifests() ([]ScannerManifest, error) {
	paths := make([]string, 0)

	if err := fs.WalkDir(manifestsFS, "manifests", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".json") {
			return nil
		}

		paths = append(paths, path)

		return nil
	}); err != nil {
		return nil, fmt.Errorf("walk manifests: %w", err)
	}

	sort.Strings(paths)

	seen := make(map[string]struct{}, len(paths))
	manifests := make([]ScannerManifest, 0, len(paths))

	for _, p := range paths {
		data, err := fs.ReadFile(manifestsFS, p)
		if err != nil {
			return nil, fmt.Errorf("read embedded manifest %s: %w", p, err)
		}

		if err := scannermanifest.ValidateManifestJSON(data); err != nil {
			return nil, fmt.Errorf("validate embedded manifest %s: %w", p, err)
		}

		var m ScannerManifest
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, fmt.Errorf("unmarshal embedded manifest %s: %w", p, err)
		}

		if err := validateManifest(&m); err != nil {
			return nil, fmt.Errorf("validate embedded manifest %s: %w", p, err)
		}

		if _, ok := seen[m.Id]; ok {
			return nil, fmt.Errorf("duplicate manifest id %q (file %s)", m.Id, p)
		}

		seen[m.Id] = struct{}{}
		manifests = append(manifests, m)
	}

	sort.Slice(manifests, func(i, j int) bool {
		return manifests[i].Id < manifests[j].Id
	})

	return manifests, nil
}
