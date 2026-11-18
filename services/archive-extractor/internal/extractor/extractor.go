// Package extractor downloads and validates customer uploads.
package extractor

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattboback/stageflow/internal/archive"
)

const (
	maxZipEntries               = archive.MaxEntries
	maxZipCompressedSize        = archive.MaxCompressedSize
	maxZipUncompressedSize      = archive.MaxUncompressedSize
	maxZipEntryUncompressedSize = archive.MaxEntryUncompressedSize
)

// Extractor validates and unpacks job ZIPs from the scoped read-only job mount.
type Extractor struct{}

func NewExtractor() *Extractor { return &Extractor{} }

func (e *Extractor) Extract(_ context.Context, zipPath, destDir string) error {
	info, err := os.Stat(zipPath)
	if err != nil {
		return fmt.Errorf("stat ZIP: %w", err)
	}

	if !info.Mode().IsRegular() {
		return errors.New("ZIP input must be a regular file")
	}

	if validationErr := validateZIP(zipPath, info.Size()); validationErr != nil {
		return validationErr
	}

	return extractZIP(zipPath, destDir)
}

// validateZIP shares intake checks with the API; extraction bounds actual bytes too.
func validateZIP(zipPath string, fileSize int64) error {
	file, err := os.Open(zipPath)
	if err != nil {
		return err
	}
	defer file.Close()

	return archive.Validate(file, fileSize)
}
func sanitizeZipEntryName(name string) (string, error) { return archive.SanitizeEntryName(name) }

func extractZIP(zipPath, destDir string) error {
	return extractZIPWithLimits(zipPath, destDir, maxZipUncompressedSize, maxZipEntryUncompressedSize)
}

func extractZIPWithLimits(zipPath, destDir string, maxAggregateBytes, maxEntryBytes int64) error {
	if err := validateExtractionLimits(maxAggregateBytes, maxEntryBytes); err != nil {
		return err
	}

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}

	defer func() {
		if cerr := r.Close(); cerr != nil {
			slog.Warn("failed to close ZIP reader", "error", cerr)
		}
	}()

	if mkdirErr := os.MkdirAll(filepath.Dir(destDir), 0o750); mkdirErr != nil {
		return fmt.Errorf("failed to create destination directory: %w", mkdirErr)
	}

	tmpDir, tmpErr := os.MkdirTemp(filepath.Dir(destDir), "."+filepath.Base(destDir)+"-*")
	if tmpErr != nil {
		return fmt.Errorf("failed to create temporary extraction directory: %w", tmpErr)
	}

	extractionComplete := false
	defer func() {
		if !extractionComplete {
			_ = os.RemoveAll(tmpDir)
		}
	}()

	var totalExtracted int64

	for _, file := range r.File {
		n, extractErr := extractZipEntry(file, tmpDir, maxEntryBytes)
		if extractErr != nil {
			return extractErr
		}

		totalExtracted += n
		if sizeErr := enforceAggregateExtractionLimit(totalExtracted, maxAggregateBytes); sizeErr != nil {
			return sizeErr
		}
	}

	if replaceErr := replaceDirectory(destDir, tmpDir); replaceErr != nil {
		return replaceErr
	}

	extractionComplete = true

	return nil
}

func replaceDirectory(destDir, tmpDir string) error {
	destDir = filepath.Clean(destDir)
	tmpDir = filepath.Clean(tmpDir)

	backupDir := ""

	if _, statErr := os.Stat(destDir); statErr == nil {
		parent := filepath.Dir(destDir)

		createdBackup, mkErr := os.MkdirTemp(parent, "."+filepath.Base(destDir)+"-old-*")
		if mkErr != nil {
			return fmt.Errorf("failed to create backup directory: %w", mkErr)
		}

		if removeErr := os.Remove(createdBackup); removeErr != nil {
			return fmt.Errorf("failed to prepare backup directory: %w", removeErr)
		}

		if renameErr := os.Rename(destDir, createdBackup); renameErr != nil {
			return fmt.Errorf("failed to move existing destination aside: %w", renameErr)
		}

		backupDir = createdBackup
	} else if !os.IsNotExist(statErr) {
		return fmt.Errorf("failed to stat destination directory: %w", statErr)
	}

	if renameErr := os.Rename(tmpDir, destDir); renameErr != nil {
		if backupDir != "" {
			if restoreErr := os.Rename(backupDir, destDir); restoreErr != nil {
				return errors.Join(
					fmt.Errorf("failed to replace destination directory: %w", renameErr),
					fmt.Errorf("failed to restore backup directory: %w", restoreErr),
				)
			}
		}

		return fmt.Errorf("failed to replace destination directory: %w", renameErr)
	}

	if backupDir != "" {
		_ = os.RemoveAll(backupDir)
	}

	return nil
}

func validateExtractionLimits(maxAggregateBytes, maxEntryBytes int64) error {
	if maxAggregateBytes < 1 {
		return errors.New("max aggregate bytes must be positive")
	}

	if maxEntryBytes < 1 {
		return errors.New("max entry bytes must be positive")
	}

	return nil
}

func extractZipEntry(file *zip.File, destDir string, maxEntryBytes int64) (int64, error) {
	cleanName, sanitizeErr := sanitizeZipEntryName(file.Name)
	if sanitizeErr != nil {
		return 0, sanitizeErr
	}

	// #nosec G305 -- file path is validated below before use.
	fpath := filepath.Join(destDir, filepath.FromSlash(cleanName))
	if !isWithinBaseDir(destDir, fpath) {
		return 0, fmt.Errorf("illegal file path: %s", file.Name)
	}

	if zipEntryIsDir(file) {
		if mkErr := os.MkdirAll(fpath, 0o750); mkErr != nil {
			return 0, fmt.Errorf("failed to create directory %s: %w", fpath, mkErr)
		}

		return 0, nil
	}

	if parentErr := os.MkdirAll(filepath.Dir(fpath), 0o750); parentErr != nil {
		return 0, fmt.Errorf("failed to create parent directory for %s: %w", fpath, parentErr)
	}

	n, extractErr := extractFile(file, fpath, maxEntryBytes)
	if extractErr != nil {
		return 0, fmt.Errorf("failed to extract %s: %w", file.Name, extractErr)
	}

	return n, nil
}

func zipEntryIsDir(file *zip.File) bool {
	return file.FileInfo().IsDir() || strings.HasSuffix(strings.ReplaceAll(file.Name, "\\", "/"), "/")
}

func enforceAggregateExtractionLimit(totalExtracted, maxAggregateBytes int64) error {
	if totalExtracted <= maxAggregateBytes {
		return nil
	}

	return fmt.Errorf(
		"ZIP too large during extraction: %d bytes exceeds %d byte limit",
		totalExtracted,
		maxAggregateBytes,
	)
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

func extractFile(file *zip.File, destPath string, maxEntryBytes int64) (int64, error) {
	rc, err := file.Open()
	if err != nil {
		return 0, err
	}

	defer func() {
		if cerr := rc.Close(); cerr != nil {
			slog.Warn("failed to close zip entry reader", "error", cerr)
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
		return 0, err
	}

	defer func() {
		if cerr := outFile.Close(); cerr != nil {
			slog.Warn("failed to close extracted file", "error", cerr)
		}
	}()

	// #nosec G110 -- ZIP payload validated before extraction
	n, err := io.Copy(outFile, io.LimitReader(rc, maxEntryBytes+1))
	if err != nil {
		return n, err
	}

	if n > maxEntryBytes {
		return n, fmt.Errorf("ZIP entry too large during extraction: %s", file.Name)
	}

	return n, nil
}
