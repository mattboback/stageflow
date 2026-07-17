// Package db provides persistence for orchestrator jobs and events.
package db

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"sync"
	"time"

	// Import pgx stdlib driver for side effects.
	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed schema.sql
var schemaFS embed.FS

const (
	defaultMaxOpenConns = 20
	defaultMaxIdleConns = 5
	defaultConnLifetime = 30 * time.Minute
	defaultConnIdleTime = 5 * time.Minute
)

// Database wraps PostgreSQL database access.
type Database struct {
	db                 *sql.DB
	backgroundMu       sync.Mutex
	backgroundWG       sync.WaitGroup
	backgroundCancels  []context.CancelFunc
	backgroundStopping bool
}

// Config holds database configuration.
type Config struct {
	URL string
}

// NewDatabase creates a new database connection. The caller is responsible for
// supplying a non-empty URL; there is intentionally no default DSN because a
// hardcoded credential would be a deployment footgun.
func NewDatabase(config *Config) (*Database, error) {
	if config == nil || config.URL == "" {
		return nil, errors.New("database URL is required")
	}

	db, err := sql.Open("pgx", config.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(defaultMaxOpenConns)
	db.SetMaxIdleConns(defaultMaxIdleConns)
	db.SetConnMaxLifetime(defaultConnLifetime)
	db.SetConnMaxIdleTime(defaultConnIdleTime)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if pingErr := db.PingContext(ctx); pingErr != nil {
		_ = db.Close()

		return nil, fmt.Errorf("failed to connect database: %w", pingErr)
	}

	database := &Database{db: db}

	// Initialize schema
	if schemaErr := database.initSchema(); schemaErr != nil {
		_ = db.Close()

		return nil, fmt.Errorf("failed to initialize schema: %w", schemaErr)
	}

	return database, nil
}

// initSchema creates tables if they don't exist.
func (d *Database) initSchema() error {
	schema, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read schema: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, execErr := d.db.ExecContext(ctx, string(schema)); execErr != nil {
		return fmt.Errorf("failed to execute schema: %w", execErr)
	}

	return nil
}

func (d *Database) execContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return d.db.ExecContext(ctx, bindPostgresParams(query), args...)
}

func (d *Database) queryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return d.db.QueryContext(ctx, bindPostgresParams(query), args...)
}

func (d *Database) queryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return d.db.QueryRowContext(ctx, bindPostgresParams(query), args...)
}

func (d *Database) startBackgroundTask(parent context.Context, task func(context.Context)) error {
	if d == nil {
		return errors.New("database is nil")
	}

	if parent == nil {
		return errors.New("background task context is nil")
	}

	if task == nil {
		return errors.New("background task is nil")
	}

	d.backgroundMu.Lock()
	defer d.backgroundMu.Unlock()

	if d.backgroundStopping {
		return errors.New("database background tasks have stopped")
	}

	taskCtx, cancel := context.WithCancel(parent)
	d.backgroundCancels = append(d.backgroundCancels, cancel)
	d.backgroundWG.Add(1)

	go func() {
		defer d.backgroundWG.Done()
		defer cancel()

		task(taskCtx)
	}()

	return nil
}

// StopBackgroundTasks cancels database-owned maintenance and waits until it
// can no longer issue queries. It is safe to call more than once.
func (d *Database) StopBackgroundTasks() {
	if d == nil {
		return
	}

	d.backgroundMu.Lock()
	d.backgroundStopping = true
	cancels := append([]context.CancelFunc(nil), d.backgroundCancels...)
	d.backgroundMu.Unlock()

	for _, cancel := range cancels {
		cancel()
	}

	// backgroundStopping prevents Add calls after this Wait begins.
	d.backgroundWG.Wait()
}

// Close closes the database connection.
func (d *Database) Close() error {
	if d == nil {
		return nil
	}

	d.StopBackgroundTasks()

	if d.db != nil {
		return d.db.Close()
	}

	return nil
}
