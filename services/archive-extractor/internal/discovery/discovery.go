// Package discovery finds HTML assets inside extracted sites.
package discovery

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// HTMLPage captures a discovered page for provenance generation.
type HTMLPage struct {
	ID   string // Unique identifier (e.g., "page-001")
	Path string // Relative path from site root (e.g., "/index.html")
	File string // Absolute file path
}

func DiscoverHTML(siteDir string) ([]HTMLPage, error) {
	info, err := os.Stat(siteDir)
	if err != nil {
		return nil, fmt.Errorf("failed to stat siteDir: %w", err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("siteDir is not a directory: %s", siteDir)
	}

	pages := make([]HTMLPage, 0, 32)

	err = filepath.WalkDir(siteDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		// Do not include or traverse symlinks.
		if d.Type()&fs.ModeSymlink != 0 {
			if d.IsDir() {
				return fs.SkipDir
			}

			return nil
		}

		if d.IsDir() {
			return nil
		}

		if !isHTMLFile(path) {
			return nil
		}

		relPath, relErr := filepath.Rel(siteDir, path)
		if relErr != nil {
			return fmt.Errorf("failed to get relative path: %w", relErr)
		}

		// Provenance paths use URL separators, not OS separators.
		urlPath := "/" + filepath.ToSlash(relPath)

		pages = append(pages, HTMLPage{
			Path: urlPath,
			File: path,
		})

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	if len(pages) == 0 {
		return nil, fmt.Errorf("no HTML files found in %s", siteDir)
	}

	// Make IDs stable regardless of OS traversal order.
	sort.Slice(pages, func(i, j int) bool {
		return pages[i].Path < pages[j].Path
	})

	for i := range pages {
		pages[i].ID = fmt.Sprintf("page-%03d", i+1)
	}

	return pages, nil
}

func isHTMLFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))

	return ext == ".html" || ext == ".htm"
}
