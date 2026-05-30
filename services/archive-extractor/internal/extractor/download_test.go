package extractor

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/minio/minio-go/v7"
)

type fakeDownloader struct {
	reader io.ReadCloser
	err    error
}

func (f fakeDownloader) DownloadFile(context.Context, string, string) (io.ReadCloser, error) {
	if f.err != nil {
		return nil, f.err
	}

	return f.reader, nil
}

type statReadCloser struct {
	info    minio.ObjectInfo
	statErr error
	readErr error
}

func (r *statReadCloser) Stat() (minio.ObjectInfo, error) {
	if r.statErr != nil {
		return minio.ObjectInfo{}, r.statErr
	}

	return r.info, nil
}

func (r *statReadCloser) Read([]byte) (int, error) {
	if r.readErr != nil {
		return 0, r.readErr
	}

	return 0, io.EOF
}

func (r *statReadCloser) Close() error {
	return nil
}

func TestExtract_RejectsOversizedCompressedObjectMetadataBeforeRead(t *testing.T) {
	reader := &statReadCloser{
		info:    minio.ObjectInfo{Size: maxZipCompressedSize + 1},
		readErr: errors.New("read should not be called after oversized metadata"),
	}

	ext := &Extractor{
		storageClient: fakeDownloader{reader: reader},
	}

	err := ext.Extract(context.Background(), "bucket", "large-object", t.TempDir())
	if err == nil {
		t.Fatal("expected oversized compressed object error, got nil")
	}

	if !strings.Contains(err.Error(), "compressed ZIP object too large") {
		t.Fatalf("expected compressed ZIP size error, got %v", err)
	}
}

func TestCopyCompressedZIPObject_RejectsOversizedInput(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "download-*.zip")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer tmpFile.Close()

	const limit = int64(8)

	n, err := copyCompressedZIPObject(tmpFile, strings.NewReader("123456789more data"), limit)
	if err == nil {
		t.Fatal("expected oversized compressed object error, got nil")
	}

	if !strings.Contains(err.Error(), "compressed ZIP object too large") {
		t.Fatalf("expected compressed ZIP size error, got %v", err)
	}

	if n != limit+1 {
		t.Fatalf("copied bytes = %d, want %d", n, limit+1)
	}

	info, statErr := tmpFile.Stat()
	if statErr != nil {
		t.Fatalf("stat temp file: %v", statErr)
	}

	if info.Size() != limit+1 {
		t.Fatalf("temp file size = %d, want %d", info.Size(), limit+1)
	}
}

func TestExtract_DownloadedNormalZIPSuccess(t *testing.T) {
	zipPath := createTestZIP(t, map[string]string{
		"index.html":     "<html><body>Index</body></html>",
		"assets/app.css": "body { color: #111; }",
	})

	zipBytes, err := os.ReadFile(zipPath) // #nosec G304 -- reading controlled temp path
	if err != nil {
		t.Fatalf("read test ZIP: %v", err)
	}

	ext := &Extractor{
		storageClient: fakeDownloader{reader: io.NopCloser(bytes.NewReader(zipBytes))},
	}

	destDir := t.TempDir()
	if extractErr := ext.Extract(context.Background(), "bucket", "site.zip", destDir); extractErr != nil {
		t.Fatalf("Extract returned error: %v", extractErr)
	}

	content, readErr := os.ReadFile(filepath.Join(destDir, "index.html")) // #nosec G304 -- reading controlled temp path
	if readErr != nil {
		t.Fatalf("read extracted index: %v", readErr)
	}

	if string(content) != "<html><body>Index</body></html>" {
		t.Fatalf("extracted index content = %q", string(content))
	}
}
