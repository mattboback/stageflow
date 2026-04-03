// Package sqlite provides shared SQLite initialization utilities.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	_ "github.com/mattn/go-sqlite3" // Register the SQLite driver.
)

// Open opens (or creates) a SQLite database at the given path with sensible defaults:
// WAL mode, busy_timeout=5000, foreign_keys=ON, and single-connection mode.
func Open(path string) (*sql.DB, error) {
	if path == "" {
		return nil, fmt.Errorf("sqlite: path is required")
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}

	ctx := context.Background()

	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA foreign_keys=ON",
	} {
		if _, pErr := db.ExecContext(ctx, pragma); pErr != nil {
			closeQuietly(db)
			return nil, fmt.Errorf("%s: %w", pragma, pErr)
		}
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	return db, nil
}

// Close logs an error if closing the database fails.
func Close(db *sql.DB) error {
	if db == nil {
		return nil
	}

	return db.Close()
}

func closeQuietly(db *sql.DB) {
	if db == nil {
		return
	}

	if err := db.Close(); err != nil {
		slog.Error("Failed to close sqlite db", "error", err)
	}
}
