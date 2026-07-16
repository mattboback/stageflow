package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/models"
)

type TerminalEventRecord struct {
	Event       string
	PayloadJSON string
}

func (d *Database) CompleteJobWithTerminalEvent(
	ctx context.Context,
	jobID string,
	payload *events.JobCompletedPayload,
) (err error) {
	if payload == nil {
		return errors.New("job.completed payload is required")
	}

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin complete job transaction: %w", err)
	}

	defer rollbackOnError(tx, &err, "complete job")

	configJSON, secrets, err := loadAndSanitizeTerminalJobConfig(ctx, tx, jobID)
	if err != nil {
		return err
	}

	now := time.Now()

	result, err := tx.ExecContext(
		ctx,
		bindPostgresParams(`
			UPDATE jobs
			SET state = ?, completed_at = ?, updated_at = ?
			WHERE id = ?
			  AND state = ?
		`),
		models.JobStateDone,
		now,
		now,
		jobID,
		models.JobStateCompleting,
	)
	if err != nil {
		return fmt.Errorf("complete job: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("complete job rows affected: %w", err)
	}

	if rows == 0 {
		state, stateErr := getJobStateTx(ctx, tx, jobID)
		if stateErr != nil {
			return stateErr
		}

		if state != models.JobStateDone {
			return fmt.Errorf("job %s not eligible for completion (state=%s)", jobID, state)
		}
	}

	if sanitizeErr := persistSanitizedTerminalJobConfig(ctx, tx, jobID, configJSON); sanitizeErr != nil {
		return sanitizeErr
	}

	if sanitizeErr := sanitizeTerminalJobTextFieldsTx(ctx, tx, jobID, secrets); sanitizeErr != nil {
		return sanitizeErr
	}

	if sanitizeErr := sanitizeTerminalAuditRecordsTx(ctx, tx, jobID, secrets); sanitizeErr != nil {
		return sanitizeErr
	}

	if insertErr := insertTerminalEventTx(ctx, tx, jobID, events.EventJobCompleted, payload); insertErr != nil {
		return insertErr
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return fmt.Errorf("commit complete job transaction: %w", commitErr)
	}

	return nil
}

//nolint:gocyclo // One transaction handles transition, idempotency, scrubbing, and outbox insertion.
func (d *Database) FailJobWithTerminalEvent(
	ctx context.Context,
	jobID, stage, errorMsg, errorDetails string,
	payload *events.JobFailedPayload,
) (transitioned bool, err error) {
	if payload == nil {
		return false, errors.New("job.failed payload is required")
	}

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin fail job transaction: %w", err)
	}

	defer rollbackOnError(tx, &err, "fail job")

	configJSON, secrets, err := loadAndSanitizeTerminalJobConfig(ctx, tx, jobID)
	if err != nil {
		return false, err
	}

	errorMsg = redactKnownConfigSecrets(errorMsg, secrets)
	errorDetails = redactKnownConfigSecrets(errorDetails, secrets)
	redactedPayload := *payload
	redactedPayload.Error = redactKnownConfigSecrets(payload.Error, secrets)
	redactedPayload.ErrorDetails = redactKnownConfigSecrets(payload.ErrorDetails, secrets)

	now := time.Now()

	result, err := tx.ExecContext(
		ctx,
		bindPostgresParams(`
			UPDATE jobs
			SET state = ?, error = ?, error_details = ?, last_stage = ?, completed_at = ?, updated_at = ?
			WHERE id = ?
			  AND state NOT IN (?, ?)
		`),
		models.JobStateFailed,
		errorMsg,
		errorDetails,
		stage,
		now,
		now,
		jobID,
		models.JobStateFailed,
		models.JobStateDone,
	)
	if err != nil {
		return false, fmt.Errorf("fail job: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("fail job rows affected: %w", err)
	}

	if rows == 0 {
		state, stateErr := getJobStateTx(ctx, tx, jobID)
		if stateErr != nil {
			return false, stateErr
		}

		if state != models.JobStateFailed {
			return false, fmt.Errorf("job %s not eligible for failure (state=%s)", jobID, state)
		}

		// Another caller completed the failure transaction while this caller
		// waited on the row lock. Reuse the canonical redacted outbox payload;
		// never overwrite or publish this caller's now-unredactable text.
		existing, loadErr := loadJobFailedTerminalEventTx(ctx, tx, jobID)
		if loadErr != nil {
			return false, loadErr
		}

		*payload = *existing

		if commitErr := tx.Commit(); commitErr != nil {
			return false, fmt.Errorf("commit idempotent fail job transaction: %w", commitErr)
		}

		return false, nil
	}

	if sanitizeErr := persistSanitizedTerminalJobConfig(ctx, tx, jobID, configJSON); sanitizeErr != nil {
		return false, sanitizeErr
	}

	if sanitizeErr := sanitizeTerminalJobTextFieldsTx(ctx, tx, jobID, secrets); sanitizeErr != nil {
		return false, sanitizeErr
	}

	if sanitizeErr := sanitizeTerminalAuditRecordsTx(ctx, tx, jobID, secrets); sanitizeErr != nil {
		return false, sanitizeErr
	}

	if insertErr := insertTerminalEventTx(
		ctx,
		tx,
		jobID,
		events.EventJobFailed,
		&redactedPayload,
	); insertErr != nil {
		return false, insertErr
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return false, fmt.Errorf("commit fail job transaction: %w", commitErr)
	}

	// The application service publishes this same pointer after the transaction.
	// Return the canonical redacted text so the terminal NATS event cannot
	// reintroduce values that were just removed from Postgres and the outbox.
	payload.Error = redactedPayload.Error
	payload.ErrorDetails = redactedPayload.ErrorDetails

	return true, nil
}

func loadJobFailedTerminalEventTx(
	ctx context.Context,
	tx *sql.Tx,
	jobID string,
) (*events.JobFailedPayload, error) {
	var payloadJSON string
	if err := tx.QueryRowContext(
		ctx,
		bindPostgresParams(`
			SELECT payload_json FROM terminal_events
			WHERE job_id = ? AND event = ?
		`),
		jobID,
		events.EventJobFailed,
	).Scan(&payloadJSON); err != nil {
		return nil, fmt.Errorf("load canonical job.failed terminal event: %w", err)
	}

	var payload events.JobFailedPayload
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		return nil, fmt.Errorf("decode canonical job.failed terminal event: %w", err)
	}

	return &payload, nil
}

func loadAndSanitizeTerminalJobConfig(
	ctx context.Context,
	tx *sql.Tx,
	jobID string,
) (string, []string, error) {
	var raw string
	if err := tx.QueryRowContext(
		ctx,
		bindPostgresParams(`SELECT config_json FROM jobs WHERE id = ? FOR UPDATE`),
		jobID,
	).Scan(&raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil, fmt.Errorf("job not found: %s", jobID)
		}

		return "", nil, fmt.Errorf("load terminal job config: %w", err)
	}

	configJSON, secrets, err := sanitizeTerminalJobConfig(raw)
	if err != nil {
		return "", nil, err
	}

	return configJSON, secrets, nil
}

func persistSanitizedTerminalJobConfig(
	ctx context.Context,
	tx *sql.Tx,
	jobID, configJSON string,
) error {
	if _, err := tx.ExecContext(
		ctx,
		bindPostgresParams(`UPDATE jobs SET config_json = ? WHERE id = ?`),
		configJSON,
		jobID,
	); err != nil {
		return fmt.Errorf("persist sanitized terminal job config: %w", err)
	}

	return nil
}

func (d *Database) ListUnpublishedTerminalEvents(ctx context.Context, jobID string) ([]TerminalEventRecord, error) {
	rows, err := d.queryContext(
		ctx,
		`
			SELECT event, payload_json
			FROM terminal_events
			WHERE job_id = ?
			  AND published_at IS NULL
			ORDER BY created_at, event
		`,
		jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("list terminal events: %w", err)
	}
	defer rows.Close()

	var out []TerminalEventRecord

	for rows.Next() {
		var eventName, payloadJSON string
		if scanErr := rows.Scan(&eventName, &payloadJSON); scanErr != nil {
			return nil, fmt.Errorf("scan terminal event: %w", scanErr)
		}

		out = append(out, TerminalEventRecord{
			Event:       eventName,
			PayloadJSON: payloadJSON,
		})
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterate terminal events: %w", rowsErr)
	}

	return out, nil
}

func (d *Database) MarkTerminalEventPublished(ctx context.Context, jobID, eventName string) error {
	_, err := d.execContext(
		ctx,
		`
			UPDATE terminal_events
			SET published_at = ?, updated_at = ?
			WHERE job_id = ?
			  AND event = ?
			  AND published_at IS NULL
		`,
		time.Now(),
		time.Now(),
		jobID,
		eventName,
	)
	if err != nil {
		return fmt.Errorf("mark terminal event published: %w", err)
	}

	return nil
}

func insertTerminalEventTx(ctx context.Context, tx *sql.Tx, jobID, eventName string, payload any) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal terminal event payload: %w", err)
	}

	now := time.Now()

	_, err = tx.ExecContext(
		ctx,
		bindPostgresParams(`
			INSERT INTO terminal_events (job_id, event, payload_json, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT (job_id, event) DO UPDATE
			SET payload_json = EXCLUDED.payload_json,
			    updated_at = EXCLUDED.updated_at
			WHERE terminal_events.published_at IS NULL
		`),
		jobID,
		eventName,
		string(payloadBytes),
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("insert terminal event: %w", err)
	}

	return nil
}

func getJobStateTx(ctx context.Context, tx *sql.Tx, jobID string) (models.JobState, error) {
	var state models.JobState

	err := tx.QueryRowContext(ctx, bindPostgresParams(`SELECT state FROM jobs WHERE id = ?`), jobID).Scan(&state)
	if errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("job not found: %s", jobID)
	}

	if err != nil {
		return "", fmt.Errorf("failed to load job state: %w", err)
	}

	return state, nil
}

func rollbackOnError(tx *sql.Tx, errp *error, label string) {
	if errp == nil || *errp == nil {
		return
	}

	if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
		*errp = errors.Join(*errp, fmt.Errorf("rollback %s transaction: %w", label, rollbackErr))
	}
}
