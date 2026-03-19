package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mattboback/stageflow/libs/go/models"
)

const jobSelectColumns = `
id, state, input_type, input_path, urls, pod_id, config_json, created_at, updated_at, completed_at, error, error_details, last_stage, total_pages, current_page, total_violations, report_json_key, report_key, scan_stage_log_key, scan_recipe_key, extraction_stage_log_key, extraction_recipe_key, provenance_path, provenance_key, expected_scanners, completed_scanners, scanner_results
`

type rowScanner interface {
	Scan(dest ...any) error
}

// CreateJob creates a new job in the database.
func (d *Database) CreateJob(ctx context.Context, job *models.Job) error {
	configJSON, err := json.Marshal(job.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	urlsJSON, err := json.Marshal(job.URLs)
	if err != nil {
		return fmt.Errorf("failed to marshal urls: %w", err)
	}

	query := `
		INSERT INTO jobs (id, state, input_type, input_path, urls, config_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = d.execContext(ctx, query,
		job.ID,
		job.State,
		job.InputType,
		job.InputPath,
		string(urlsJSON),
		string(configJSON),
		job.CreatedAt,
		job.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}

	return nil
}

// CreateJobIfAbsent inserts a new job only when it does not already exist.
func (d *Database) CreateJobIfAbsent(ctx context.Context, job *models.Job) (bool, error) {
	configJSON, err := json.Marshal(job.Config)
	if err != nil {
		return false, fmt.Errorf("failed to marshal config: %w", err)
	}

	urlsJSON, err := json.Marshal(job.URLs)
	if err != nil {
		return false, fmt.Errorf("failed to marshal urls: %w", err)
	}

	query := `
		INSERT INTO jobs (id, state, input_type, input_path, urls, config_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO NOTHING
	`

	result, err := d.execContext(ctx, query,
		job.ID,
		job.State,
		job.InputType,
		job.InputPath,
		string(urlsJSON),
		string(configJSON),
		job.CreatedAt,
		job.UpdatedAt,
	)
	if err != nil {
		return false, fmt.Errorf("failed to create job: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to read rows affected: %w", err)
	}

	return rows > 0, nil
}

//nolint:gocognit,gocyclo // Scanning many nullable columns requires multiple nil checks
func scanJobRow(s rowScanner) (*models.Job, error) {
	var (
		job                   models.Job
		podID                 sql.NullString
		urlsJSON              sql.NullString
		configJSON            string
		completedAt           sql.NullTime
		errorStr              sql.NullString
		errorDetails          sql.NullString
		lastStage             sql.NullString
		totalPages            sql.NullInt64
		currentPage           sql.NullInt64
		totalViolations       sql.NullInt64
		reportJSONKey         sql.NullString
		reportKey             sql.NullString
		scanStageLogKey       sql.NullString
		scanRecipeKey         sql.NullString
		extractionStageLogKey sql.NullString
		extractionRecipeKey   sql.NullString
		provenancePath        sql.NullString
		provenanceKey         sql.NullString
		expectedScannersJSON  sql.NullString
		completedScannersJSON sql.NullString
		scannerResultsJSON    sql.NullString
	)

	if err := s.Scan(
		&job.ID,
		&job.State,
		&job.InputType,
		&job.InputPath,
		&urlsJSON,
		&podID,
		&configJSON,
		&job.CreatedAt,
		&job.UpdatedAt,
		&completedAt,
		&errorStr,
		&errorDetails,
		&lastStage,
		&totalPages,
		&currentPage,
		&totalViolations,
		&reportJSONKey,
		&reportKey,
		&scanStageLogKey,
		&scanRecipeKey,
		&extractionStageLogKey,
		&extractionRecipeKey,
		&provenancePath,
		&provenanceKey,
		&expectedScannersJSON,
		&completedScannersJSON,
		&scannerResultsJSON,
	); err != nil {
		return nil, err
	}

	if podID.Valid {
		job.PodID = podID.String
	}

	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}

	if errorStr.Valid {
		job.Error = errorStr.String
	}

	if errorDetails.Valid {
		job.ErrorDetails = errorDetails.String
	}

	if lastStage.Valid {
		job.LastStage = lastStage.String
	}

	if totalPages.Valid {
		job.TotalPages = int(totalPages.Int64)
	}

	if currentPage.Valid {
		job.CurrentPage = int(currentPage.Int64)
	}

	if totalViolations.Valid {
		job.TotalViolations = int(totalViolations.Int64)
	}

	if reportJSONKey.Valid {
		job.ReportJSONKey = reportJSONKey.String
	}

	if reportKey.Valid {
		job.ReportKey = reportKey.String
	}

	if scanStageLogKey.Valid {
		job.ScanStageLogKey = scanStageLogKey.String
	}

	if scanRecipeKey.Valid {
		job.ScanRecipeKey = scanRecipeKey.String
	}

	if extractionStageLogKey.Valid {
		job.ExtractionStageLogKey = extractionStageLogKey.String
	}

	if extractionRecipeKey.Valid {
		job.ExtractionRecipeKey = extractionRecipeKey.String
	}

	if provenancePath.Valid {
		job.ProvenancePath = provenancePath.String
	}

	if provenanceKey.Valid {
		job.ProvenanceKey = provenanceKey.String
	}

	if err := json.Unmarshal([]byte(configJSON), &job.Config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if urlsJSON.Valid && urlsJSON.String != "" {
		if err := json.Unmarshal([]byte(urlsJSON.String), &job.URLs); err != nil {
			return nil, fmt.Errorf("failed to unmarshal urls: %w", err)
		}
	}

	if expectedScannersJSON.Valid && expectedScannersJSON.String != "" {
		if err := json.Unmarshal([]byte(expectedScannersJSON.String), &job.ExpectedScanners); err != nil {
			return nil, fmt.Errorf("failed to unmarshal expected_scanners: %w", err)
		}
	}

	if completedScannersJSON.Valid && completedScannersJSON.String != "" {
		if err := json.Unmarshal([]byte(completedScannersJSON.String), &job.CompletedScanners); err != nil {
			return nil, fmt.Errorf("failed to unmarshal completed_scanners: %w", err)
		}
	}

	if scannerResultsJSON.Valid && scannerResultsJSON.String != "" {
		if err := json.Unmarshal([]byte(scannerResultsJSON.String), &job.ScannerResults); err != nil {
			return nil, fmt.Errorf("failed to unmarshal scanner_results: %w", err)
		}
	}

	return &job, nil
}

// GetJob retrieves a job by ID.
func (d *Database) GetJob(ctx context.Context, jobID string) (*models.Job, error) {
	query := fmt.Sprintf(`
		SELECT %s
		FROM jobs
		WHERE id = ?
	`, jobSelectColumns)

	row := d.queryRowContext(ctx, query, jobID)

	job, err := scanJobRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	return job, nil
}

// ListJobsByState lists all jobs in a specific state.
func (d *Database) ListJobsByState(ctx context.Context, state models.JobState) ([]*models.Job, error) {
	query := fmt.Sprintf(`
		SELECT %s
		FROM jobs
		WHERE state = ?
		ORDER BY created_at ASC
	`, jobSelectColumns)

	rows, err := d.queryContext(ctx, query, state)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}

	defer func() {
		_ = rows.Close()
	}()

	return d.scanJobs(rows)
}

// ListJobsOptions contains options for listing jobs.
type ListJobsOptions struct {
	State  *models.JobState // Optional filter by state
	Limit  int              // Max number of results (0 = no limit)
	Offset int              // Number of results to skip
}

// ListJobs retrieves jobs with optional filtering and pagination.
func (d *Database) ListJobs(ctx context.Context, opts ListJobsOptions) ([]*models.Job, error) {
	query := fmt.Sprintf(`
		SELECT %s
			FROM jobs
	`, jobSelectColumns)

	var args []any

	// Add state filter if provided
	if opts.State != nil {
		query += " WHERE state = ?"

		args = append(args, *opts.State)
	}

	// Order by created_at descending (newest first)
	query += " ORDER BY created_at DESC"

	// Add pagination
	if opts.Limit > 0 {
		query += " LIMIT ?"

		args = append(args, opts.Limit)
	}

	if opts.Offset > 0 {
		query += " OFFSET ?"

		args = append(args, opts.Offset)
	}

	rows, err := d.queryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}

	defer func() {
		_ = rows.Close()
	}()

	return d.scanJobs(rows)
}

func (d *Database) scanJobs(rows *sql.Rows) ([]*models.Job, error) {
	var jobs []*models.Job

	for rows.Next() {
		job, err := scanJobRow(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job: %w", err)
		}

		jobs = append(jobs, job)
	}

	return jobs, nil
}

// CountJobs counts total jobs, optionally filtered by state.
func (d *Database) CountJobs(ctx context.Context, state *models.JobState) (int, error) {
	query := "SELECT COUNT(*) FROM jobs"

	var args []any

	if state != nil {
		query += " WHERE state = ?"

		args = append(args, *state)
	}

	var count int

	err := d.queryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count jobs: %w", err)
	}

	return count, nil
}
