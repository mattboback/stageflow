package db

import (
	"context"
	"testing"
)

func TestNewDatabase(t *testing.T) {
	db := setupTestDB(t)

	if db.DB() == nil {
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

		scanErr := db.DB().QueryRowContext(context.Background(), query, table).Scan(&name)
		if scanErr != nil {
			t.Errorf("Expected table %s to exist: %v", table, scanErr)
		}
	}
}

func TestClose(t *testing.T) {
	db := setupTestDB(t)

	if closeErr := db.Close(); closeErr != nil {
		t.Errorf("Failed to close database: %v", closeErr)
	}
}
