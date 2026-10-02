package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/models"
)

// ErrIntakeUncertain means database failure prevented a safe rollback. Uploaded
// files must be preserved because a delivered message may still run this job.
var ErrIntakeUncertain = errors.New("job intake outcome is uncertain")

// PublishPendingJob commits readable status before publishing. The delivery-side
// advisory lock keeps orchestration behind this handoff, including lost publish
// acknowledgements. This bridge disappears with transactional queue admission.
func (d *Database) PublishPendingJob(
	ctx context.Context,
	job *models.Job,
	publish func(context.Context) error,
) (err error) {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	conn, err := d.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err = conn.ExecContext(ctx, "SELECT pg_advisory_lock(hashtextextended($1,0))", job.ID); err != nil {
		// A lost response may still have acquired the session lock.
		discardIntakeConnection(conn)
		return err
	}

	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if _, unlockErr := conn.ExecContext(
			unlockCtx,
			"SELECT pg_advisory_unlock(hashtextextended($1,0))",
			job.ID,
		); unlockErr != nil {
			discardIntakeConnection(conn)

			err = errors.Join(err, fmt.Errorf("%w: release intake lock: %w", ErrIntakeUncertain, unlockErr))
		}
	}()

	config, err := json.Marshal(job.Config)
	if err != nil {
		return err
	}

	urls, err := json.Marshal(job.URLs)
	if err != nil {
		return err
	}

	scanners, err := json.Marshal(job.ExpectedScanners)
	if err != nil {
		return err
	}

	_, err = conn.ExecContext(
		ctx,
		`INSERT INTO jobs(id,state,input_type,input_path,urls,config_json,created_at,updated_at,expected_scanners) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		job.ID,
		job.State,
		job.InputType,
		job.InputPath,
		string(urls),
		string(config),
		job.CreatedAt,
		job.UpdatedAt,
		string(scanners),
	)
	if err == nil {
		err = ctx.Err()
	}

	if err == nil {
		err = publish(ctx)
	}

	if err != nil {
		return failIntake(conn, job.ID, err)
	}

	return nil
}

func discardIntakeConnection(conn *sql.Conn) {
	// ErrBadConn forces database/sql to close this session rather than returning a
	// possibly locked connection to the pool.
	if rawErr := conn.Raw(
		func(any) error { return driver.ErrBadConn },
	); rawErr != nil && !errors.Is(rawErr, driver.ErrBadConn) &&
		!errors.Is(rawErr, sql.ErrConnDone) {
		slog.Warn("Discard intake connection", "error", rawErr)
	}
}

func failIntake(conn *sql.Conn, id string, cause error) error {
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	payload := &events.JobFailedPayload{JobID: id, Status: "failed", Stage: "queue", Error: "Failed to queue job"}
	if _, err := failJobWithTerminalEvent(cleanupCtx, conn, id, "queue", payload.Error, "", payload); err != nil {
		return fmt.Errorf("%w: %w", ErrIntakeUncertain, err)
	}

	return cause
}
