package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mattboback/stageflow/libs/go/models"
)

const terminalSecretMigration = "2026-07-terminal-secret-sanitization-v1"

type storedAuditEvent struct {
	id           int64
	payload      sql.NullString
	handlerError sql.NullString
}

type storedTerminalEvent struct {
	event   string
	payload string
}

// SanitizeLegacyTerminalRecords performs the one-time upgrade backfill for
// terminal jobs created before terminal secret scrubbing existed. Each job is
// committed atomically; the durable marker is written only after the complete
// backfill, so an interrupted startup safely retries the remaining work.
func (d *Database) SanitizeLegacyTerminalRecords(ctx context.Context) (int, error) {
	completed, err := d.terminalSecretMigrationCompleted(ctx)
	if err != nil || completed {
		return 0, err
	}

	jobIDs, err := d.listTerminalJobIDs(ctx)
	if err != nil {
		return 0, err
	}

	for _, jobID := range jobIDs {
		if err = d.sanitizeLegacyTerminalJob(ctx, jobID); err != nil {
			return 0, fmt.Errorf("sanitize terminal job %s: %w", jobID, err)
		}
	}

	if _, err = d.execContext(ctx, `
		INSERT INTO maintenance_migrations (name, completed_at)
		VALUES (?, ?)
		ON CONFLICT (name) DO NOTHING
	`, terminalSecretMigration, time.Now().UTC()); err != nil {
		return 0, fmt.Errorf("mark terminal secret migration complete: %w", err)
	}

	return len(jobIDs), nil
}

func (d *Database) terminalSecretMigrationCompleted(ctx context.Context) (bool, error) {
	var exists bool
	if err := d.queryRowContext(ctx, `
		SELECT EXISTS (SELECT 1 FROM maintenance_migrations WHERE name = ?)
	`, terminalSecretMigration).Scan(&exists); err != nil {
		return false, fmt.Errorf("check terminal secret migration: %w", err)
	}

	return exists, nil
}

func (d *Database) listTerminalJobIDs(ctx context.Context) ([]string, error) {
	rows, err := d.queryContext(ctx, `
		SELECT id FROM jobs WHERE state IN (?, ?) ORDER BY id
	`, models.JobStateDone, models.JobStateFailed)
	if err != nil {
		return nil, fmt.Errorf("list terminal jobs for secret migration: %w", err)
	}
	defer rows.Close()

	var jobIDs []string

	for rows.Next() {
		var jobID string
		if scanErr := rows.Scan(&jobID); scanErr != nil {
			return nil, fmt.Errorf("scan terminal job for secret migration: %w", scanErr)
		}

		jobIDs = append(jobIDs, jobID)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate terminal jobs for secret migration: %w", err)
	}

	return jobIDs, nil
}

func (d *Database) sanitizeLegacyTerminalJob(ctx context.Context, jobID string) (err error) {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin terminal secret migration: %w", err)
	}
	defer rollbackOnError(tx, &err, "terminal secret migration")

	var (
		rawConfig string
	)
	if err = tx.QueryRowContext(ctx, bindPostgresParams(`
		SELECT config_json FROM jobs WHERE id = ? FOR UPDATE
	`), jobID).Scan(&rawConfig); err != nil {
		return fmt.Errorf("load legacy terminal job: %w", err)
	}

	configJSON, secrets, err := sanitizeTerminalJobConfig(rawConfig)
	if err != nil {
		return err
	}

	if err = persistSanitizedTerminalJobConfig(ctx, tx, jobID, configJSON); err != nil {
		return err
	}

	if err = sanitizeTerminalJobTextFieldsTx(ctx, tx, jobID, secrets); err != nil {
		return err
	}

	if err = sanitizeTerminalAuditRecordsTx(ctx, tx, jobID, secrets); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit terminal secret migration: %w", err)
	}

	return nil
}

func redactNullableString(value sql.NullString, secrets []string) any {
	if !value.Valid {
		return nil
	}

	return redactKnownConfigSecrets(value.String, secrets)
}

func sanitizeTerminalJobTextFieldsTx(
	ctx context.Context,
	tx *sql.Tx,
	jobID string,
	secrets []string,
) error {
	var errorMessage, errorDetails, scannerResults sql.NullString
	if err := tx.QueryRowContext(ctx, bindPostgresParams(`
		SELECT error, error_details, scanner_results FROM jobs WHERE id = ? FOR UPDATE
	`), jobID).Scan(&errorMessage, &errorDetails, &scannerResults); err != nil {
		return fmt.Errorf("load terminal job text fields: %w", err)
	}

	results := any(nil)
	if scannerResults.Valid {
		results = sanitizeStoredPayloadJSON(scannerResults.String, secrets)
	}

	if _, err := tx.ExecContext(ctx, bindPostgresParams(`
		UPDATE jobs SET error = ?, error_details = ?, scanner_results = ? WHERE id = ?
	`),
		redactNullableString(errorMessage, secrets),
		redactNullableString(errorDetails, secrets),
		results,
		jobID,
	); err != nil {
		return fmt.Errorf("sanitize terminal job text fields: %w", err)
	}

	return nil
}

func sanitizeTerminalAuditRecordsTx(
	ctx context.Context,
	tx *sql.Tx,
	jobID string,
	secrets []string,
) error {
	auditEvents, err := loadStoredAuditEventsTx(ctx, tx, jobID)
	if err != nil {
		return err
	}

	for _, event := range auditEvents {
		payload := any(nil)
		if event.payload.Valid {
			payload = sanitizeStoredPayloadJSON(event.payload.String, secrets)
		}

		handlerError := any(nil)
		if event.handlerError.Valid {
			handlerError = redactKnownConfigSecrets(event.handlerError.String, secrets)
		}

		if _, err = tx.ExecContext(ctx, bindPostgresParams(`
			UPDATE job_events SET payload_json = ?, handler_error = ? WHERE id = ?
		`), payload, handlerError, event.id); err != nil {
			return fmt.Errorf("sanitize job event %d: %w", event.id, err)
		}
	}

	terminalEvents, err := loadStoredTerminalEventsTx(ctx, tx, jobID)
	if err != nil {
		return err
	}

	for _, event := range terminalEvents {
		if _, err = tx.ExecContext(ctx, bindPostgresParams(`
			UPDATE terminal_events SET payload_json = ?, updated_at = ?
			WHERE job_id = ? AND event = ?
		`), sanitizeStoredPayloadJSON(event.payload, secrets), time.Now().UTC(), jobID, event.event); err != nil {
			return fmt.Errorf("sanitize terminal event %s: %w", event.event, err)
		}
	}

	return nil
}

func loadStoredAuditEventsTx(
	ctx context.Context,
	tx *sql.Tx,
	jobID string,
) ([]storedAuditEvent, error) {
	rows, err := tx.QueryContext(ctx, bindPostgresParams(`
		SELECT id, payload_json, handler_error FROM job_events
		WHERE job_id = ? OR payload_job_id = ?
		ORDER BY id FOR UPDATE
	`), jobID, jobID)
	if err != nil {
		return nil, fmt.Errorf("load job events for terminal sanitization: %w", err)
	}
	defer rows.Close()

	var events []storedAuditEvent

	for rows.Next() {
		var event storedAuditEvent
		if scanErr := rows.Scan(&event.id, &event.payload, &event.handlerError); scanErr != nil {
			return nil, fmt.Errorf("scan job event for terminal sanitization: %w", scanErr)
		}

		events = append(events, event)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterate job events for terminal sanitization: %w", rowsErr)
	}

	if closeErr := rows.Close(); closeErr != nil {
		return nil, fmt.Errorf("close job events for terminal sanitization: %w", closeErr)
	}

	return events, nil
}

func loadStoredTerminalEventsTx(
	ctx context.Context,
	tx *sql.Tx,
	jobID string,
) ([]storedTerminalEvent, error) {
	rows, err := tx.QueryContext(ctx, bindPostgresParams(`
		SELECT event, payload_json FROM terminal_events WHERE job_id = ? FOR UPDATE
	`), jobID)
	if err != nil {
		return nil, fmt.Errorf("load terminal events for sanitization: %w", err)
	}
	defer rows.Close()

	var events []storedTerminalEvent

	for rows.Next() {
		var event storedTerminalEvent
		if scanErr := rows.Scan(&event.event, &event.payload); scanErr != nil {
			return nil, fmt.Errorf("scan terminal event for sanitization: %w", scanErr)
		}

		events = append(events, event)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterate terminal events for sanitization: %w", rowsErr)
	}

	if closeErr := rows.Close(); closeErr != nil {
		return nil, fmt.Errorf("close terminal events for sanitization: %w", closeErr)
	}

	return events, nil
}

func sanitizeStoredPayloadJSON(raw string, secrets []string) string {
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return `{"redacted":true,"reason":"invalid legacy audit payload"}`
	}

	encoded, err := json.Marshal(sanitizeStoredAuditValue(value, secrets))
	if err != nil {
		return `{"redacted":true,"reason":"audit payload could not be sanitized"}`
	}

	return string(encoded)
}

func sanitizeStoredAuditValue(value any, secrets []string) any {
	switch typed := value.(type) {
	case string:
		return redactKnownConfigSecrets(typed, secrets)
	case []any:
		out := make([]any, len(typed))
		for index, entry := range typed {
			out[index] = sanitizeStoredAuditValue(entry, secrets)
		}

		return out
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, entry := range typed {
			switch normalizeConfigKey(key) {
			case "auth":
				out[key] = summarizeStoredAuth(entry)
			case "inputvalues":
				out[key] = redactStoredInputValues(entry)
			case "prescanactions":
				out[key] = map[string]any{"configured": true, "redacted": true}
			case "contentb64", "password", "passwd", "username":
				out[key] = redactedConfigValue
			default:
				out[key] = sanitizeStoredAuditValue(entry, secrets)
			}
		}

		return out
	default:
		return value
	}
}

func summarizeStoredAuth(value any) map[string]any {
	summary := map[string]any{"configured": true, "redacted": true}

	if auth, isAuth := value.(map[string]any); isAuth {
		if mode, hasMode := auth["mode"].(string); hasMode && mode != "" {
			summary["mode"] = mode
		}
	}

	return summary
}

func redactStoredInputValues(value any) map[string]any {
	redacted := map[string]any{"redacted": true}

	if values, ok := value.(map[string]any); ok {
		for key := range values {
			redacted[key] = redactedConfigValue
		}
	}

	return redacted
}
