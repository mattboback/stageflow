package tombstone

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/mattboback/stageflow/services/platform-api/internal/sqlite"
)

func TestStorePreservesExistingDeletionHistory(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "projects.db")
	createLegacyDatabase(t, path)

	store, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}

	for _, id := range []string{"deleted-before-upgrade", "deleted-after-upgrade"} {
		if err = store.TombstoneJob(ctx, id); err != nil {
			t.Fatal(err)
		}

		if err = store.TombstoneJob(ctx, id); err != nil {
			t.Fatal(err)
		}
	}

	var count int

	err = store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM deleted_jobs
        WHERE job_id = 'deleted-before-upgrade' AND deleted_at = '2026-01-01 00:00:00'`).Scan(&count)
	if err != nil || count != 1 {
		t.Fatalf("existing tombstone changed: count=%d err=%v", count, err)
	}

	if err = store.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if closeErr := reopened.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	})

	for _, id := range []string{"deleted-before-upgrade", "deleted-after-upgrade"} {
		deleted, lookupErr := reopened.JobIsDeleted(ctx, id)
		if lookupErr != nil || !deleted {
			t.Fatalf("tombstone lost for %s: deleted=%v err=%v", id, deleted, lookupErr)
		}
	}

	deleted, err := reopened.JobIsDeleted(ctx, "never-deleted")
	if err != nil || deleted {
		t.Fatalf("unknown job: deleted=%v err=%v", deleted, err)
	}
}

func TestStoreCreatesFreshDatabase(t *testing.T) {
	t.Parallel()

	store, err := NewStore(filepath.Join(t.TempDir(), "projects.db"))
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	})

	if err = store.TombstoneJob(context.Background(), "job-1"); err != nil {
		t.Fatal(err)
	}

	deleted, err := store.JobIsDeleted(context.Background(), "job-1")
	if err != nil || !deleted {
		t.Fatalf("fresh store: deleted=%v err=%v", deleted, err)
	}
}

func createLegacyDatabase(t *testing.T, path string) {
	t.Helper()

	ctx := context.Background()

	legacy, err := sqlite.Open(path)
	if err != nil {
		t.Fatal(err)
	}

	_, err = legacy.ExecContext(ctx, `CREATE TABLE projects (id TEXT PRIMARY KEY);
 CREATE TABLE deleted_jobs (job_id TEXT PRIMARY KEY, deleted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP);
 INSERT INTO deleted_jobs (job_id, deleted_at) VALUES ('deleted-before-upgrade', '2026-01-01 00:00:00');
 PRAGMA user_version=2;`)
	if err != nil {
		t.Fatal(err)
	}

	if err = legacy.Close(); err != nil {
		t.Fatal(err)
	}
}
