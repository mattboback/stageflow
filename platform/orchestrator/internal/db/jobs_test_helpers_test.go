package db

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/mattboback/stageflow/packages/shared-go/models"
)

func setupTestDB(t *testing.T) *Database {
	t.Helper()
	db, err := NewDatabase(&Config{Path: ":memory:"})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("Failed to close database: %v", err)
		}
	})
	return db
}

func setupTestDBFile(t *testing.T) *Database {
	t.Helper()
	path := filepath.Join(t.TempDir(), "jobs.db")
	db, err := NewDatabase(&Config{Path: path})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("Failed to close database: %v", err)
		}
	})
	return db
}

func mustCreateJob(t *testing.T, db *Database, job *models.Job) {
	t.Helper()
	if err := db.CreateJob(context.Background(), job); err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}
}

func mustLogEvent(t *testing.T, db *Database, jobID, event, payload string) {
	t.Helper()
	if err := db.InsertJobEvent(context.Background(), &JobEventInsert{
		JobID:     jobID,
		Event:     event,
		Timestamp: time.Now().UTC(),
		Payload:   payload,
	}); err != nil {
		t.Fatalf("Failed to log event: %v", err)
	}
}
