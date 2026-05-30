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
	"path"
	"path/filepath"
	"strings"

	"github.com/mattboback/stageflow/libs/go/storage"
	"github.com/minio/minio-go/v7"
)

const (
	maxZipEntries               = 5000                   // protect against tiny-file floods
	maxZipExpansionRatio        = 100                    // 100x expansion max
	maxZipCompressedSize        = 100 * 1024 * 1024      // mirror platform-api ZIP upload limit
	maxZipUncompressedSize      = 1 * 1024 * 1024 * 1024 // 1 GiB
	maxZipEntryUncompressedSize = 250 * 1024 * 1024      // 250 MiB
	maxZipNameLen               = 4096
)

type objectDownloader interface {
	DownloadFile(ctx context.Context, bucket, path string) (io.ReadCloser, error)
}

type objectStatReader interface {
	Stat() (minio.ObjectInfo, error)
}

// Extractor downloads job ZIPs from staging and extracts them safely.
type Extractor struct {
	storageClient objectDownloader
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
			slog.Warn("failed to remove temp file", "path", tmpFile.Name(), "error", cerr)
		}
	}()
	defer func() {
		if cerr := tmpFile.Close(); cerr != nil {
			slog.Warn("failed to close temp file", "error", cerr)
		}
	}()

	obj, err := e.storageClient.DownloadFile(ctx, bucket, objectPath)
	if err != nil {
		return fmt.Errorf("failed to get object from MinIO: %w", err)
	}

	defer func() {
		if cerr := obj.Close(); cerr != nil {
			slog.Warn("failed to close MinIO object", "error", cerr)
		}
	}()

	if statErr := enforceCompressedZIPObjectMetadataLimit(obj, maxZipCompressedSize); statErr != nil {
		return statErr
	}

	size, err := copyCompressedZIPObject(tmpFile, obj, maxZipCompressedSize)
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

func enforceCompressedZIPObjectMetadataLimit(obj io.Reader, maxBytes int64) error {
	statReader, ok := obj.(objectStatReader)
	if !ok {
		return nil
	}

	info, err := statReader.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat ZIP object: %w", err)
	}

	if info.Size > maxBytes {
		return fmt.Errorf(
			"compressed ZIP object too large (%d bytes > %d byte limit)",
			info.Size,
			maxBytes,
		)
	}

	return nil
}

func copyCompressedZIPObject(dst io.Writer, src io.Reader, maxBytes int64) (int64, error) {
	n, err := io.Copy(dst, io.LimitReader(src, maxBytes+1))
	if err != nil {
		return n, err
	}

	if n > maxBytes {
		return n, fmt.Errorf(
			"compressed ZIP object too large (%d bytes > %d byte limit)",
			n,
			maxBytes,
		)
	}

	return n, nil
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
			slog.Warn("failed to close validation reader", "error", cerr)
		}
	}()

	if len(r.File) > maxZipEntries {
		return fmt.Errorf("ZIP has too many files (%d > %d)", len(r.File), maxZipEntries)
	}

	var totalUncompressed uint64

	for _, file := range r.File {
		if file.Name == "" || len(file.Name) > maxZipNameLen {
			return fmt.Errorf("invalid entry name length (%d)", len(file.Name))
		}

		if strings.ContainsRune(file.Name, '\x00') {
			return fmt.Errorf("invalid entry name (NUL byte): %q", file.Name)
		}

		if _, sanitizeErr := sanitizeZipEntryName(file.Name); sanitizeErr != nil {
			return sanitizeErr
		}

		if file.UncompressedSize64 > maxZipEntryUncompressedSize {
			return fmt.Errorf(
				"ZIP entry too large (%s: %d bytes > %d)",
				file.Name,
				file.UncompressedSize64,
				maxZipEntryUncompressedSize,
			)
		}

		totalUncompressed += file.UncompressedSize64
	}

	if fileSize > 0 {
		ratio := float64(totalUncompressed) / float64(fileSize)
		if ratio > maxZipExpansionRatio {
			return fmt.Errorf("ZIP bomb detected: compression ratio %.1fx exceeds limit", ratio)
		}
	}

	if totalUncompressed > maxZipUncompressedSize {
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

	if mkdirErr := os.MkdirAll(destDir, 0o750); mkdirErr != nil {
		return fmt.Errorf("failed to create destination directory: %w", mkdirErr)
	}

	var totalExtracted int64

	for _, file := range r.File {
		n, extractErr := extractZipEntry(file, destDir, maxEntryBytes)
		if extractErr != nil {
			return extractErr
		}

		totalExtracted += n
		if sizeErr := enforceAggregateExtractionLimit(totalExtracted, maxAggregateBytes); sizeErr != nil {
			return sizeErr
		}
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
