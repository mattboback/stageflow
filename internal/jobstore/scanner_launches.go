package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/mattboback/stageflow/libs/go/models"
)

const (
	ScannerLaunchPending   = "pending"
	ScannerLaunchLaunching = "launching"
	ScannerLaunchLaunched  = "launched"
	ScannerLaunchFailed    = "failed"
)

// ScannerLaunch is the durable launch record for one scanner container.
type ScannerLaunch struct {
	JobID        string
	ScannerType  string
	State        string
	ContainerID  string
	AttemptCount int
	LastError    string
}

// PrepareScannerLaunches atomically initializes the expected scanner set and
// its launch ledger. Repeated calls preserve completion/results and reject a
// different scanner set, preventing duplicate events from resetting an active
// scan.
func (d *Database) PrepareScannerLaunches(ctx context.Context, jobID string, scanners []string) (err error) {
	if validationErr := validateScannerTypes(scanners); validationErr != nil {
		return validationErr
	}

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin scanner launch preparation: %w", err)
	}

	defer func() {
		if err == nil {
			return
		}

		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			err = errors.Join(err, fmt.Errorf("rollback scanner launch preparation: %w", rollbackErr))
		}
	}()

	preparation, loadErr := loadScannerLaunchPreparation(ctx, tx, jobID)
	if loadErr != nil {
		return loadErr
	}

	if len(preparation.expected) == 0 {
		if initializeErr := initializeExpectedScanners(ctx, tx, jobID, scanners); initializeErr != nil {
			return initializeErr
		}
	} else if !slices.Equal(preparation.expected, scanners) {
		return fmt.Errorf("job %s scanner set changed from %v to %v", jobID, preparation.expected, scanners)
	}

	if insertErr := insertScannerLaunches(ctx, tx, jobID, scanners, preparation.completed); insertErr != nil {
		return insertErr
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return fmt.Errorf("commit scanner launch preparation: %w", commitErr)
	}

	return nil
}

type scannerLaunchPreparation struct {
	expected  []string
	completed map[string]struct{}
}

func validateScannerTypes(scanners []string) error {
	if len(scanners) == 0 {
		return errors.New("at least one scanner is required")
	}

	if slices.Contains(scanners, "") {
		return errors.New("scanner type cannot be empty")
	}

	seen := make(map[string]struct{}, len(scanners))
	for _, scannerType := range scanners {
		if _, duplicate := seen[scannerType]; duplicate {
			return fmt.Errorf("duplicate scanner type: %s", scannerType)
		}

		seen[scannerType] = struct{}{}
	}

	return nil
}

func loadScannerLaunchPreparation(
	ctx context.Context,
	tx *sql.Tx,
	jobID string,
) (*scannerLaunchPreparation, error) {
	var (
		state         models.JobState
		expectedJSON  sql.NullString
		completedJSON sql.NullString
	)

	row := tx.QueryRowContext(ctx, bindPostgresParams(`
		SELECT state, expected_scanners, completed_scanners
		FROM jobs
		WHERE id = ?
		FOR UPDATE
	`), jobID)

	scanErr := row.Scan(&state, &expectedJSON, &completedJSON)
	if errors.Is(scanErr, sql.ErrNoRows) {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}

	if scanErr != nil {
		return nil, fmt.Errorf("load job for scanner launch preparation: %w", scanErr)
	}

	if state != models.JobStateScanning {
		return nil, fmt.Errorf("job %s is not scanning (state=%s)", jobID, state)
	}

	expected, decodeErr := decodeScannerList(expectedJSON, "expected")
	if decodeErr != nil {
		return nil, decodeErr
	}

	completedScanners, decodeErr := decodeScannerList(completedJSON, "completed")
	if decodeErr != nil {
		return nil, decodeErr
	}

	completed := make(map[string]struct{}, len(completedScanners))
	for _, scannerType := range completedScanners {
		completed[scannerType] = struct{}{}
	}

	return &scannerLaunchPreparation{expected: expected, completed: completed}, nil
}

func decodeScannerList(raw sql.NullString, field string) ([]string, error) {
	if !raw.Valid || raw.String == "" {
		return nil, nil
	}

	var scanners []string
	if err := json.Unmarshal([]byte(raw.String), &scanners); err != nil {
		return nil, fmt.Errorf("decode %s scanners: %w", field, err)
	}

	return scanners, nil
}

func initializeExpectedScanners(ctx context.Context, tx *sql.Tx, jobID string, scanners []string) error {
	expectedBytes, err := json.Marshal(scanners)
	if err != nil {
		return fmt.Errorf("encode expected scanners: %w", err)
	}

	_, err = tx.ExecContext(ctx, bindPostgresParams(`
		UPDATE jobs
		SET expected_scanners = ?,
		    completed_scanners = COALESCE(completed_scanners, '[]'),
		    scanner_results = COALESCE(scanner_results, '{}'),
		    updated_at = ?
		WHERE id = ?
	`), string(expectedBytes), time.Now(), jobID)
	if err != nil {
		return fmt.Errorf("initialize expected scanners: %w", err)
	}

	return nil
}

func insertScannerLaunches(
	ctx context.Context,
	tx *sql.Tx,
	jobID string,
	scanners []string,
	completed map[string]struct{},
) error {
	for _, scannerType := range scanners {
		launchState := ScannerLaunchPending
		if _, ok := completed[scannerType]; ok {
			launchState = ScannerLaunchLaunched
		}

		_, err := tx.ExecContext(ctx, bindPostgresParams(`
			INSERT INTO scanner_launches (job_id, scanner_type, state, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT (job_id, scanner_type) DO NOTHING
		`), jobID, scannerType, launchState, time.Now(), time.Now())
		if err != nil {
			return fmt.Errorf("initialize launch for scanner %s: %w", scannerType, err)
		}
	}

	return nil
}

// ClaimScannerLaunch grants exactly one normal event handler ownership of a
// pending launch. A launching or launched row is deliberately left untouched.
func (d *Database) ClaimScannerLaunch(ctx context.Context, jobID, scannerType string) (bool, error) {
	return d.claimScannerLaunch(ctx, jobID, scannerType, false)
}

// ClaimScannerLaunchRecovery takes ownership of any incomplete scanner launch
// during process startup, including a row marked launched so the new process
// can reattach a container monitor. Deterministic Podman names make this safe.
func (d *Database) ClaimScannerLaunchRecovery(ctx context.Context, jobID, scannerType string) (bool, error) {
	return d.claimScannerLaunch(ctx, jobID, scannerType, true)
}

func (d *Database) claimScannerLaunch(
	ctx context.Context,
	jobID, scannerType string,
	recovery bool,
) (bool, error) {
	eligible := "state = ?"
	args := []any{ScannerLaunchLaunching, time.Now(), time.Now(), jobID, scannerType, ScannerLaunchPending}

	if recovery {
		eligible = "state IN (?, ?, ?, ?)"
		args = []any{
			ScannerLaunchLaunching,
			time.Now(),
			time.Now(),
			jobID,
			scannerType,
			ScannerLaunchPending,
			ScannerLaunchLaunching,
			ScannerLaunchLaunched,
			ScannerLaunchFailed,
		}
	}

	query := `
		UPDATE scanner_launches
		SET state = ?, attempt_count = attempt_count + 1, claimed_at = ?, updated_at = ?, last_error = NULL
		WHERE job_id = ?
		  AND scanner_type = ?
		  AND ` + eligible + `
		  AND EXISTS (
		      SELECT 1 FROM jobs
		      WHERE jobs.id = scanner_launches.job_id AND jobs.state = 'SCANNING'
		  )
	`

	result, err := d.execContext(ctx, query, args...)
	if err != nil {
		return false, fmt.Errorf("claim scanner launch: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read scanner launch claim result: %w", err)
	}

	if rows > 0 {
		return true, nil
	}

	var exists bool

	existenceErr := d.queryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM scanner_launches WHERE job_id = ? AND scanner_type = ?
		)
	`, jobID, scannerType).Scan(&exists)
	if existenceErr != nil {
		return false, fmt.Errorf("check scanner launch existence: %w", existenceErr)
	}

	if !exists {
		return false, fmt.Errorf("scanner launch not found: job=%s scanner=%s", jobID, scannerType)
	}

	return false, nil
}

func (d *Database) MarkScannerLaunched(ctx context.Context, jobID, scannerType, containerID string) error {
	if containerID == "" {
		return errors.New("container ID is required")
	}

	result, err := d.execContext(ctx, `
		UPDATE scanner_launches
		SET state = ?, container_id = ?, launched_at = ?, updated_at = ?, last_error = NULL
		WHERE job_id = ? AND scanner_type = ? AND state = ?
	`, ScannerLaunchLaunched, containerID, time.Now(), time.Now(), jobID, scannerType, ScannerLaunchLaunching)
	if err != nil {
		return fmt.Errorf("mark scanner launched: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read scanner launched result: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("scanner launch is not claimed: job=%s scanner=%s", jobID, scannerType)
	}

	return nil
}

func (d *Database) MarkScannerLaunchFailed(
	ctx context.Context,
	jobID, scannerType, errorMessage string,
) error {
	result, err := d.execContext(ctx, `
		UPDATE scanner_launches
		SET state = ?, last_error = ?, updated_at = ?
		WHERE job_id = ? AND scanner_type = ?
	`, ScannerLaunchFailed, errorMessage, time.Now(), jobID, scannerType)
	if err != nil {
		return fmt.Errorf("mark scanner launch failed: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read scanner launch failure result: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("scanner launch not found: job=%s scanner=%s", jobID, scannerType)
	}

	return nil
}

// GetScannerLaunch returns one launch record for diagnostics and tests.
func (d *Database) GetScannerLaunch(ctx context.Context, jobID, scannerType string) (*ScannerLaunch, error) {
	var (
		launch      ScannerLaunch
		containerID sql.NullString
		lastError   sql.NullString
	)

	err := d.queryRowContext(ctx, `
		SELECT job_id, scanner_type, state, container_id, attempt_count, last_error
		FROM scanner_launches
		WHERE job_id = ? AND scanner_type = ?
	`, jobID, scannerType).Scan(
		&launch.JobID,
		&launch.ScannerType,
		&launch.State,
		&containerID,
		&launch.AttemptCount,
		&lastError,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("scanner launch not found: job=%s scanner=%s", jobID, scannerType)
	}

	if err != nil {
		return nil, fmt.Errorf("get scanner launch: %w", err)
	}

	if containerID.Valid {
		launch.ContainerID = containerID.String
	}

	if lastError.Valid {
		launch.LastError = lastError.String
	}

	return &launch, nil
}
