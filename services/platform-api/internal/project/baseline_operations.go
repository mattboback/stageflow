package project

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// QueueBaselinePromotion durably records a promotion before any object-store
// write. Repeating the same request returns the existing operation; a different
// concurrent promotion is rejected until reconciliation completes it.
//
//nolint:gocyclo // The transaction validates every mutually exclusive journal state before writing.
func (s *Store) QueueBaselinePromotion(
	ctx context.Context,
	projectID, previousJobID, jobID, sourceKey string,
) (*BaselineOperation, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer rollbackTx(tx)

	if exists, checkErr := operationExistsTx(ctx, tx, BaselineOperationDeleteProject, projectID); checkErr != nil {
		return nil, checkErr
	} else if exists {
		return nil, ErrProjectDeletionPending
	}

	if exists, checkErr := operationExistsTx(ctx, tx, BaselineOperationBackfill, projectID); checkErr != nil {
		return nil, checkErr
	} else if exists {
		return nil, ErrBaselineOperationPending
	}

	currentJobID, projectExists, checkErr := currentBaselineJobTx(ctx, tx, projectID)
	if checkErr != nil {
		return nil, checkErr
	}

	if !projectExists {
		return nil, ErrNotFound
	}

	existing, getErr := getOperationByKindProjectTx(ctx, tx, BaselineOperationPromote, projectID)
	if getErr == nil {
		if existing.JobID == jobID && existing.PreviousJobID == previousJobID && existing.SourceKey == sourceKey {
			return existing, tx.Commit()
		}

		return nil, ErrBaselineOperationPending
	}

	if !errors.Is(getErr, sql.ErrNoRows) {
		return nil, getErr
	}

	// The handler may have loaded the project before another promotion
	// committed. Compare the baseline pointer inside this transaction so a
	// stale request cannot copy an object and then report a false success.
	if currentJobID != previousJobID && currentJobID != jobID {
		return nil, ErrBaselineSuperseded
	}

	// Re-promoting a previously replaced job makes it active again. Cancel any
	// queued deletion for that exact object before recording the promotion.
	if _, execErr := tx.ExecContext(ctx, `
		DELETE FROM baseline_operations
		WHERE kind = ? AND project_id = ? AND job_id = ?`,
		BaselineOperationDeleteObject, projectID, jobID); execErr != nil {
		return nil, execErr
	}

	op, err := insertOperationTx(ctx, tx, BaselineOperation{
		Kind:          BaselineOperationPromote,
		State:         BaselineOperationObjectPending,
		ProjectID:     projectID,
		JobID:         jobID,
		PreviousJobID: previousJobID,
		SourceKey:     sourceKey,
	})
	if err != nil {
		return nil, err
	}

	return op, tx.Commit()
}

// QueueBaselineBackfill records preservation of an already-selected legacy
// baseline. It is mutually exclusive with project deletion and promotion.
func (s *Store) QueueBaselineBackfill(
	ctx context.Context,
	projectID, jobID, sourceKey string,
) (*BaselineOperation, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer rollbackTx(tx)

	for _, kind := range []string{BaselineOperationDeleteProject, BaselineOperationPromote} {
		if exists, checkErr := operationExistsTx(ctx, tx, kind, projectID); checkErr != nil {
			return nil, checkErr
		} else if exists {
			return nil, ErrBaselineOperationPending
		}
	}

	existing, getErr := getOperationByKindProjectTx(ctx, tx, BaselineOperationBackfill, projectID)
	if getErr == nil {
		if existing.JobID == jobID && existing.SourceKey == sourceKey {
			return existing, tx.Commit()
		}

		return nil, ErrBaselineOperationPending
	}

	if !errors.Is(getErr, sql.ErrNoRows) {
		return nil, getErr
	}

	if currentJobID, projectExists, checkErr := currentBaselineJobTx(ctx, tx, projectID); checkErr != nil {
		return nil, checkErr
	} else if !projectExists {
		return nil, ErrNotFound
	} else if currentJobID != jobID {
		return nil, ErrBaselineSuperseded
	}

	if _, execErr := tx.ExecContext(ctx, `
		DELETE FROM baseline_operations
		WHERE kind = ? AND project_id = ? AND job_id = ?`,
		BaselineOperationDeleteObject, projectID, jobID); execErr != nil {
		return nil, execErr
	}

	op, err := insertOperationTx(ctx, tx, BaselineOperation{
		Kind:      BaselineOperationBackfill,
		State:     BaselineOperationObjectPending,
		ProjectID: projectID,
		JobID:     jobID,
		SourceKey: sourceKey,
	})
	if err != nil {
		return nil, err
	}

	return op, tx.Commit()
}

// QueueProjectDeletion records the active baseline object that must be removed
// before the project row may be deleted.
func (s *Store) QueueProjectDeletion(
	ctx context.Context,
	projectID string,
) (*BaselineOperation, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer rollbackTx(tx)

	existing, getErr := getOperationByKindProjectTx(ctx, tx, BaselineOperationDeleteProject, projectID)
	if getErr == nil {
		return existing, tx.Commit()
	}

	if !errors.Is(getErr, sql.ErrNoRows) {
		return nil, getErr
	}

	for _, kind := range []string{BaselineOperationPromote, BaselineOperationBackfill} {
		if exists, checkErr := operationExistsTx(ctx, tx, kind, projectID); checkErr != nil {
			return nil, checkErr
		} else if exists {
			return nil, ErrBaselineOperationPending
		}
	}

	// Select the active pointer in the same transaction that journals the
	// deletion. A project object loaded by the handler may be stale after a
	// concurrent promotion; trusting it would orphan the newly active object in
	// the lifecycle-exempt bucket.
	baselineJobID, projectExists, checkErr := currentBaselineJobTx(ctx, tx, projectID)
	if checkErr != nil {
		return nil, checkErr
	}

	if !projectExists {
		return nil, ErrNotFound
	}

	op, err := insertOperationTx(ctx, tx, BaselineOperation{
		Kind:      BaselineOperationDeleteProject,
		State:     BaselineOperationObjectPending,
		ProjectID: projectID,
		JobID:     baselineJobID,
	})
	if err != nil {
		return nil, err
	}

	return op, tx.Commit()
}

func (s *Store) ListBaselineOperations(ctx context.Context) ([]BaselineOperation, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, kind, state, project_id, job_id, previous_job_id, source_key,
		       attempts, last_error, created_at, updated_at
		FROM baseline_operations
		ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var operations []BaselineOperation

	for rows.Next() {
		var op BaselineOperation

		if scanErr := scanBaselineOperation(rows, &op); scanErr != nil {
			return nil, scanErr
		}

		operations = append(operations, op)
	}

	return operations, rows.Err()
}

func (s *Store) MarkBaselineOperationObjectReady(ctx context.Context, id int64) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE baseline_operations
		SET state = ?, last_error = '', updated_at = ?
		WHERE id = ?`, BaselineOperationCommitPending, time.Now().UTC(), id)
	if err != nil {
		return err
	}

	return requireChangedOperation(result)
}

func (s *Store) RecordBaselineOperationFailure(ctx context.Context, id int64, cause string) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE baseline_operations
		SET attempts = attempts + 1, last_error = ?, updated_at = ?
		WHERE id = ?`, cause, time.Now().UTC(), id)
	if err != nil {
		return err
	}

	return requireChangedOperation(result)
}

// CompleteBaselinePromotion atomically selects the new baseline, queues
// deletion of the replaced object, and removes the promotion journal entry.
func (s *Store) CompleteBaselinePromotion(ctx context.Context, op BaselineOperation) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollbackTx(tx)

	currentJobID, projectExists, err := currentBaselineJobTx(ctx, tx, op.ProjectID)
	if err != nil {
		return err
	}

	if !projectExists || baselinePromotionSuperseded(currentJobID, op) {
		return completeSupersededPromotionTx(ctx, tx, op)
	}

	if currentJobID != op.JobID {
		result, updateErr := tx.ExecContext(ctx, `
			UPDATE projects SET baseline_job_id = ?, updated_at = ? WHERE id = ?`,
			op.JobID, time.Now().UTC(), op.ProjectID)
		if updateErr != nil {
			return updateErr
		}

		if changedErr := requireChangedOperation(result); changedErr != nil {
			return changedErr
		}
	}

	if op.PreviousJobID != "" && op.PreviousJobID != op.JobID {
		if queueErr := enqueueObjectDeleteTx(ctx, tx, op.ProjectID, op.PreviousJobID); queueErr != nil {
			return queueErr
		}
	}

	if deleteErr := deleteBaselineOperationTx(ctx, tx, op.ID); deleteErr != nil {
		return deleteErr
	}

	return tx.Commit()
}

// CompleteBaselineOperation commits a non-promotion operation after its object
// mutation succeeded. Project deletion and journal removal share one SQLite
// transaction, closing the crash window that previously orphaned objects.
func (s *Store) CompleteBaselineOperation(ctx context.Context, op BaselineOperation) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollbackTx(tx)

	if op.Kind == BaselineOperationDeleteProject {
		if _, deleteErr := tx.ExecContext(
			ctx,
			`DELETE FROM projects WHERE id = ?`,
			op.ProjectID,
		); deleteErr != nil {
			return deleteErr
		}
	}

	if deleteErr := deleteBaselineOperationTx(ctx, tx, op.ID); deleteErr != nil {
		return deleteErr
	}

	return tx.Commit()
}

func insertOperationTx(
	ctx context.Context,
	tx *sql.Tx,
	op BaselineOperation,
) (*BaselineOperation, error) {
	now := time.Now().UTC()

	result, err := tx.ExecContext(ctx, `
		INSERT INTO baseline_operations
		(kind, state, project_id, job_id, previous_job_id, source_key, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		op.Kind, op.State, op.ProjectID, op.JobID, op.PreviousJobID, op.SourceKey, now, now)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	op.ID = id
	op.CreatedAt = now
	op.UpdatedAt = now

	return &op, nil
}

func enqueueObjectDeleteTx(ctx context.Context, tx *sql.Tx, projectID, jobID string) error {
	if jobID == "" {
		return nil
	}

	now := time.Now().UTC()
	_, err := tx.ExecContext(ctx, `
		INSERT INTO baseline_operations
		(kind, state, project_id, job_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(kind, project_id, job_id) DO NOTHING`,
		BaselineOperationDeleteObject, BaselineOperationObjectPending,
		projectID, jobID, now, now)

	return err
}

func operationExistsTx(ctx context.Context, tx *sql.Tx, kind, projectID string) (bool, error) {
	var one int

	err := tx.QueryRowContext(ctx,
		`SELECT 1 FROM baseline_operations WHERE kind = ? AND project_id = ? LIMIT 1`,
		kind, projectID,
	).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}

	return err == nil, err
}

func getOperationByKindProjectTx(
	ctx context.Context,
	tx *sql.Tx,
	kind, projectID string,
) (*BaselineOperation, error) {
	row := tx.QueryRowContext(ctx, `
		SELECT id, kind, state, project_id, job_id, previous_job_id, source_key,
		       attempts, last_error, created_at, updated_at
		FROM baseline_operations WHERE kind = ? AND project_id = ? LIMIT 1`,
		kind, projectID)

	var op BaselineOperation
	if err := scanBaselineOperation(row, &op); err != nil {
		return nil, err
	}

	return &op, nil
}

type operationScanner interface {
	Scan(dest ...any) error
}

func scanBaselineOperation(row operationScanner, op *BaselineOperation) error {
	return row.Scan(
		&op.ID, &op.Kind, &op.State, &op.ProjectID, &op.JobID,
		&op.PreviousJobID, &op.SourceKey, &op.Attempts, &op.LastError,
		&op.CreatedAt, &op.UpdatedAt,
	)
}

func requireChangedOperation(result sql.Result) error {
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if count == 0 {
		return errors.New("baseline operation no longer exists")
	}

	return nil
}

func rollbackTx(tx *sql.Tx) {
	_ = tx.Rollback() //nolint:errcheck // Best-effort deferred cleanup; commit errors are returned directly.
}

func currentBaselineJobTx(
	ctx context.Context,
	tx *sql.Tx,
	projectID string,
) (string, bool, error) {
	var current sql.NullString

	err := tx.QueryRowContext(ctx,
		`SELECT baseline_job_id FROM projects WHERE id = ?`, projectID,
	).Scan(&current)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}

	if err != nil {
		return "", false, err
	}

	return current.String, true, nil
}

func baselinePromotionSuperseded(currentJobID string, op BaselineOperation) bool {
	return currentJobID != op.PreviousJobID && currentJobID != op.JobID
}

func completeSupersededPromotionTx(
	ctx context.Context,
	tx *sql.Tx,
	op BaselineOperation,
) error {
	if err := enqueueObjectDeleteTx(ctx, tx, op.ProjectID, op.JobID); err != nil {
		return err
	}

	if err := deleteBaselineOperationTx(ctx, tx, op.ID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return ErrBaselineSuperseded
}

func deleteBaselineOperationTx(ctx context.Context, tx *sql.Tx, id int64) error {
	_, err := tx.ExecContext(ctx, `DELETE FROM baseline_operations WHERE id = ?`, id)

	return err
}
