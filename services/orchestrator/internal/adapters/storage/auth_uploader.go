package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	sharedstorage "github.com/mattboback/stageflow/libs/go/storage"
)

// AuthStorageStateUploader uploads a captured Playwright storage-state blob
// shipped inline with the job-created event into the artifacts bucket under a
// stable, job-scoped key. It satisfies the AuthUploader port consumed by
// services/orchestrator/internal/application/jobs.Service.
//
// The bytes never leave this hop: the orchestrator persists only the returned
// artifact_key in JobConfig.Auth, and downstream artifacts (Provenance, stage
// log, NATS payloads) reference the same key. The Web UI's signed-URL surface
// does not expose this object.
type AuthStorageStateUploader struct {
	storage sharedstorage.Uploader
}

// AuthStorageStateCleaner removes the job-scoped storage-state object after
// scanners no longer need it. It refuses non-auth keys so cleanup cannot delete
// report artifacts accidentally.
type AuthStorageStateCleaner struct {
	storage sharedstorage.Deleter
}

// NewAuthStorageStateUploader returns an uploader bound to the given storage
// client. The shared storage client is the same one the rest of the
// orchestrator uses; we only need the Uploader half.
func NewAuthStorageStateUploader(storageClient sharedstorage.Uploader) *AuthStorageStateUploader {
	return &AuthStorageStateUploader{storage: storageClient}
}

func NewAuthStorageStateCleaner(storageClient sharedstorage.Deleter) *AuthStorageStateCleaner {
	return &AuthStorageStateCleaner{storage: storageClient}
}

// UploadAuthStorageState writes the storage-state JSON to
// scanner-artifacts/<jobID>/auth/storage-state.json and returns the relative
// key.
func (u *AuthStorageStateUploader) UploadAuthStorageState(
	ctx context.Context,
	jobID string,
	content []byte,
) (string, error) {
	if u == nil || u.storage == nil {
		return "", errors.New("auth storage_state uploader: no storage client configured")
	}

	if jobID == "" {
		return "", errors.New("auth storage_state uploader: jobID is required")
	}

	if len(content) == 0 {
		return "", errors.New("auth storage_state uploader: empty content")
	}

	key := fmt.Sprintf("%s/auth/storage-state.json", jobID)

	if err := u.storage.UploadFile(
		ctx,
		sharedstorage.BucketArtifacts,
		key,
		bytes.NewReader(content),
		int64(len(content)),
	); err != nil {
		return "", fmt.Errorf("upload auth storage_state to %s/%s: %w", sharedstorage.BucketArtifacts, key, err)
	}

	return key, nil
}

func (c *AuthStorageStateCleaner) DeleteAuthStorageState(ctx context.Context, key string) error {
	if c == nil || c.storage == nil {
		return errors.New("auth storage_state cleaner: no storage client configured")
	}

	if key == "" {
		return nil
	}

	if strings.HasPrefix(key, "/") || strings.Contains(key, "..") ||
		!strings.HasSuffix(key, "/auth/storage-state.json") {
		return fmt.Errorf("auth storage_state cleaner: refusing non-auth key %q", key)
	}

	if err := c.storage.DeleteFile(ctx, sharedstorage.BucketArtifacts, key); err != nil {
		return fmt.Errorf("delete auth storage_state %s/%s: %w", sharedstorage.BucketArtifacts, key, err)
	}

	return nil
}
