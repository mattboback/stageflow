// Package db provides persistence for orchestrator jobs and events.
package db

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"

	// Import sqlite3 driver for side effects.
	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schemaFS embed.FS

// Database wraps SQLite database.
type Database struct {
	db   *sql.DB
	path string
}

// Config holds database configuration.
type Config struct {
	Path string
}

// DefaultConfig returns default database configuration.
func DefaultConfig() *Config {
	return &Config{
		Path: "./jobs.db",
	}
}

// NewDatabase creates a new database connection.
func NewDatabase(config *Config) (*Database, error) {
	if config == nil {
		config = DefaultConfig()
	}

	dsn := buildSQLiteDSN(config.Path)

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	ctx := context.Background()

	// Enable WAL mode for better concurrency

	if _, walErr := db.ExecContext(ctx, "PRAGMA journal_mode=WAL"); walErr != nil {
		_ = db.Close()

		return nil, fmt.Errorf("failed to enable WAL mode: %w", walErr)
	}

	// Allow concurrent writers to wait instead of failing with SQLITE_BUSY.
	if _, busyErr := db.ExecContext(ctx, "PRAGMA busy_timeout=5000"); busyErr != nil {
		_ = db.Close()

		return nil, fmt.Errorf("failed to set busy_timeout: %w", busyErr)
	}

	// Enable foreign keys
	if _, fkErr := db.ExecContext(ctx, "PRAGMA foreign_keys=ON"); fkErr != nil {
		_ = db.Close()

		return nil, fmt.Errorf("failed to enable foreign keys: %w", fkErr)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	database := &Database{db: db, path: config.Path}

	// Initialize schema
	if schemaErr := database.initSchema(); schemaErr != nil {
		_ = db.Close()

		return nil, fmt.Errorf("failed to initialize schema: %w", schemaErr)
	}

	return database, nil
}

const schemaVersion = 2

var memDBCounter uint64

func buildSQLiteDSN(path string) string {
	const pragmas = "_busy_timeout=5000&_journal_mode=WAL&_foreign_keys=ON"

	if path == "" {
		path = ":memory:"
	}

	if path == ":memory:" {
		// Use a unique name per NewDatabase call to avoid cross-test contamination.
		// Keep cache=shared so the sqlite driver can still use a single logical DB
		// when it opens multiple connections internally.
		n := atomic.AddUint64(&memDBCounter, 1)

		return fmt.Sprintf("file:stageflow_memdb_%d?mode=memory&cache=shared&%s", n, pragmas)
	}

	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}

	return path + separator + pragmas
}

func sqliteHasTables(db *sql.DB) (bool, error) {
	query := "SELECT name FROM sqlite_master WHERE type='table' LIMIT 1"

	var name string

	err := db.QueryRowContext(context.Background(), query).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("failed to check sqlite tables: %w", err)
	}

	return true, nil
}

func sqliteUserVersion(db *sql.DB) (int, error) {
	var version int
	if err := db.QueryRowContext(context.Background(), "PRAGMA user_version").Scan(&version); err != nil {
		return 0, fmt.Errorf("failed to read sqlite user_version: %w", err)
	}

	return version, nil
}

// initSchema creates tables if they don't exist.
func (d *Database) initSchema() error {
	hasTables, err := sqliteHasTables(d.db)
	if err != nil {
		return err
	}

	version, err := sqliteUserVersion(d.db)
	if err != nil {
		return err
	}

	schema, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read schema: %w", err)
	}

	ctx := context.Background()

	if !hasTables {
		if _, execErr := d.db.ExecContext(ctx, string(schema)); execErr != nil {
			return fmt.Errorf("failed to execute schema: %w", execErr)
		}

		if _, versionErr := d.db.ExecContext(
			ctx,
			fmt.Sprintf("PRAGMA user_version=%d", schemaVersion),
		); versionErr != nil {
			return fmt.Errorf("failed to set sqlite user_version: %w", versionErr)
		}

		return nil
	}

	// Treat version 0 as schemaVersion=1 for older DBs created before user_version was set.
	if version == 0 {
		version = 1
	}

	switch version {
	case schemaVersion:
		// Ensure indexes and any CREATE TABLE additions exist.
		if _, schemaExecErr := d.db.ExecContext(ctx, string(schema)); schemaExecErr != nil {
			return fmt.Errorf("failed to execute schema: %w", schemaExecErr)
		}

		return nil
	case 1:
		if migrateErr := d.migrate1To2(ctx); migrateErr != nil {
			return migrateErr
		}

		// Re-apply the latest schema to ensure new indexes exist.
		if _, schemaAfterMigrateErr := d.db.ExecContext(ctx, string(schema)); schemaAfterMigrateErr != nil {
			return fmt.Errorf("failed to execute schema after migration: %w", schemaAfterMigrateErr)
		}
	default:
		return fmt.Errorf(
			"orchestrator DB schema mismatch (have=%d want=%d). Delete %s and restart",
			version,
			schemaVersion,
			d.path,
		)
	}

	return nil
}

func (d *Database) migrate1To2(ctx context.Context) error {
	// v2 adds richer job_events columns for debugging/replay analysis.
	alterStmts := []string{
		"ALTER TABLE job_events ADD COLUMN request_id TEXT",
		"ALTER TABLE job_events ADD COLUMN run_id TEXT",
		"ALTER TABLE job_events ADD COLUMN producer TEXT",
		"ALTER TABLE job_events ADD COLUMN nats_subject TEXT",
		"ALTER TABLE job_events ADD COLUMN nats_stream TEXT",
		"ALTER TABLE job_events ADD COLUMN nats_consumer TEXT",
		"ALTER TABLE job_events ADD COLUMN nats_stream_seq INTEGER",
		"ALTER TABLE job_events ADD COLUMN nats_consumer_seq INTEGER",
		"ALTER TABLE job_events ADD COLUMN nats_deliveries INTEGER",
		"ALTER TABLE job_events ADD COLUMN nats_stored_at TIMESTAMP",
		"ALTER TABLE job_events ADD COLUMN handler_status TEXT",
		"ALTER TABLE job_events ADD COLUMN handler_error TEXT",
		"ALTER TABLE job_events ADD COLUMN duration_ms INTEGER",
	}

	for _, stmt := range alterStmts {
		if _, err := d.db.ExecContext(ctx, stmt); err != nil {
			// SQLite will error if the column already exists; make migrations idempotent.
			if strings.Contains(err.Error(), "duplicate column name") {
				continue
			}

			return fmt.Errorf("migrate v1->v2: %s: %w", stmt, err)
		}
	}

	if _, err := d.db.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version=%d", schemaVersion)); err != nil {
		return fmt.Errorf("failed to set sqlite user_version: %w", err)
	}

	return nil
}

// Close closes the database connection.
func (d *Database) Close() error {
	if d.db != nil {
		return d.db.Close()
	}

	return nil
}

// DB returns the underlying sql.DB for direct access if needed.
func (d *Database) DB() *sql.DB {
	return d.db
}
