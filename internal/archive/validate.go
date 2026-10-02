// Package archive validates static site archives before admission and extraction.
package archive

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
)

const (
	MaxEntries               = 5000
	MaxExpansionRatio        = 100
	MaxCompressedSize        = 100 * 1024 * 1024
	MaxUncompressedSize      = 512 * 1024 * 1024
	MaxEntryUncompressedSize = 250 * 1024 * 1024
	MaxNameLen               = 4096
)

// Validate checks archive metadata for excessive expansion, path traversal,
// duplicate entries, and file/directory collisions. Extraction separately
// bounds actual bytes, protecting against falsified central directory sizes.
func Validate(reader io.ReaderAt, fileSize int64) error {
	if fileSize < 0 || fileSize > MaxCompressedSize {
		return errors.New("ZIP compressed size exceeds limit")
	}

	r, err := zip.NewReader(reader, fileSize)
	if err != nil {
		return fmt.Errorf("not a valid ZIP file: %w", err)
	}

	if len(r.File) > MaxEntries {
		return fmt.Errorf("ZIP has too many files (%d > %d)", len(r.File), MaxEntries)
	}

	var totalUncompressed uint64

	seenFiles := make(map[string]struct{}, len(r.File))
	seenDirs := make(map[string]struct{}, len(r.File))

	for _, file := range r.File {
		if file.Name == "" || len(file.Name) > MaxNameLen {
			return fmt.Errorf("invalid entry name length (%d)", len(file.Name))
		}

		if strings.ContainsRune(file.Name, '\x00') {
			return fmt.Errorf("invalid entry name (NUL byte): %q", file.Name)
		}

		cleanName, sanitizeErr := SanitizeEntryName(file.Name)
		if sanitizeErr != nil {
			return sanitizeErr
		}

		if collisionErr := trackZipEntryName(cleanName, zipEntryIsDir(file), seenFiles, seenDirs); collisionErr != nil {
			return collisionErr
		}

		if file.UncompressedSize64 > MaxEntryUncompressedSize {
			return fmt.Errorf(
				"ZIP entry too large (%s: %d bytes > %d)",
				file.Name,
				file.UncompressedSize64,
				MaxEntryUncompressedSize,
			)
		}

		totalUncompressed += file.UncompressedSize64
	}

	if fileSize > 0 {
		ratio := float64(totalUncompressed) / float64(fileSize)
		if ratio > MaxExpansionRatio {
			return fmt.Errorf("ZIP bomb detected: compression ratio %.1fx exceeds limit", ratio)
		}
	}

	if totalUncompressed > MaxUncompressedSize {
		return fmt.Errorf("ZIP too large: %d bytes uncompressed exceeds 512 MiB limit", totalUncompressed)
	}

	return nil
}

func trackZipEntryName(
	cleanName string,
	isDir bool,
	seenFiles map[string]struct{},
	seenDirs map[string]struct{},
) error {
	if _, ok := seenFiles[cleanName]; ok {
		return fmt.Errorf("duplicate ZIP entry path after normalization: %s", cleanName)
	}

	if _, ok := seenDirs[cleanName]; ok {
		return fmt.Errorf("duplicate ZIP entry path after normalization: %s", cleanName)
	}

	if isDir {
		seenDirs[cleanName] = struct{}{}

		return nil
	}

	for parent := path.Dir(cleanName); parent != "." && parent != "/"; parent = path.Dir(parent) {
		if _, ok := seenFiles[parent]; ok {
			return fmt.Errorf("ZIP entry collides with file path: %s", cleanName)
		}
	}

	seenFiles[cleanName] = struct{}{}

	return nil
}

// SanitizeEntryName normalizes a ZIP path and rejects ambiguous or escaping names.
func SanitizeEntryName(name string) (string, error) {
	// ZIP paths use forward slashes, but malicious archives may include backslashes.
	n := strings.ReplaceAll(name, "\\", "/")

	if strings.HasPrefix(n, "/") {
		return "", fmt.Errorf("absolute path not allowed: %q", name)
	}

	for _, segment := range strings.Split(n, "/") {
		if segment == ".." {
			return "", fmt.Errorf("path traversal detected: %q", name)
		}

		if segment == "." {
			return "", fmt.Errorf("ambiguous path segment not allowed: %q", name)
		}
	}

	// Disallow Windows drive letters (C:/...).
	if len(n) >= 2 {
		c0 := n[0]
		if n[1] == ':' && ((c0 >= 'A' && c0 <= 'Z') || (c0 >= 'a' && c0 <= 'z')) {
			return "", fmt.Errorf("volume name not allowed: %q", name)
		}
	}

	clean := path.Clean(n)
	if clean == "." {
		return "", fmt.Errorf("invalid entry name: %q", name)
	}

	if clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("path traversal detected: %q", name)
	}

	return clean, nil
}

func zipEntryIsDir(file *zip.File) bool {
	return file.FileInfo().IsDir() || strings.HasSuffix(strings.ReplaceAll(file.Name, "\\", "/"), "/")
}
