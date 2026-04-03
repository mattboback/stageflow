package project

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/mattboback/stageflow/services/platform-api/internal/sqlite"
)

//go:embed schema.sql
var schemaFS embed.FS

const schemaVersion = 1

var ErrNotFound = errors.New("project not found")

type Store struct {
	db *sql.DB
}

func NewStore(path string) (*Store, error) {
	if path == "" {
		return nil, errors.New("project store path is required")
	}

	db, err := sqlite.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open project store: %w", err)
	}

	s := &Store{db: db}

	err = s.initSchema()
	if err != nil {
		sqlite.Close(db)

		return nil, err
	}

	return s, nil
}

func (s *Store) Close() error {
	return sqlite.Close(s.db)
}

func (s *Store) initSchema() error {
	schema, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("read schema: %w", err)
	}

	ctx := context.Background()

	if _, execErr := s.db.ExecContext(ctx, string(schema)); execErr != nil {
		return fmt.Errorf("apply schema: %w", execErr)
	}

	if _, vErr := s.db.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version=%d", schemaVersion)); vErr != nil {
		return fmt.Errorf("set user_version: %w", vErr)
	}

	return nil
}

func (s *Store) CreateProject(ctx context.Context, p *Project) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}

	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now

	urlsJSON, err := json.Marshal(p.URLs)
	if err != nil {
		return fmt.Errorf("marshal urls: %w", err)
	}

	var scannersJSON []byte
	if len(p.Scanners) > 0 {
		scannersJSON, err = json.Marshal(p.Scanners)
		if err != nil {
			return fmt.Errorf("marshal scanners: %w", err)
		}
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO projects (id, slug, name, urls, scanners, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.Slug, p.Name, string(urlsJSON), nullString(scannersJSON),
		p.CreatedAt, p.UpdatedAt,
	)

	return err
}

func (s *Store) GetProjectBySlug(ctx context.Context, slug string) (*Project, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, slug, name, urls, scanners, baseline_job_id, created_at, updated_at
		 FROM projects WHERE slug = ?`, slug)

	return scanProject(row)
}

func (s *Store) GetProjectByID(ctx context.Context, id string) (*Project, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, slug, name, urls, scanners, baseline_job_id, created_at, updated_at
		 FROM projects WHERE id = ?`, id)

	return scanProject(row)
}

func (s *Store) ListProjects(ctx context.Context) ([]Project, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, slug, name, urls, scanners, baseline_job_id, created_at, updated_at
		 FROM projects ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []Project

	for rows.Next() {
		p, rowErr := scanProjectRow(rows)
		if rowErr != nil {
			return nil, rowErr
		}

		projects = append(projects, *p)
	}

	return projects, rows.Err()
}

func (s *Store) UpdateProject(ctx context.Context, id string, u Update) error {
	now := time.Now().UTC()

	if u.Name != nil {
		if _, err := s.db.ExecContext(ctx,
			`UPDATE projects SET name = ?, updated_at = ? WHERE id = ?`,
			*u.Name, now, id,
		); err != nil {
			return err
		}
	}

	if u.URLs != nil {
		urlsJSON, err := json.Marshal(u.URLs)
		if err != nil {
			return fmt.Errorf("marshal urls: %w", err)
		}

		_, err = s.db.ExecContext(ctx,
			`UPDATE projects SET urls = ?, updated_at = ? WHERE id = ?`,
			string(urlsJSON), now, id,
		)
		if err != nil {
			return err
		}
	}

	if u.Scanners != nil {
		var scannersJSON []byte

		if len(u.Scanners) > 0 {
			marshaledScanners, err := json.Marshal(u.Scanners)
			if err != nil {
				return fmt.Errorf("marshal scanners: %w", err)
			}

			scannersJSON = marshaledScanners
		}

		_, execErr := s.db.ExecContext(ctx,
			`UPDATE projects SET scanners = ?, updated_at = ? WHERE id = ?`,
			nullString(scannersJSON), now, id,
		)
		if execErr != nil {
			return execErr
		}
	}

	return nil
}

func (s *Store) DeleteProject(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM projects WHERE id = ?`, id)
	if err != nil {
		return err
	}

	n, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if n == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *Store) RecordProjectJob(ctx context.Context, projectID, jobID string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO project_jobs (project_id, job_id) VALUES (?, ?)`,
		projectID, jobID,
	)

	return err
}

func (s *Store) GetProjectForJob(ctx context.Context, jobID string) (*Project, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT p.id, p.slug, p.name, p.urls, p.scanners, p.baseline_job_id, p.created_at, p.updated_at
		 FROM projects p
		 JOIN project_jobs pj ON pj.project_id = p.id
		 WHERE pj.job_id = ?`, jobID)

	return scanProject(row)
}

func (s *Store) SetBaseline(ctx context.Context, projectID, jobID string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE projects SET baseline_job_id = ?, updated_at = ? WHERE id = ?`,
		jobID, time.Now().UTC(), projectID,
	)

	return err
}

// JobBelongsToProject checks whether a given job is associated with a project.
func (s *Store) JobBelongsToProject(ctx context.Context, projectID, jobID string) (bool, error) {
	var count int

	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM project_jobs WHERE project_id = ? AND job_id = ?`,
		projectID, jobID,
	).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanProject(row scanner) (*Project, error) {
	var (
		p             Project
		urlsJSON      string
		scannersJSON  sql.NullString
		baselineJobID sql.NullString
	)

	err := row.Scan(
		&p.ID, &p.Slug, &p.Name, &urlsJSON, &scannersJSON, &baselineJobID,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}

	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(urlsJSON), &p.URLs)
	if err != nil {
		return nil, fmt.Errorf("unmarshal urls: %w", err)
	}

	if scannersJSON.Valid {
		err = json.Unmarshal([]byte(scannersJSON.String), &p.Scanners)
		if err != nil {
			return nil, fmt.Errorf("unmarshal scanners: %w", err)
		}
	}

	if baselineJobID.Valid {
		p.BaselineJobID = baselineJobID.String
	}

	return &p, nil
}

func scanProjectRow(rows *sql.Rows) (*Project, error) {
	return scanProject(rows)
}

func nullString(b []byte) sql.NullString {
	if b == nil {
		return sql.NullString{}
	}

	return sql.NullString{String: string(b), Valid: true}
}
