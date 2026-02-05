// Package extractor downloads and validates customer uploads.
package extractor

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/mattboback/stageflow/packages/shared-go/storage"
)

// Extractor downloads job ZIPs from staging and extracts them safely.
type Extractor struct {
	storageClient *storage.MinIOClient
}

// NewExtractor wires a shared MinIO client into an Extractor.
func NewExtractor(storageClient *storage.MinIOClient) *Extractor {
	return &Extractor{
		storageClient: storageClient,
	}
}

// Extract downloads, validates, and safely unpacks a ZIP into destDir.
func (e *Extractor) Extract(ctx context.Context, bucket, objectPath, destDir string) error {
	tmpFile, err := os.CreateTemp("", "scanner-*.zip")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	defer func() {
		if cerr := os.Remove(tmpFile.Name()); cerr != nil {
			log.Printf("failed to remove temp file %s: %v", tmpFile.Name(), cerr)
		}
	}()
	defer func() {
		if cerr := tmpFile.Close(); cerr != nil {
			log.Printf("failed to close temp file: %v", cerr)
		}
	}()

	obj, err := e.storageClient.DownloadFile(ctx, bucket, objectPath)
	if err != nil {
		return fmt.Errorf("failed to get object from MinIO: %w", err)
	}

	defer func() {
		if cerr := obj.Close(); cerr != nil {
			log.Printf("failed to close MinIO object: %v", cerr)
		}
	}()

	size, err := io.Copy(tmpFile, obj)
	if err != nil {
		return fmt.Errorf("failed to download ZIP: %w", err)
	}

	if validateErr := validateZIP(tmpFile.Name(), size); validateErr != nil {
		return fmt.Errorf("ZIP validation failed: %w", validateErr)
	}

	if extractErr := extractZIP(tmpFile.Name(), destDir); extractErr != nil {
		return fmt.Errorf("failed to extract ZIP: %w", extractErr)
	}

	return nil
}

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

// validateZIP checks for ZIP bombs and path traversal.
func validateZIP(zipPath string, fileSize int64) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("not a valid ZIP file: %w", err)
	}

	defer func() {
		if cerr := r.Close(); cerr != nil {
			log.Printf("failed to close validation reader: %v", cerr)
		}
	}()

	const (
		maxEntries               = 5000                   // protect against tiny-file floods
		maxExpansionRatio        = 100                    // 100x expansion max
		maxUncompressedSize      = 1 * 1024 * 1024 * 1024 // 1 GiB
		maxEntryUncompressedSize = 250 * 1024 * 1024      // 250 MiB
		maxNameLen               = 4096
	)

	if len(r.File) > maxEntries {
		return fmt.Errorf("ZIP has too many files (%d > %d)", len(r.File), maxEntries)
	}

	var totalUncompressed uint64

	for _, file := range r.File {
		if file.Name == "" || len(file.Name) > maxNameLen {
			return fmt.Errorf("invalid entry name length (%d)", len(file.Name))
		}

		if strings.ContainsRune(file.Name, '\x00') {
			return fmt.Errorf("invalid entry name (NUL byte): %q", file.Name)
		}

		if _, sanitizeErr := sanitizeZipEntryName(file.Name); sanitizeErr != nil {
			return sanitizeErr
		}

		if file.UncompressedSize64 > maxEntryUncompressedSize {
			return fmt.Errorf(
				"ZIP entry too large (%s: %d bytes > %d)",
				file.Name,
				file.UncompressedSize64,
				maxEntryUncompressedSize,
			)
		}

		totalUncompressed += file.UncompressedSize64
	}

	if fileSize > 0 {
		ratio := float64(totalUncompressed) / float64(fileSize)
		if ratio > maxExpansionRatio {
			return fmt.Errorf("ZIP bomb detected: compression ratio %.1fx exceeds limit", ratio)
		}
	}

	if totalUncompressed > maxUncompressedSize {
		return fmt.Errorf("ZIP too large: %d bytes uncompressed exceeds 1GB limit", totalUncompressed)
	}

	return nil
}

func sanitizeZipEntryName(name string) (string, error) {
	// ZIP paths use forward slashes, but malicious archives may include backslashes.
	n := strings.ReplaceAll(name, "\\", "/")

	if strings.HasPrefix(n, "/") {
		return "", fmt.Errorf("absolute path not allowed: %q", name)
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

func extractZIP(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}

	defer func() {
		if cerr := r.Close(); cerr != nil {
			log.Printf("failed to close ZIP reader: %v", cerr)
		}
	}()

	if mkdirErr := os.MkdirAll(destDir, 0o750); mkdirErr != nil {
		return fmt.Errorf("failed to create destination directory: %w", mkdirErr)
	}

	for _, file := range r.File {
		cleanName, sanitizeErr := sanitizeZipEntryName(file.Name)
		if sanitizeErr != nil {
			return sanitizeErr
		}

		// #nosec G305 -- file path is validated below before use.
		fpath := filepath.Join(destDir, filepath.FromSlash(cleanName))
		if !isWithinBaseDir(destDir, fpath) {
			return fmt.Errorf("illegal file path: %s", file.Name)
		}

		isDir := file.FileInfo().IsDir() || strings.HasSuffix(strings.ReplaceAll(file.Name, "\\", "/"), "/")
		if isDir {
			if mkErr := os.MkdirAll(fpath, 0o750); mkErr != nil {
				return fmt.Errorf("failed to create directory %s: %w", fpath, mkErr)
			}

			continue
		}

		if parentErr := os.MkdirAll(filepath.Dir(fpath), 0o750); parentErr != nil {
			return fmt.Errorf("failed to create parent directory for %s: %w", fpath, parentErr)
		}

		if extractErr := extractFile(file, fpath); extractErr != nil {
			return fmt.Errorf("failed to extract %s: %w", file.Name, extractErr)
		}
	}

	return nil
}

func isWithinBaseDir(baseDir, targetPath string) bool {
	baseDir = filepath.Clean(baseDir)
	targetPath = filepath.Clean(targetPath)

	rel, err := filepath.Rel(baseDir, targetPath)
	if err != nil {
		return false
	}

	if rel == ".." {
		return false
	}

	if strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return false
	}

	return true
}

func extractFile(file *zip.File, destPath string) error {
	rc, err := file.Open()
	if err != nil {
		return err
	}

	defer func() {
		if cerr := rc.Close(); cerr != nil {
			log.Printf("failed to close zip entry reader: %v", cerr)
		}
	}()

	// Use safe defaults; don't trust ZIP permission bits.
	perm := os.FileMode(0o600)

	outFile, err := os.OpenFile(
		destPath,
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		perm,
	) // #nosec G304 -- destPath validated earlier
	if err != nil {
		return err
	}

	defer func() {
		if cerr := outFile.Close(); cerr != nil {
			log.Printf("failed to close extracted file: %v", cerr)
		}
	}()

	const maxEntryBytes = int64(250 * 1024 * 1024) // 250 MiB

	// #nosec G110 -- ZIP payload validated before extraction
	n, err := io.Copy(outFile, io.LimitReader(rc, maxEntryBytes+1))
	if err != nil {
		return err
	}

	if n > maxEntryBytes {
		return fmt.Errorf("ZIP entry too large during extraction: %s", file.Name)
	}

	return nil
}
