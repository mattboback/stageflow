package jobstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const backupInterrupted = "This job was interrupted in a sanitized backup; authentication and running containers are not restored."

// SanitizeBackup scrubs an isolated restored snapshot. It must never be called
// against the live queue. The backup command permits only a private Unix socket.
func (d *Database) SanitizeBackup(ctx context.Context) error {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollback(tx)

	if err = checkBackupSchema(ctx, tx); err != nil {
		return err
	}

	if _, err = tx.ExecContext(ctx, "LOCK TABLE jobs, job_scanners IN ACCESS EXCLUSIVE MODE"); err != nil {
		return err
	}

	jobs, secrets, err := loadBackupJobs(ctx, tx)
	if err != nil {
		return err
	}

	for _, j := range jobs {
		config, jsonErr := sanitizedJSON(j.config, secrets)
		if jsonErr != nil {
			return jsonErr
		}

		urls, jsonErr := sanitizedJSON(j.urls, secrets)
		if jsonErr != nil {
			return jsonErr
		}

		if _, err = tx.ExecContext(
			ctx,
			"UPDATE jobs SET config_json=$2,urls=$3 WHERE id=$1",
			j.id,
			config,
			urls,
		); err != nil {
			return err
		}
	}

	if err = scrubBackupText(ctx, tx, secrets); err != nil {
		return err
	}

	if _, err = tx.ExecContext(
		ctx,
		`UPDATE job_scanners SET state='failed',error=$1 WHERE state IN ('pending','running')`,
		backupInterrupted,
	); err != nil {
		return err
	}

	if _, err = tx.ExecContext(
		ctx,
		`UPDATE jobs SET state='FAILED',error=$1,error_details='',completed_at=COALESCE(completed_at,CURRENT_TIMESTAMP) WHERE state NOT IN ('DONE','FAILED')`,
		backupInterrupted,
	); err != nil {
		return err
	}

	if _, err = tx.ExecContext(
		ctx,
		`UPDATE jobs SET pod_id='',cleanup_pending=FALSE; UPDATE job_scanners SET container_id=''`,
	); err != nil {
		return err
	}

	return tx.Commit()
}

func scrubBackupText(ctx context.Context, tx *sql.Tx, secrets []string) error {
	// Scrub all textual metadata, including URLs and paths which can echo another
	// job's credentials. SQL identifiers are fixed by the schema contract below.
	for table, columns := range backupTextColumns {
		for _, column := range strings.Split(columns, ",") {
			for _, secret := range encodedSecretRepresentations(secrets) {
				if _, err := tx.ExecContext(
					ctx,
					fmt.Sprintf("UPDATE %s SET %s=replace(%s,$1,$2)", table, column, column),
					secret,
					redactedConfigValue,
				); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

type backupJob struct{ id, config, urls string }

func loadBackupJobs(ctx context.Context, tx *sql.Tx) ([]backupJob, []string, error) {
	rows, err := tx.QueryContext(ctx, "SELECT id,config_json,urls FROM jobs ORDER BY id")
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var (
		jobs    []backupJob
		secrets []string
	)

	for rows.Next() {
		var (
			j   backupJob
			raw string
		)
		if err = rows.Scan(&j.id, &raw, &j.urls); err != nil {
			return nil, nil, err
		}

		var values []string

		j.config, values, err = sanitizeTerminalJobConfig(raw)
		if err != nil || j.config == "null" {
			return nil, nil, errors.New("backup contains unreadable job configuration")
		}

		secrets = append(secrets, values...)
		jobs = append(jobs, j)
	}

	err = rows.Err()
	if err != nil {
		return nil, nil, err
	}

	secrets = uniqueSensitiveValues(secrets)

	return jobs, secrets, nil
}

var backupTextColumns = map[string]string{
	"jobs":         "input_path,error,error_details,last_stage,report_json_key",
	"job_scanners": "error,results_path",
}

func sanitizedJSON(raw string, secrets []string) (string, error) {
	var value any
	// Both inputs have already been parsed by PostgreSQL as JSONB.
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return "", errors.New("invalid database JSONB")
	}

	value = redactKnownConfigSecretsInValue(value, secrets)

	encoded, err := json.Marshal(value)
	if err != nil {
		return "", errors.New("invalid sanitized JSON value")
	}

	return string(encoded), nil
}

func checkBackupSchema(ctx context.Context, tx *sql.Tx) error {
	expected := map[string]map[string]bool{}
	for table, columns := range map[string]string{
		"jobs":         jobColumns,
		"job_scanners": "job_id,scanner,state,attempts,container_id,pages_scanned,total_issues,critical_issues,serious_issues,moderate_issues,minor_issues,info_issues,timing_ms,error,results_path",
	} {
		expected[table] = map[string]bool{}
		for _, column := range strings.Split(columns, ",") {
			expected[table][column] = true
		}
	}

	rows, err := tx.QueryContext(
		ctx,
		`SELECT table_schema,table_name,column_name FROM information_schema.columns WHERE table_schema NOT IN ('pg_catalog','information_schema') AND table_schema NOT LIKE 'pg_toast%'`,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var schema, table, column string
		if err = rows.Scan(&schema, &table, &column); err != nil {
			return err
		}

		if schema != "public" || !expected[table][column] {
			return errors.New("backup schema differs from the reviewed jobs schema")
		}

		delete(expected[table], column)
	}

	if err = rows.Err(); err != nil {
		return err
	}

	for _, columns := range expected {
		if len(columns) > 0 {
			return errors.New("backup schema is incomplete")
		}
	}

	return nil
}
