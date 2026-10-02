package artifacts

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestFilesConfinesPathsAndSymlinks(t *testing.T) {
	dir := t.TempDir()

	files, initErr := NewFiles(dir)
	if initErr != nil {
		t.Fatal(initErr)
	}
	defer files.Close()

	ctx := context.Background()
	for _, key := range []string{"../outside/report.json", "job/../../outside", "job/../report.json", "job//report.json", "/job/report.json", "job/report\\x"} {
		if err := files.UploadFile(ctx, BucketArtifacts, key, strings.NewReader("secret"), 6); err == nil {
			t.Errorf("accepted %q", key)
		}
	}

	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "jobs", "job"), 0o700); err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink(outside, filepath.Join(dir, "jobs", "job", "escape")); err != nil {
		t.Fatal(err)
	}

	if err := files.UploadFile(
		ctx,
		BucketArtifacts,
		"job/escape/report.json",
		strings.NewReader("secret"),
		6,
	); err == nil {
		t.Fatal("symlink escaped root")
	}
}

func TestFilesPublishesCompleteReportAndRejectsOversize(t *testing.T) {
	files, initErr := NewFiles(t.TempDir())
	if initErr != nil {
		t.Fatal(initErr)
	}
	defer files.Close()

	ctx := context.Background()

	key := "job/report.json"
	if err := files.UploadFile(ctx, BucketArtifacts, key, strings.NewReader("original"), 8); err != nil {
		t.Fatal(err)
	}

	if err := files.UploadFile(
		ctx,
		BucketArtifacts,
		key,
		strings.NewReader("replacement"),
		MaxArtifactBytes+1,
	); err == nil {
		t.Fatal("accepted known oversized artifact")
	}

	reader, err := files.DownloadFile(ctx, BucketArtifacts, key)
	if err != nil {
		t.Fatal(err)
	}

	data, err := io.ReadAll(reader)
	reader.Close()

	if err != nil || string(data) != "original" {
		t.Fatalf("oversize write corrupted original: %q %v", data, err)
	}
}

func TestPublicFilesAllowList(t *testing.T) {
	for _, name := range []string{"report.json", "provenance.json", "axe/results.json", "seo/page-1/screenshots/overview.webp"} {
		if !PublicPath(name) {
			t.Errorf("rejected public %s", name)
		}
	}

	for _, name := range []string{"auth.json", "input.zip", "site/index.html", "axe/progress.jsonl", "axe/outcome.json", "axe/page-1/screenshots/x.svg", "axe/../screenshots/x.png", "axe/page-1/screenshots/../x.png"} {
		if PublicPath(name) {
			t.Errorf("exposed private %s", name)
		}
	}
}

func TestPublicReadRejectsLinksToPrivateFilesAndOtherJobs(t *testing.T) {
	dir := t.TempDir()

	files, initErr := NewFiles(dir)
	if initErr != nil {
		t.Fatal(initErr)
	}
	defer files.Close()

	ctx := context.Background()
	if err := files.UploadFile(
		ctx,
		BucketArtifacts,
		"other/auth/storage-state.json",
		strings.NewReader("secret"),
		6,
	); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Join(dir, "jobs", "job", "axe"), 0o700); err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink(
		"../../other/auth/storage-state.json",
		filepath.Join(dir, "jobs", "job", "axe", "results.json"),
	); err != nil {
		t.Fatal(err)
	}

	if reader, err := files.DownloadFile(ctx, BucketArtifacts, "job/axe/results.json"); err == nil {
		reader.Close()
		t.Fatal("public result followed a link to private auth")
	}

	if err := os.Symlink("other", filepath.Join(dir, "jobs", "linked")); err != nil {
		t.Fatal(err)
	}

	if reader, err := files.DownloadFile(ctx, BucketArtifacts, "linked/auth/storage-state.json"); err == nil {
		reader.Close()
		t.Fatal("read followed a job directory link")
	}
}

func TestArtifactReadRejectsFIFOWithoutWaitingForWriter(t *testing.T) {
	dir := t.TempDir()

	files, initErr := NewFiles(dir)
	if initErr != nil {
		t.Fatal(initErr)
	}
	defer files.Close()

	if err := os.MkdirAll(filepath.Join(dir, "jobs", "job"), 0o700); err != nil {
		t.Fatal(err)
	}

	if err := syscall.Mkfifo(filepath.Join(dir, "jobs", "job", "report.json"), 0o600); err != nil {
		t.Fatal(err)
	}

	if reader, err := files.DownloadFile(context.Background(), BucketArtifacts, "job/report.json"); err == nil {
		reader.Close()
		t.Fatal("accepted non-regular report")
	}
}
