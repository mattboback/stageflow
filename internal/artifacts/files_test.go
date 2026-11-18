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
		if err := files.Write(ctx, key, strings.NewReader("secret")); err == nil {
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

	if err := files.Write(ctx, "job/escape/report.json", strings.NewReader("secret")); err == nil {
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
	if err := files.Write(ctx, key, strings.NewReader("original")); err != nil {
		t.Fatal(err)
	}

	if err := files.write(ctx, key, strings.NewReader("replacement"), 8); err == nil {
		t.Fatal("accepted oversized artifact")
	}

	reader, err := files.Open(ctx, key)
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
	if err := files.Write(ctx, "other/auth/storage-state.json", strings.NewReader("secret")); err != nil {
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

	if reader, err := files.Open(ctx, "job/axe/results.json"); err == nil {
		reader.Close()
		t.Fatal("public result followed a link to private auth")
	}

	if err := os.Symlink("other", filepath.Join(dir, "jobs", "linked")); err != nil {
		t.Fatal(err)
	}

	if reader, err := files.Open(ctx, "linked/auth/storage-state.json"); err == nil {
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

	if reader, err := files.Open(context.Background(), "job/report.json"); err == nil {
		reader.Close()
		t.Fatal("accepted non-regular report")
	}
}

func TestFilesStoreInputAndRemoveJob(t *testing.T) {
	files, initErr := NewFiles(t.TempDir())
	if initErr != nil {
		t.Fatal(initErr)
	}
	defer files.Close()

	ctx := context.Background()
	if err := files.WriteInput(ctx, "job", strings.NewReader("zip")); err != nil {
		t.Fatal(err)
	}

	if err := files.Write(ctx, "job/auth.json", strings.NewReader("{}")); err != nil {
		t.Fatal(err)
	}

	input, err := files.OpenInput(ctx, "job")
	if err != nil {
		t.Fatal(err)
	}

	data, err := io.ReadAll(input)
	input.Close()

	if err != nil || string(data) != "zip" {
		t.Fatalf("input = %q %v", data, err)
	}

	// Removing a file twice is fine: cleanup after a rejected job may repeat.
	for range 2 {
		if removeErr := files.Remove(ctx, "job/auth.json"); removeErr != nil {
			t.Fatal(removeErr)
		}
	}

	if reader, openErr := files.Open(ctx, "job/auth.json"); openErr == nil {
		reader.Close()
		t.Fatal("removed file is still readable")
	}

	if removeErr := files.RemoveJob(ctx, "../job"); removeErr == nil {
		t.Fatal("removed a path that is not a job ID")
	}

	if removeErr := files.RemoveJob(ctx, "job"); removeErr != nil {
		t.Fatal(removeErr)
	}

	if reader, openErr := files.OpenInput(ctx, "job"); openErr == nil {
		reader.Close()
		t.Fatal("job input survived RemoveJob")
	}
}
