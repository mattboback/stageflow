package db

import (
	"context"
	"testing"

	"github.com/mattboback/stageflow/libs/go/models"
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

	tables := []string{"jobs", "scanner_launches", "terminal_events", "maintenance_migrations"}

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

func TestInitSchemaDoesNotCreateJobEvents(t *testing.T) {
	database := setupTestDB(t)

	var exists bool
	if err := database.db.QueryRowContext(context.Background(), `
  SELECT EXISTS (SELECT 1 FROM information_schema.tables
  WHERE table_schema = current_schema() AND table_name = 'job_events')
 `).Scan(&exists); err != nil {
		t.Fatal(err)
	}

	if exists {
		t.Fatal("fresh database contains retired job_events table")
	}
}

func TestInitSchemaPreservesDormantJobEvents(t *testing.T) {
	database := setupTestDB(t)
	ctx := context.Background()

	createSensitiveTerminalJob(t, database, "legacy-audit-job", models.JobStateFailed)
	// An old audit table may remain through rollback. Deliberately omit newer
	// audit columns so any accidental application access fails this test.
	if _, err := database.db.ExecContext(ctx, `
  CREATE TABLE job_events (job_id TEXT NOT NULL, payload_json TEXT);
  INSERT INTO job_events VALUES ('legacy-audit-job', 'retained rollback evidence');
 `); err != nil {
		t.Fatal(err)
	}

	if err := database.initSchema(); err != nil {
		t.Fatal(err)
	}

	if _, err := database.SanitizeLegacyTerminalRecords(ctx); err != nil {
		t.Fatal(err)
	}

	assertTerminalJobConfigRedacted(t, database, "legacy-audit-job")

	var payload string
	if err := database.db.QueryRowContext(ctx, `SELECT payload_json FROM job_events`).Scan(&payload); err != nil {
		t.Fatal(err)
	}

	if payload != "retained rollback evidence" {
		t.Fatalf("dormant audit row changed: %q", payload)
	}

	var nullable string
	if err := database.db.QueryRowContext(ctx, `
  SELECT is_nullable FROM information_schema.columns
  WHERE table_schema = current_schema() AND table_name = 'job_events' AND column_name = 'job_id'
 `).Scan(&nullable); err != nil {
		t.Fatal(err)
	}

	if nullable != "NO" {
		t.Fatalf("dormant audit schema changed: nullable=%s", nullable)
	}
}

func TestClose(t *testing.T) {
	db := setupTestDB(t)

	if closeErr := db.Close(); closeErr != nil {
		t.Errorf("Failed to close database: %v", closeErr)
	}
}
