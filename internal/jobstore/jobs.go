package jobstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mattboback/stageflow/libs/go/models"
)

const QueueCapacity = 5
const admissionLock = 1937006950

var ErrAttemptsExhausted = errors.New("scanner launch attempts exhausted")

const jobColumns = `id,state,input_type,input_path,urls,config_json,pod_id,created_at,updated_at,claimed_at,extraction_started_at,scan_started_at,completed_at,cleanup_pending,error,error_details,last_stage,total_pages,total_violations,report_json_key`

type rowScanner interface{ Scan(...any) error }

func (d *Database) checkSchema(ctx context.Context) error {
	var valid bool

	err := d.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema=current_schema() AND table_name='jobs' AND column_name='claimed_at') AND to_regclass('job_scanners') IS NOT NULL`).
		Scan(&valid)
	if err != nil {
		return err
	}

	if !valid {
		return errors.New("PostgreSQL schema v2 is missing; initialize a fresh database volume with the orchestrator")
	}

	return nil
}
func decodeJob(row rowScanner) (*models.Job, error) {
	j := &models.Job{}

	var urls, config []byte
	if err := row.Scan(
		&j.ID,
		&j.State,
		&j.InputType,
		&j.InputPath,
		&urls,
		&config,
		&j.PodID,
		&j.CreatedAt,
		&j.UpdatedAt,
		&j.ClaimedAt,
		&j.ExtractionStartedAt,
		&j.ScanStartedAt,
		&j.CompletedAt,
		&j.CleanupPending,
		&j.Error,
		&j.ErrorDetails,
		&j.LastStage,
		&j.TotalPages,
		&j.TotalViolations,
		&j.ReportJSONKey,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrJobNotFound
		}

		return nil, err
	}

	if err := json.Unmarshal(urls, &j.URLs); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(config, &j.Config); err != nil {
		return nil, err
	}

	j.ScannerResults = map[string]*models.ScannerResult{}
	j.ScannerProgress = map[string]int{}
	j.ScannerStates = map[string]string{}
	j.ScannerAttempts = map[string]int{}
	j.ScannerContainerIDs = map[string]string{}
	j.ProvenanceKey = j.ID + "/provenance.json"

	return j, nil
}
func loadScanners(ctx context.Context, tx *sql.Tx, j *models.Job) error {
	rows, err := tx.QueryContext(
		ctx,
		`SELECT scanner,state,attempts,container_id,pages_scanned,total_issues,critical_issues,serious_issues,moderate_issues,minor_issues,timing_ms,error,results_path FROM job_scanners WHERE job_id=$1 ORDER BY scanner`,
		j.ID,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			name, state, container string
			attempts               int
		)

		r := &models.ScannerResult{}
		if err = rows.Scan(
			&name,
			&state,
			&attempts,
			&container,
			&r.PagesScanned,
			&r.TotalIssues,
			&r.CriticalIssues,
			&r.SeriousIssues,
			&r.ModerateIssues,
			&r.MinorIssues,
			&r.TotalMs,
			&r.Error,
			&r.ResultsPath,
		); err != nil {
			return err
		}

		r.ScannerType = name
		r.Success = state == "success" || state == "partial"

		j.ExpectedScanners = append(j.ExpectedScanners, name)
		j.ScannerStates[name] = state
		j.ScannerAttempts[name] = attempts
		j.ScannerContainerIDs[name] = container

		j.ScannerProgress[name] = r.PagesScanned
		if r.PagesScanned > j.CurrentPage {
			j.CurrentPage = r.PagesScanned
		}

		if state == "success" || state == "partial" || state == "failed" {
			j.ScannerResults[name] = r
		}

		if state == "success" || state == "partial" || state == "failed" {
			j.CompletedScanners = append(j.CompletedScanners, name)
		}
	}

	return rows.Err()
}
func getJobTx(ctx context.Context, tx *sql.Tx, id string) (*models.Job, error) {
	j, err := decodeJob(tx.QueryRowContext(ctx, `SELECT `+jobColumns+` FROM jobs WHERE id=$1`, id))
	if err != nil {
		return nil, err
	}

	if err = loadScanners(ctx, tx, j); err != nil {
		return nil, err
	}

	return j, nil
}
func (d *Database) GetJob(ctx context.Context, id string) (*models.Job, error) {
	tx, err := d.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer rollback(tx)

	j, err := getJobTx(ctx, tx, id)
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return j, nil
}

// listJobs receives only fixed SQL predicates and internally generated placeholders.
//
//nolint:gosec // All values use bound parameters; the predicate is never user input.
func (d *Database) listJobs(ctx context.Context, where string, args ...any) ([]*models.Job, error) {
	tx, err := d.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer rollback(tx)

	rows, err := tx.QueryContext(ctx, `SELECT `+jobColumns+` FROM jobs `+where+` ORDER BY created_at,id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*models.Job

	for rows.Next() {
		j, e := decodeJob(rows)
		if e != nil {
			rows.Close()
			return nil, e
		}

		jobs = append(jobs, j)
	}

	err = rows.Err()
	rows.Close()

	if err != nil {
		return nil, err
	}

	for _, j := range jobs {
		if err = loadScanners(ctx, tx, j); err != nil {
			return nil, err
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return jobs, nil
}
func (d *Database) ListJobsByState(ctx context.Context, states ...models.JobState) ([]*models.Job, error) {
	if len(states) == 0 {
		return d.listJobs(ctx, "")
	}

	parts := make([]string, len(states))

	args := make([]any, len(states))
	for i, state := range states {
		parts[i] = fmt.Sprintf("$%d", i+1)
		args[i] = state
	}

	return d.listJobs(ctx, `WHERE state IN (`+strings.Join(parts, ",")+`)`, args...)
}
func (d *Database) TerminalJobsBefore(ctx context.Context, before time.Time) ([]*models.Job, error) {
	return d.listJobs(ctx, `WHERE state IN ('DONE','FAILED') AND NOT cleanup_pending AND completed_at<$1`, before)
}
func (d *Database) JobExists(ctx context.Context, id string) (bool, error) {
	var exists bool

	err := d.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM jobs WHERE id=$1)`, id).Scan(&exists)

	return exists, err
}
func (d *Database) DeleteJob(ctx context.Context, id string) error {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollback(tx)

	if err = lockAdmission(ctx, tx); err != nil {
		return err
	}

	var (
		state   models.JobState
		claimed *time.Time
		cleanup bool
	)

	err = tx.QueryRowContext(ctx, `SELECT state,claimed_at,cleanup_pending FROM jobs WHERE id=$1 FOR UPDATE`, id).
		Scan(&state, &claimed, &cleanup)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}

	if err != nil {
		return err
	}

	if cleanup {
		return ErrCleanupPending
	}

	if claimed != nil && !state.IsTerminal() {
		return ErrJobBusy
	}

	if _, err = tx.ExecContext(ctx, `DELETE FROM jobs WHERE id=$1`, id); err != nil {
		return err
	}

	return tx.Commit()
}

func lockAdmission(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1)`, admissionLock)
	return err
}
func insertJob(ctx context.Context, tx *sql.Tx, j *models.Job) error {
	if j.ID == "" || !j.State.IsValid() {
		return errors.New("invalid job identity or state")
	}

	if j.CreatedAt.IsZero() {
		j.CreatedAt = time.Now().UTC()
	}

	if j.UpdatedAt.IsZero() {
		j.UpdatedAt = j.CreatedAt
	}

	config, err := json.Marshal(j.Config)
	if err != nil {
		return err
	}

	urls, err := json.Marshal(j.URLs)
	if err != nil {
		return err
	}

	if j.URLs == nil {
		urls = []byte("[]")
	}

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO jobs(id,state,input_type,input_path,urls,config_json,pod_id,created_at,updated_at,claimed_at,extraction_started_at,scan_started_at,completed_at,cleanup_pending,total_pages) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		j.ID,
		j.State,
		j.InputType,
		j.InputPath,
		string(urls),
		string(config),
		j.PodID,
		j.CreatedAt,
		j.UpdatedAt,
		j.ClaimedAt,
		j.ExtractionStartedAt,
		j.ScanStartedAt,
		j.CompletedAt,
		j.CleanupPending,
		j.TotalPages,
	)
	if err != nil {
		return err
	}

	names := j.ExpectedScanners
	if len(names) == 0 {
		names = j.Config.Modules
	}

	for _, name := range names {
		if name == "" {
			return errors.New("empty scanner name")
		}

		if _, err = tx.ExecContext(
			ctx,
			`INSERT INTO job_scanners(job_id,scanner) VALUES($1,$2) ON CONFLICT DO NOTHING`,
			j.ID,
			name,
		); err != nil {
			return err
		}
	}

	return nil
}
func (d *Database) AdmitJob(ctx context.Context, j *models.Job) error {
	if j.State == "" {
		j.State = models.JobStatePending
	}

	if j.State != models.JobStatePending || j.ClaimedAt != nil {
		return errors.New("admitted jobs must be pending and unclaimed")
	}

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer rollback(tx)

	if err = lockAdmission(ctx, tx); err != nil {
		return err
	}

	var queued int
	if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM jobs WHERE state='PENDING' AND claimed_at IS NULL`).
		Scan(&queued); err != nil {
		return err
	}

	if queued >= QueueCapacity {
		return ErrQueueFull
	}

	if err = insertJob(ctx, tx, j); err != nil {
		return err
	}

	return tx.Commit()
}

// jobsAhead counts the jobs that start before an unclaimed PENDING job: the
// running job, if any, and every unclaimed PENDING job ClaimNext takes first.
func (d *Database) jobsAhead(ctx context.Context, id string) (int, error) {
	var ahead int

	err := d.db.QueryRowContext(ctx, `SELECT count(*) FROM jobs j, (SELECT created_at, id FROM jobs WHERE id=$1) me
		WHERE j.id <> me.id AND (
			(j.state='PENDING' AND j.claimed_at IS NULL AND (j.created_at, j.id) < (me.created_at, me.id))
			OR (j.claimed_at IS NOT NULL AND j.state NOT IN ('DONE','FAILED')))`, id).Scan(&ahead)

	return ahead, err
}

//nolint:nilnil // A nil job means the durable queue has no claim available.
func (d *Database) ClaimNext(ctx context.Context) (*models.Job, error) {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer rollback(tx)

	if err = lockAdmission(ctx, tx); err != nil {
		return nil, err
	}

	var active int
	if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM jobs WHERE (claimed_at IS NOT NULL AND state NOT IN ('DONE','FAILED')) OR cleanup_pending`).
		Scan(&active); err != nil {
		return nil, err
	}

	if active > 0 {
		return nil, nil
	}

	var id string

	err = tx.QueryRowContext(ctx, `UPDATE jobs SET claimed_at=CURRENT_TIMESTAMP,updated_at=CURRENT_TIMESTAMP WHERE id=(SELECT id FROM jobs WHERE state='PENDING' AND claimed_at IS NULL ORDER BY created_at,id LIMIT 1 FOR UPDATE SKIP LOCKED) AND state='PENDING' AND claimed_at IS NULL RETURNING id`).
		Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	j, err := getJobTx(ctx, tx, id)
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return j, nil
}
func changed(result sql.Result, err error) error {
	if err != nil {
		return err
	}

	n, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if n == 0 {
		return ErrJobNotFound
	}

	return nil
}
func (d *Database) SetPodID(ctx context.Context, id, pod string) error {
	return changed(
		d.db.ExecContext(
			ctx,
			`UPDATE jobs SET pod_id=$2,updated_at=CURRENT_TIMESTAMP WHERE id=$1 AND state NOT IN ('DONE','FAILED')`,
			id,
			pod,
		),
	)
}
func (d *Database) BeginStage(ctx context.Context, id string, state models.JobState) error {
	if !state.IsValid() || state.IsTerminal() {
		return errors.New("invalid active stage")
	}

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer rollback(tx)

	var previous models.JobState
	if err = tx.QueryRowContext(ctx, `SELECT state FROM jobs WHERE id=$1 FOR UPDATE`, id).
		Scan(&previous); errors.Is(
		err,
		sql.ErrNoRows,
	) {
		return ErrJobNotFound
	} else if err != nil {
		return err
	}

	if previous != state && !canTransition(previous, state) {
		return fmt.Errorf("invalid job transition %s -> %s", previous, state)
	}

	query := `UPDATE jobs SET state=$2,updated_at=CURRENT_TIMESTAMP`
	if state == models.JobStateExtracting {
		query += `,extraction_started_at=COALESCE(extraction_started_at,CURRENT_TIMESTAMP)`
	}

	if state == models.JobStateScanning {
		query += `,scan_started_at=COALESCE(scan_started_at,CURRENT_TIMESTAMP)`
	}

	query += ` WHERE id=$1 AND ` + stateRankSQL + ` <= $3 AND state NOT IN ('DONE','FAILED')`
	if err = changed(tx.ExecContext(ctx, query, id, state, stateRank(state))); err != nil {
		return err
	}

	return tx.Commit()
}
func lockActiveJob(ctx context.Context, tx *sql.Tx, id string) error {
	var state models.JobState
	if err := tx.QueryRowContext(ctx, `SELECT state FROM jobs WHERE id=$1 FOR UPDATE`, id).
		Scan(&state); errors.Is(
		err,
		sql.ErrNoRows,
	) {
		return ErrJobNotFound
	} else if err != nil {
		return err
	}

	if state.IsTerminal() {
		return ErrJobNotFound
	}

	return nil
}
func (d *Database) BeginScannerAttempt(ctx context.Context, id, name string) (int, error) {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer rollback(tx)

	if err = lockActiveJob(ctx, tx, id); err != nil {
		return 0, err
	}

	var attempts int

	err = tx.QueryRowContext(ctx, `UPDATE job_scanners SET attempts=attempts+1,state='running' WHERE job_id=$1 AND scanner=$2 AND attempts<2 AND state IN ('pending','running') RETURNING attempts`, id, name).
		Scan(&attempts)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrAttemptsExhausted
	}

	if err != nil {
		return 0, err
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}

	return attempts, nil
}
func (d *Database) SetScannerContainer(ctx context.Context, id, name, container string) error {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollback(tx)

	if err = lockActiveJob(ctx, tx, id); err != nil {
		return err
	}

	if err = changed(
		tx.ExecContext(
			ctx,
			`UPDATE job_scanners SET container_id=$3 WHERE job_id=$1 AND scanner=$2 AND state IN ('pending','running')`,
			id,
			name,
			container,
		),
	); err != nil {
		return err
	}

	return tx.Commit()
}
func (d *Database) RecordScannerResult(ctx context.Context, id, status string, r *models.ScannerResult) error {
	if r == nil || r.ScannerType == "" || (status != "success" && status != "partial" && status != "failed") {
		return errors.New("invalid scanner outcome")
	}

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer rollback(tx)

	if err = lockActiveJob(ctx, tx, id); err != nil {
		return err
	}

	if err = changed(
		tx.ExecContext(
			ctx,
			`UPDATE job_scanners SET state=$3,pages_scanned=GREATEST(pages_scanned,$4),total_issues=$5,critical_issues=$6,serious_issues=$7,moderate_issues=$8,minor_issues=$9,info_issues=GREATEST(0,$5::int-($6::int+$7::int+$8::int+$9::int)),timing_ms=$10,error=$11,results_path=$12 WHERE job_id=$1 AND scanner=$2 AND state IN ('pending','running')`,
			id,
			r.ScannerType,
			status,
			r.PagesScanned,
			r.TotalIssues,
			r.CriticalIssues,
			r.SeriousIssues,
			r.ModerateIssues,
			r.MinorIssues,
			r.TotalMs,
			r.Error,
			r.ResultsPath,
		),
	); err != nil {
		return err
	}

	if _, err = tx.ExecContext(ctx, `UPDATE jobs SET updated_at=CURRENT_TIMESTAMP WHERE id=$1`, id); err != nil {
		return err
	}

	return tx.Commit()
}

func (d *Database) UpdateScannerProgress(ctx context.Context, id, name string, pages, total int) error {
	if pages < 0 || total < 0 {
		return errors.New("invalid progress")
	}

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer rollback(tx)

	if err = changed(
		tx.ExecContext(
			ctx,
			`UPDATE jobs SET total_pages=GREATEST(total_pages,$2),updated_at=CURRENT_TIMESTAMP WHERE id=$1 AND state NOT IN ('DONE','FAILED')`,
			id,
			total,
		),
	); err != nil {
		return err
	}

	if err = changed(
		tx.ExecContext(
			ctx,
			`UPDATE job_scanners SET pages_scanned=GREATEST(pages_scanned,$3) WHERE job_id=$1 AND scanner=$2 AND state IN ('pending','running')`,
			id,
			name,
			pages,
		),
	); err != nil {
		return err
	}

	return tx.Commit()
}

func (d *Database) MarkTerminal(
	ctx context.Context,
	id string,
	state models.JobState,
	report string,
	total int,
	errorMessage, details, stage string,
) error {
	if !state.IsTerminal() || total < 0 {
		return errors.New("invalid terminal state or totals")
	}

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer rollback(tx)

	var (
		previous models.JobState
		raw      []byte
	)
	if err = tx.QueryRowContext(ctx, `SELECT state,config_json FROM jobs WHERE id=$1 FOR UPDATE`, id).
		Scan(&previous, &raw); errors.Is(
		err,
		sql.ErrNoRows,
	) {
		return ErrJobNotFound
	} else if err != nil {
		return err
	}

	if previous.IsTerminal() {
		if previous != state {
			return errors.New("terminal state is immutable")
		}

		return tx.Commit()
	}

	if !canTransition(previous, state) {
		return fmt.Errorf("invalid job transition %s -> %s", previous, state)
	}

	config, secrets, err := sanitizeTerminalJobConfig(string(raw))
	if err != nil {
		return err
	}

	if err = scrubScannerText(ctx, tx, id, secrets); err != nil {
		return err
	}

	if err = changed(
		tx.ExecContext(
			ctx,
			`UPDATE jobs SET state=$2,config_json=$3,error=$4,error_details=$5,last_stage=$6,report_json_key=$7,total_violations=$8,completed_at=COALESCE(completed_at,CURRENT_TIMESTAMP),cleanup_pending=TRUE,updated_at=CURRENT_TIMESTAMP WHERE id=$1 AND state NOT IN ('DONE','FAILED') AND `+stateRankSQL+` <= $9`,
			id,
			state,
			config,
			redactKnownConfigSecrets(errorMessage, secrets),
			redactKnownConfigSecrets(details, secrets),
			stage,
			report,
			total,
			stateRank(state),
		),
	); err != nil {
		return err
	}

	return tx.Commit()
}
func (d *Database) MarkCleaned(ctx context.Context, id string) error {
	return changed(
		d.db.ExecContext(ctx, `UPDATE jobs SET cleanup_pending=FALSE WHERE id=$1 AND state IN ('DONE','FAILED')`, id),
	)
}

func scrubScannerText(ctx context.Context, tx *sql.Tx, id string, secrets []string) error {
	rows, err := tx.QueryContext(
		ctx,
		`SELECT scanner,error,results_path FROM job_scanners WHERE job_id=$1 FOR UPDATE`,
		id,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	type scannerText struct{ name, error, path string }

	var texts []scannerText

	for rows.Next() {
		v := scannerText{}
		if err = rows.Scan(&v.name, &v.error, &v.path); err != nil {
			rows.Close()
			return err
		}

		texts = append(texts, v)
	}

	err = rows.Err()
	rows.Close()

	if err != nil {
		return err
	}

	for _, v := range texts {
		if _, err = tx.ExecContext(
			ctx,
			`UPDATE job_scanners SET error=$3,results_path=$4 WHERE job_id=$1 AND scanner=$2`,
			id,
			v.name,
			redactKnownConfigSecrets(v.error, secrets),
			redactKnownConfigSecrets(v.path, secrets),
		); err != nil {
			return err
		}
	}

	return nil
}

func rollback(tx *sql.Tx) {
	if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		return // The primary operation error takes precedence over rollback cleanup.
	}
}
