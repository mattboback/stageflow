package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/mattboback/stageflow/packages/shared-go/models"
)

// SetExpectedScanners sets the list of expected scanners for a job.
func (d *Database) SetExpectedScanners(ctx context.Context, jobID string, scanners []string) error {
	scannersJSON, err := json.Marshal(scanners)
	if err != nil {
		return fmt.Errorf("failed to marshal scanners: %w", err)
	}

	query := `
		UPDATE jobs
		SET expected_scanners = ?, completed_scanners = ?, scanner_results = ?, updated_at = ?
		WHERE id = ?
	`

	updateErr := d.execJobUpdate(ctx, jobID, query, string(scannersJSON), "[]", "{}", time.Now(), jobID)
	if updateErr != nil {
		return fmt.Errorf("failed to set expected scanners: %w", updateErr)
	}

	return nil
}

// RecordScannerCompletion records that a scanner has completed and stores its results.
// Returns (allComplete, error) where allComplete is true if all expected scanners are done.
//
//nolint:gocognit,gocyclo // Complex transaction logic requires multiple validation steps
func (d *Database) RecordScannerCompletion(
	ctx context.Context,
	jobID string,
	result *models.ScannerResult,
) (allComplete bool, err error) {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err == nil {
			return
		}

		rollbackErr := tx.Rollback()
		if rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			err = errors.Join(err, fmt.Errorf("failed to rollback scanner completion transaction: %w", rollbackErr))
		}
	}()

	var expectedJSON, completedJSON, resultsJSON sql.NullString

	row := tx.QueryRowContext(ctx, `
		SELECT expected_scanners, completed_scanners, scanner_results
		FROM jobs
		WHERE id = ?
	`, jobID)

	rowScanErr := row.Scan(&expectedJSON, &completedJSON, &resultsJSON)
	if errors.Is(rowScanErr, sql.ErrNoRows) {
		return false, fmt.Errorf("job not found: %s", jobID)
	}

	if rowScanErr != nil {
		return false, fmt.Errorf("failed to load job for scanner completion: %w", rowScanErr)
	}

	var expectedScanners []string
	if expectedJSON.Valid && expectedJSON.String != "" {
		unmarshalErr := json.Unmarshal([]byte(expectedJSON.String), &expectedScanners)
		if unmarshalErr != nil {
			return false, fmt.Errorf("failed to unmarshal expected scanners: %w", unmarshalErr)
		}
	}

	if len(expectedScanners) == 0 {
		return false, fmt.Errorf("no expected scanners configured for job %s", jobID)
	}

	found := false

	for _, s := range expectedScanners {
		if s == result.ScannerType {
			found = true

			break
		}
	}

	if !found {
		return false, fmt.Errorf("unexpected scanner type: %s", result.ScannerType)
	}

	completedScanners := make([]string, 0, 1)
	if completedJSON.Valid && completedJSON.String != "" {
		unmarshalErr := json.Unmarshal([]byte(completedJSON.String), &completedScanners)
		if unmarshalErr != nil {
			return false, fmt.Errorf("failed to unmarshal completed scanners: %w", unmarshalErr)
		}
	}

	for _, s := range completedScanners {
		if s == result.ScannerType {
			allComplete = len(completedScanners) >= len(expectedScanners)

			commitErr := tx.Commit()
			if commitErr != nil {
				return false, fmt.Errorf("failed to commit transaction: %w", commitErr)
			}

			return allComplete, nil
		}
	}

	completedScanners = append(completedScanners, result.ScannerType)

	var scannerResults map[string]*models.ScannerResult
	if resultsJSON.Valid && resultsJSON.String != "" {
		unmarshalErr := json.Unmarshal([]byte(resultsJSON.String), &scannerResults)
		if unmarshalErr != nil {
			return false, fmt.Errorf("failed to unmarshal scanner results: %w", unmarshalErr)
		}
	}

	if scannerResults == nil {
		scannerResults = make(map[string]*models.ScannerResult)
	}

	scannerResults[result.ScannerType] = result

	completedBytes, err := json.Marshal(completedScanners)
	if err != nil {
		return false, fmt.Errorf("failed to marshal completed scanners: %w", err)
	}

	resultsBytes, err := json.Marshal(scannerResults)
	if err != nil {
		return false, fmt.Errorf("failed to marshal scanner results: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE jobs
		SET completed_scanners = ?, scanner_results = ?, updated_at = ?
		WHERE id = ?
	`, string(completedBytes), string(resultsBytes), time.Now(), jobID)
	if err != nil {
		return false, fmt.Errorf("failed to record scanner completion: %w", err)
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return false, fmt.Errorf("failed to commit scanner completion: %w", commitErr)
	}

	allComplete = len(completedScanners) >= len(expectedScanners)

	return allComplete, nil
}

// RecordScannerFailure records that a scanner has failed.
// Returns (allComplete, error) where allComplete is true if all expected scanners are done (including failures).
func (d *Database) RecordScannerFailure(ctx context.Context, jobID, scannerType, errorMsg string) (bool, error) {
	result := &models.ScannerResult{
		ScannerType: scannerType,
		Success:     false,
		Error:       errorMsg,
	}

	return d.RecordScannerCompletion(ctx, jobID, result)
}

// GetScannerResults returns all scanner results for a job.
func (d *Database) GetScannerResults(ctx context.Context, jobID string) (map[string]*models.ScannerResult, error) {
	job, err := d.GetJob(ctx, jobID)
	if err != nil {
		return nil, err
	}

	return job.ScannerResults, nil
}
