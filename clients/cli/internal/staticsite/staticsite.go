// Package staticsite packages a local build directory (or an existing ZIP
// archive) for upload to the platform's ZIP intake.
package staticsite

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// MaxUploadBytes mirrors the platform API's ZIP upload limit.
const MaxUploadBytes = 100 * 1024 * 1024

// skippedDirs are directories never useful in a deployed static site.
var skippedDirs = map[string]struct{}{
	".git":         {},
	".svn":         {},
	".hg":          {},
	"node_modules": {},
}

// Target describes a ready-to-upload ZIP archive.
type Target struct {
	// ZipPath is the archive to upload.
	ZipPath string
	// HasRootIndex reports whether the archive has a top-level index.html.
	HasRootIndex bool
	// Cleanup removes any temp file this package created. Always non-nil.
	Cleanup func()
}

// IsPathTarget reports whether a scan argument refers to a local directory or
// ZIP archive rather than a URL.
func IsPathTarget(arg string) bool {
	if strings.Contains(arg, "://") {
		return false
	}

	info, err := os.Stat(arg)
	if err != nil {
		return false
	}

	return info.IsDir() || strings.EqualFold(filepath.Ext(arg), ".zip")
}

// Package prepares path (a directory or .zip file) for upload. Directories
// are zipped into a temp file; existing archives are used as-is.
func Package(path string) (Target, error) {
	info, err := os.Stat(path)
	if err != nil {
		return Target{}, fmt.Errorf("stat %s: %w", path, err)
	}

	if !info.IsDir() {
		return packageExistingZip(path, info.Size())
	}

	return packageDirectory(path)
}

func packageExistingZip(path string, size int64) (Target, error) {
	if size > MaxUploadBytes {
		return Target{}, fmt.Errorf(
			"%s is %d bytes; the API accepts ZIP uploads up to %d bytes",
			path, size, MaxUploadBytes,
		)
	}

	reader, err := zip.OpenReader(path)
	if err != nil {
		return Target{}, fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = reader.Close() }()

	hasIndex := false

	for _, f := range reader.File {
		if f.Name == "index.html" {
			hasIndex = true

			break
		}
	}

	return Target{ZipPath: path, HasRootIndex: hasIndex, Cleanup: func() {}}, nil
}

func packageDirectory(dir string) (Target, error) {
	tmp, err := os.CreateTemp("", "stageflow-site-*.zip")
	if err != nil {
		return Target{}, fmt.Errorf("create temp archive: %w", err)
	}

	cleanup := func() { _ = os.Remove(tmp.Name()) }

	hasIndex, err := writeDirArchive(tmp, dir)
	if err != nil {
		_ = tmp.Close()

		cleanup()

		return Target{}, err
	}

	if closeErr := tmp.Close(); closeErr != nil {
		cleanup()

		return Target{}, fmt.Errorf("finalize archive: %w", closeErr)
	}

	stat, err := os.Stat(tmp.Name())
	if err != nil {
		cleanup()

		return Target{}, fmt.Errorf("stat archive: %w", err)
	}

	if stat.Size() > MaxUploadBytes {
		cleanup()

		return Target{}, fmt.Errorf(
			"zipped %s is %d bytes; the API accepts ZIP uploads up to %d bytes",
			dir, stat.Size(), MaxUploadBytes,
		)
	}

	return Target{ZipPath: tmp.Name(), HasRootIndex: hasIndex, Cleanup: cleanup}, nil
}

func writeDirArchive(out io.Writer, dir string) (bool, error) {
	zw := zip.NewWriter(out)
	hasIndex := false
	fileCount := 0

	walkErr := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() {
			if _, skip := skippedDirs[entry.Name()]; skip && path != dir {
				return filepath.SkipDir
			}

			return nil
		}

		if !entry.Type().IsRegular() {
			// Symlinks and other special files are not part of a static site
			// and are rejected by the server-side archive validation.
			return nil
		}

		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return relErr
		}

		name := filepath.ToSlash(rel)
		if name == "index.html" {
			hasIndex = true
		}

		writer, createErr := zw.Create(name)
		if createErr != nil {
			return createErr
		}

		file, openErr := os.Open(path) // #nosec G304 -- walking a user-supplied directory is the point
		if openErr != nil {
			return openErr
		}

		_, copyErr := io.Copy(writer, file)

		_ = file.Close()

		if copyErr != nil {
			return copyErr
		}

		fileCount++

		return nil
	})
	if walkErr != nil {
		_ = zw.Close()

		return false, fmt.Errorf("zip %s: %w", dir, walkErr)
	}

	if fileCount == 0 {
		_ = zw.Close()

		return false, errors.New(dir + " contains no files to scan")
	}

	if err := zw.Close(); err != nil {
		return false, fmt.Errorf("zip %s: %w", dir, err)
	}

	return hasIndex, nil
}
