package status

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/mattboback/stageflow/services/platform-api/internal/sqlite"
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

	db, err := sqlite.Open(cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("open status store: %w", err)
	}

	store := &Store{db: db, path: cfg.Path}
	if schemaErr := store.initSchema(); schemaErr != nil {
		sqlite.Close(db)

		return nil, schemaErr
	}

	return store, nil
}

// Close shuts down the store.
func (s *Store) Close() error {
	return sqlite.Close(s.db)
}

func closeStatusDB(db *sql.DB) {
	if db == nil {
		return
	}

	if err := db.Close(); err != nil {
		slog.Error("Failed to close status DB", "error", err)
	}
}
