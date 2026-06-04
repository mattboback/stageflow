package db

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"
)

func TestNewDatabase(t *testing.T) {
	db := setupTestDB(t)

	if db.db == nil {
		t.Error("Expected database connection to be non-nil")
	}
}

func TestNewDatabase_RequiresURL(t *testing.T) {
	if _, err := NewDatabase(nil); err == nil {
		t.Fatal("expected error when config is nil")
	}

	if _, err := NewDatabase(&Config{URL: ""}); err == nil {
		t.Fatal("expected error when URL is empty")
	}
}

func TestInitSchema(t *testing.T) {
	db := setupTestDB(t)

	tables := []string{"jobs", "job_events"}

	for _, table := range tables {
		query := `
			SELECT table_name
			FROM information_schema.tables
			WHERE table_schema = current_schema()
			  AND table_name = $1
		`

		var name string

		scanErr := db.db.QueryRowContext(context.Background(), query, table).Scan(&name)
		if scanErr != nil {
			t.Errorf("Expected table %s to exist: %v", table, scanErr)
		}
	}
}

func TestInitSchemaDropsLegacyJobEventsJobIDNotNull(t *testing.T) {
	admin, err := sql.Open("pgx", testDatabaseURL)
	if err != nil {
		t.Fatalf("Failed to connect admin database: %v", err)
	}

	t.Cleanup(func() {
		if closeErr := admin.Close(); closeErr != nil {
			t.Fatalf("Failed to close admin database: %v", closeErr)
		}
	})

	schema := fmt.Sprintf("legacy_events_%d", time.Now().UnixNano())
	quotedSchema := quoteIdentifier(schema)

	if _, execErr := admin.ExecContext(
		context.Background(),
		fmt.Sprintf("CREATE SCHEMA %s", quotedSchema),
	); execErr != nil {
		t.Fatalf("Failed to create legacy schema %q: %v", schema, execErr)
	}

	t.Cleanup(func() {
		if _, dropErr := admin.ExecContext(
			context.Background(),
			fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", quotedSchema),
		); dropErr != nil {
			t.Fatalf("Failed to drop legacy schema %q: %v", schema, dropErr)
		}
	})

	legacyDDL := fmt.Sprintf(`
		SET search_path TO %s;
		CREATE TABLE jobs (
			id TEXT PRIMARY KEY,
			state TEXT NOT NULL DEFAULT 'PENDING',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			completed_at TIMESTAMP
		);
		CREATE TABLE job_events (
			id BIGSERIAL PRIMARY KEY,
			job_id TEXT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
			event TEXT NOT NULL,
			timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			payload_json TEXT
		);
	`, quotedSchema)
	if _, execErr := admin.ExecContext(context.Background(), legacyDDL); execErr != nil {
		t.Fatalf("Failed to create legacy job_events schema: %v", execErr)
	}

	dbURL := fmt.Sprintf("%s&search_path=%s", testDatabaseURL, schema)

	database, err := NewDatabase(&Config{URL: dbURL})
	if err != nil {
		t.Fatalf("NewDatabase failed on legacy schema: %v", err)
	}

	t.Cleanup(func() {
		if closeErr := database.Close(); closeErr != nil {
			t.Fatalf("Failed to close migrated database: %v", closeErr)
		}
	})

	if insertErr := database.InsertJobEvent(context.Background(), &JobEventInsert{
		JobID:     "missing-job",
		Event:     "scan.failed",
		Timestamp: time.Now().UTC(),
		Payload:   `{"job_id":"missing-job"}`,
	}); insertErr != nil {
		t.Fatalf("InsertJobEvent after legacy migration failed: %v", insertErr)
	}

	var jobIDIsNullable string

	nullabilityQuery := `
			SELECT is_nullable
			FROM information_schema.columns
		WHERE table_schema = $1
			  AND table_name = 'job_events'
			  AND column_name = 'job_id'
		`

	row := admin.QueryRowContext(context.Background(), nullabilityQuery, schema)
	if scanErr := row.Scan(&jobIDIsNullable); scanErr != nil {
		t.Fatalf("Failed to read migrated job_id nullability: %v", scanErr)
	}

	if jobIDIsNullable != "YES" {
		t.Fatalf("job_events.job_id is_nullable = %q, want YES", jobIDIsNullable)
	}
}

func TestClose(t *testing.T) {
	db := setupTestDB(t)

	if closeErr := db.Close(); closeErr != nil {
		t.Errorf("Failed to close database: %v", closeErr)
	}
}
