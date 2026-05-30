package storage

import (
	"context"
	"strings"
	"testing"

	sharedstorage "github.com/mattboback/stageflow/libs/go/storage"
)

func TestAuthStorageStateUploaderWritesJobScopedAuthObject(t *testing.T) {
	t.Parallel()

	store := newMemoryStorage()
	uploader := NewAuthStorageStateUploader(store)

	key, err := uploader.UploadAuthStorageState(context.Background(), "job-1", []byte(`{"cookies":[]}`))
	if err != nil {
		t.Fatalf("UploadAuthStorageState: %v", err)
	}

	if key != "job-1/auth/storage-state.json" {
		t.Fatalf("key = %q", key)
	}

	stored := store.objects[store.key(sharedstorage.BucketArtifacts, key)]
	if string(stored) != `{"cookies":[]}` {
		t.Fatalf("stored auth state = %q", stored)
	}
}

func TestAuthStorageStateCleanerDeletesOnlyStorageStateCredential(t *testing.T) {
	t.Parallel()

	store := newMemoryStorage()
	cleaner := NewAuthStorageStateCleaner(store)
	ctx := context.Background()

	authKey := "job-1/auth/storage-state.json"
	reportKey := "job-1/report.html"
	store.objects[store.key(sharedstorage.BucketArtifacts, authKey)] = []byte("auth")
	store.objects[store.key(sharedstorage.BucketArtifacts, reportKey)] = []byte("report")

	if err := cleaner.DeleteAuthStorageState(ctx, authKey); err != nil {
		t.Fatalf("DeleteAuthStorageState valid key: %v", err)
	}

	if _, ok := store.objects[store.key(sharedstorage.BucketArtifacts, authKey)]; ok {
		t.Fatal("expected auth object to be deleted")
	}

	if err := cleaner.DeleteAuthStorageState(ctx, reportKey); err == nil {
		t.Fatal("expected report key delete to be rejected")
	}

	if err := cleaner.DeleteAuthStorageState(ctx, "../auth/storage-state.json"); err == nil ||
		!strings.Contains(err.Error(), "refusing non-auth key") {
		t.Fatalf("expected traversal key to be rejected, got %v", err)
	}

	if string(store.objects[store.key(sharedstorage.BucketArtifacts, reportKey)]) != "report" {
		t.Fatal("report object should not be deleted by auth cleaner")
	}
}
