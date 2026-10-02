package orchestrator

import (
	"context"
	"errors"
	"strings"

	storage "github.com/mattboback/stageflow/internal/artifacts"
)

type authFileCleaner struct{ files storage.Deleter }

func (c authFileCleaner) DeleteAuthStorageState(ctx context.Context, key string) error {
	if !strings.HasSuffix(key, "/auth.json") {
		return errors.New("refusing to delete non-auth file")
	}

	return c.files.DeleteFile(ctx, storage.BucketArtifacts, key)
}
