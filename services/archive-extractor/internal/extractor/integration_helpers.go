//go:build integration

package extractor

import (
	"fmt"
	"os"
)

// ExtractZIPToDir validates and extracts a ZIP already on disk.
//
// This is intended for tests and local tooling; production extraction should use (*Extractor).Extract.
func ExtractZIPToDir(zipPath, destDir string) error {
	info, err := os.Stat(zipPath)
	if err != nil {
		return fmt.Errorf("stat zip: %w", err)
	}

	if info.IsDir() {
		return fmt.Errorf("zipPath is a directory: %s", zipPath)
	}

	if validateErr := validateZIP(zipPath, info.Size()); validateErr != nil {
		return fmt.Errorf("ZIP validation failed: %w", validateErr)
	}

	if extractErr := extractZIP(zipPath, destDir); extractErr != nil {
		return fmt.Errorf("failed to extract ZIP: %w", extractErr)
	}

	return nil
}
