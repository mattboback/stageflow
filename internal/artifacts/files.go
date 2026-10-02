// Package artifacts stores job inputs and results as files under one data directory.
package artifacts

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"regexp"
	"strings"
	"syscall"

	"github.com/google/uuid"
)

const (
	// MaxArtifactBytes bounds every file a job produces: reports, results, auth state.
	MaxArtifactBytes int64 = 64 << 20
	// MaxInputBytes bounds an uploaded ZIP.
	MaxInputBytes int64 = 512 << 20
	// MinFreeDiskBytes is the free space intake and claiming require.
	MinFreeDiskBytes = uint64(2 << 30)

	inputName = "input.zip"
)

var safeJob = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

var ErrLowDisk = errors.New("less than 2 GB of disk space is available")

// Files confines all operations to the jobs directory using os.Root. Every
// file lives at "<job ID>/<path>". Writers publish complete files with rename,
// so readers never observe partial reports.
type Files struct{ root *os.Root }

func NewFiles(dataDir string) (*Files, error) {
	if err := os.MkdirAll(path.Join(dataDir, "jobs"), 0o700); err != nil {
		return nil, err
	}

	root, err := os.OpenRoot(path.Join(dataDir, "jobs"))
	if err != nil {
		return nil, err
	}

	return &Files{root: root}, nil
}

func (f *Files) Close() error { return f.root.Close() }

func (f *Files) CheckDisk(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	var info syscall.Statfs_t
	if err := syscall.Statfs(f.root.Name(), &info); err != nil {
		return err
	}

	if info.Bsize <= 0 {
		return errors.New("invalid filesystem block size")
	}

	blockSize := uint64(info.Bsize)
	if info.Bavail < (MinFreeDiskBytes+blockSize-1)/blockSize {
		return ErrLowDisk
	}

	return nil
}

// Write stores one job file of at most MaxArtifactBytes.
func (f *Files) Write(ctx context.Context, key string, reader io.Reader) error {
	return f.write(ctx, key, reader, MaxArtifactBytes)
}

// WriteInput stores a job's uploaded ZIP, of at most MaxInputBytes.
func (f *Files) WriteInput(ctx context.Context, jobID string, reader io.Reader) error {
	return f.write(ctx, jobID+"/"+inputName, reader, MaxInputBytes)
}

// OpenInput opens a job's uploaded ZIP.
func (f *Files) OpenInput(ctx context.Context, jobID string) (*os.File, error) {
	if !safeJob.MatchString(jobID) {
		return nil, errors.New("invalid job ID")
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return f.openRegularFile(jobID + "/" + inputName)
}

// Open opens one job file for reading.
func (f *Files) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	key, scopeErr := scoped(key)
	if scopeErr != nil {
		return nil, scopeErr
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return f.openRegularFile(key)
}

// Remove deletes one job file. A file that is already gone is not an error.
func (f *Files) Remove(ctx context.Context, key string) error {
	key, scopeErr := scoped(key)
	if scopeErr != nil {
		return scopeErr
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	err := f.root.Remove(key)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	return err
}

// RemoveJob deletes everything a job stored.
func (f *Files) RemoveJob(ctx context.Context, jobID string) error {
	if !safeJob.MatchString(jobID) {
		return errors.New("deletion requires a job ID")
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	return f.root.RemoveAll(jobID)
}

func scoped(key string) (string, error) {
	parts := strings.Split(key, "/")
	if len(parts) < 2 || !safeJob.MatchString(parts[0]) {
		return "", errors.New("invalid job artifact path")
	}

	for _, part := range parts {
		if part == "" || part == "." || part == ".." || strings.ContainsAny(part, "\\\x00") {
			return "", errors.New("invalid artifact path component")
		}
	}

	return key, nil
}

func (f *Files) write(ctx context.Context, key string, reader io.Reader, limit int64) error {
	key, scopeErr := scoped(key)
	if scopeErr != nil {
		return scopeErr
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	if err := f.root.MkdirAll(path.Dir(key), 0o700); err != nil {
		return err
	}

	tmp := path.Dir(key) + "/.write-" + uuid.NewString()

	mode := os.FileMode(0o600)
	if PublicPath(strings.TrimPrefix(key, strings.Split(key, "/")[0]+"/")) {
		mode = 0o644
	}

	file, err := f.root.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer f.removeTemporary(tmp)

	n, copyErr := io.Copy(file, io.LimitReader(reader, limit+1))
	if copyErr == nil && n > limit {
		copyErr = fmt.Errorf("file exceeds %d byte limit", limit)
	}

	if copyErr == nil {
		copyErr = ctx.Err()
	}

	if copyErr == nil {
		copyErr = file.Sync()
	}

	closeErr := file.Close()

	if copyErr != nil {
		return copyErr
	}

	if closeErr != nil {
		return closeErr
	}

	return f.root.Rename(tmp, key)
}

func (f *Files) removeTemporary(name string) {
	if err := f.root.Remove(name); err != nil && !errors.Is(err, os.ErrNotExist) {
		slog.Warn("Failed to remove temporary artifact", "error", err)
	}
}

// openRegularFile walks without following links at any component. Confining
// access to the volume alone would allow a worker's public result to link to
// another job's private auth file inside that same volume.
func (f *Files) openRegularFile(key string) (*os.File, error) {
	dir, err := f.root.Open(".")
	if err != nil {
		return nil, err
	}

	parts := strings.Split(key, "/")
	for i, part := range parts {
		flags := syscall.O_RDONLY | syscall.O_NOFOLLOW | syscall.O_CLOEXEC | syscall.O_NONBLOCK
		if i < len(parts)-1 {
			flags |= syscall.O_DIRECTORY
		}

		fd, openErr := syscall.Openat(int(dir.Fd()), part, flags, 0)
		_ = dir.Close()

		if openErr != nil {
			return nil, openErr
		}

		dir = os.NewFile(uintptr(fd), part)
	}

	info, err := dir.Stat()
	if err != nil || !info.Mode().IsRegular() {
		_ = dir.Close()

		if err != nil {
			return nil, err
		}

		return nil, errors.New("artifact is not a regular file")
	}

	return dir, nil
}

// PublicPath is the single allow-list used by the API file route.
func PublicPath(value string) bool {
	if value == "report.json" || value == "provenance.json" {
		return true
	}

	parts := strings.Split(value, "/")
	if len(parts) == 2 && parts[1] == "results.json" {
		return scanner(parts[0])
	}

	if len(parts) != 4 || !scanner(parts[0]) || parts[2] != "screenshots" || !safeJob.MatchString(parts[1]) {
		return false
	}

	switch strings.ToLower(path.Ext(parts[3])) {
	case ".png", ".jpg", ".jpeg", ".webp":
		return !strings.HasPrefix(parts[3], ".") && !strings.ContainsAny(parts[3], "\\\x00")
	default:
		return false
	}
}

func scanner(value string) bool {
	switch value {
	case "axe", "lighthouse", "link-checker", "security-headers", "seo":
		return true
	}

	return false
}
