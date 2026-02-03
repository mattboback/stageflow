package status

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
)

//go:embed schema.sql
var schemaFS embed.FS

const schemaVersion = 1

func sqliteHasTables(db *sql.DB) (bool, error) {
	query := "SELECT name FROM sqlite_master WHERE type='table' LIMIT 1"

	var name string

	err := db.QueryRowContext(context.Background(), query).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("check sqlite tables: %w", err)
	}

	return true, nil
}

func sqliteUserVersion(db *sql.DB) (int, error) {
	var version int
	if err := db.QueryRowContext(context.Background(), "PRAGMA user_version").Scan(&version); err != nil {
		return 0, fmt.Errorf("read sqlite user_version: %w", err)
	}

	return version, nil
}

func (s *Store) initSchema() error {
	hasTables, err := sqliteHasTables(s.db)
	if err != nil {
		return err
	}

	version, err := sqliteUserVersion(s.db)
	if err != nil {
		return err
	}

	if hasTables && version != schemaVersion {
		return fmt.Errorf(
			"platform API status DB schema mismatch (have=%d want=%d). Delete %s and restart",
			version,
			schemaVersion,
			s.path,
		)
	}

	schema, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("read schema: %w", err)
	}

	if _, err := s.db.ExecContext(context.Background(), string(schema)); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}

	if _, err := s.db.ExecContext(context.Background(), fmt.Sprintf("PRAGMA user_version=%d", schemaVersion)); err != nil {
		return fmt.Errorf("set sqlite user_version: %w", err)
	}

	return nil
}
