package status

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	// Import sqlite3 driver for its side effects.
	_ "github.com/mattn/go-sqlite3"
)

// ErrJobNotFound is returned when a job is missing from the projection store.
var ErrJobNotFound = errors.New("job not found")

// Config holds store configuration.
type Config struct {
	Path string
}

// Store persists the platform API job projection for fast lookups.
type Store struct {
	db   *sql.DB
	path string
}

// NewStore opens (or creates) the projection store at the given path.
func NewStore(cfg *Config) (*Store, error) {
	if cfg == nil || cfg.Path == "" {
		return nil, errors.New("status store path is required")
	}

	db, err := sql.Open("sqlite3", cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("open status store: %w", err)
	}

	ctx := context.Background()
	if _, walErr := db.ExecContext(ctx, "PRAGMA journal_mode=WAL"); walErr != nil {
		closeStatusDB(db)

		return nil, fmt.Errorf("enable WAL: %w", walErr)
	}

	// Allow concurrent writers to wait instead of failing with SQLITE_BUSY.
	if _, busyErr := db.ExecContext(ctx, "PRAGMA busy_timeout=5000"); busyErr != nil {
		closeStatusDB(db)

		return nil, fmt.Errorf("set busy_timeout: %w", busyErr)
	}

	if _, fkErr := db.ExecContext(ctx, "PRAGMA foreign_keys=ON"); fkErr != nil {
		closeStatusDB(db)

		return nil, fmt.Errorf("enable foreign keys: %w", fkErr)
	}

	// Best practice for SQLite with database/sql: one connection avoids lock contention and
	// subtle concurrency issues (WAL still allows concurrent readers).
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	store := &Store{db: db, path: cfg.Path}
	if schemaErr := store.initSchema(); schemaErr != nil {
		closeStatusDB(db)

		return nil, schemaErr
	}

	return store, nil
}

// Close shuts down the store.
func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}

	return nil
}

func closeStatusDB(db *sql.DB) {
	if db == nil {
		return
	}

	if err := db.Close(); err != nil {
		slog.Error("Failed to close status DB", "error", err)
	}
}
