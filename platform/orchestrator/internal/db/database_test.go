package db

import (
	"context"
	"os"
	"testing"
)

func TestNewDatabase(t *testing.T) {
	// Use in-memory database for testing
	config := &Config{
		Path: ":memory:",
	}

	db, err := NewDatabase(config)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatalf("Failed to close database: %v", err)
		}
	}()

	if db.DB() == nil {
		t.Error("Expected database connection to be non-nil")
	}
}

func TestNewDatabaseWithFile(t *testing.T) {
	tmpFile := "./test-jobs.db"
	defer func() {
		if err := os.Remove(tmpFile); err != nil && !os.IsNotExist(err) {
			t.Fatalf("Failed to remove tmp file: %v", err)
		}
	}()

	config := &Config{
		Path: tmpFile,
	}

	db, err := NewDatabase(config)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatalf("Failed to close database: %v", err)
		}
	}()

	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Error("Expected database file to exist")
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Path != "./jobs.db" {
		t.Errorf("Expected default path ./jobs.db, got %s", config.Path)
	}
}

func TestInitSchema(t *testing.T) {
	config := &Config{
		Path: ":memory:",
	}

	db, err := NewDatabase(config)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatalf("Failed to close database: %v", err)
		}
	}()

	tables := []string{"jobs", "job_events"}
	for _, table := range tables {
		query := "SELECT name FROM sqlite_master WHERE type='table' AND name=?"
		var name string
		err := db.DB().QueryRowContext(context.Background(), query, table).Scan(&name)
		if err != nil {
			t.Errorf("Expected table %s to exist: %v", table, err)
		}
	}
}

func TestClose(t *testing.T) {
	config := &Config{
		Path: ":memory:",
	}

	db, err := NewDatabase(config)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	if err := db.Close(); err != nil {
		t.Errorf("Failed to close database: %v", err)
	}
}
