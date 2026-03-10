package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mattboback/stageflow/packages/shared-go/models"
)

var testSchemaCounter uint64

func setupTestDB(t *testing.T) *Database {
	t.Helper()

	db := setupIsolatedTestDB(t)

	return db
}

func setupTestDBFile(t *testing.T) *Database {
	t.Helper()

	db := setupIsolatedTestDB(t)

	return db
}

func setupIsolatedTestDB(t *testing.T) *Database {
	t.Helper()

	admin, err := sql.Open("pgx", testDatabaseURL)
	if err != nil {
		t.Fatalf("Failed to connect admin database: %v", err)
	}

	t.Cleanup(func() {
		if closeErr := admin.Close(); closeErr != nil {
			t.Fatalf("Failed to close admin database: %v", closeErr)
		}
	})

	schema := fmt.Sprintf("t_%d_%d", time.Now().UnixNano(), atomic.AddUint64(&testSchemaCounter, 1))

	createSchemaQuery := fmt.Sprintf("CREATE SCHEMA %s", quoteIdentifier(schema))
	if _, execErr := admin.ExecContext(context.Background(), createSchemaQuery); execErr != nil {
		t.Fatalf("Failed to create test schema %q: %v", schema, execErr)
	}

	dbURL := fmt.Sprintf("%s&search_path=%s", testDatabaseURL, schema)

	db, err := NewDatabase(&Config{URL: dbURL})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	t.Cleanup(func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Fatalf("Failed to close database: %v", closeErr)
		}

		dropSchemaQuery := fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", quoteIdentifier(schema))
		if _, dropErr := admin.ExecContext(context.Background(), dropSchemaQuery); dropErr != nil {
			t.Fatalf("Failed to drop test schema %q: %v", schema, dropErr)
		}
	})

	return db
}

func quoteIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
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
