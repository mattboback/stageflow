// Package tombstone persists deleted job IDs until the job store owns deletion.
package tombstone

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/mattboback/stageflow/services/platform-api/internal/sqlite"
)

type Store struct{ db *sql.DB }

// NewStore opens the existing projects.db without replacing its deletion history.
func NewStore(path string) (*Store, error) {
	db, err := sqlite.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open tombstone store: %w", err)
	}

	_, err = db.ExecContext(context.Background(), `CREATE TABLE IF NOT EXISTS deleted_jobs (
  job_id TEXT PRIMARY KEY,
  deleted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
 )`)
	if err != nil {
		sqlite.Close(db)
		return nil, fmt.Errorf("initialize tombstone store: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error { return sqlite.Close(s.db) }

func (s *Store) TombstoneJob(ctx context.Context, jobID string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO deleted_jobs (job_id, deleted_at) VALUES (?, ?)
 ON CONFLICT(job_id) DO NOTHING`, jobID, time.Now().UTC())

	return err
}

func (s *Store) JobIsDeleted(ctx context.Context, jobID string) (bool, error) {
	var count int

	err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM deleted_jobs WHERE job_id = ?`, jobID).Scan(&count)

	return count > 0, err
}
