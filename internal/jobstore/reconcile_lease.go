package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"time"
)

const reconcileLeaseLock = 1937006951

// WithReconcileLease runs one scheduler tick under a PostgreSQL session lock.
// A competing scheduler skips its tick instead of consuming launch attempts.
func (d *Database) WithReconcileLease(ctx context.Context, reconcile func(context.Context) error) (resultErr error) {
	conn, err := d.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("open reconciliation lease connection: %w", err)
	}

	defer func() {
		if closeErr := conn.Close(); closeErr != nil && !errors.Is(closeErr, sql.ErrConnDone) {
			resultErr = errors.Join(resultErr, closeErr)
		}
	}()

	var acquired bool
	if err = conn.QueryRowContext(ctx, `SELECT pg_try_advisory_lock($1)`, reconcileLeaseLock).
		Scan(&acquired); err != nil {
		// A canceled query can leave acquisition ambiguous; discard that session.
		return fmt.Errorf("acquire reconciliation lease: %w", errors.Join(err, discardLeaseConnection(conn)))
	}

	if !acquired {
		return nil
	}

	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var unlocked bool

		unlockErr := conn.QueryRowContext(unlockCtx, `SELECT pg_advisory_unlock($1)`, reconcileLeaseLock).
			Scan(&unlocked)
		if unlockErr == nil && unlocked {
			return
		}

		if unlockErr == nil {
			unlockErr = errors.New("reconciliation lease was not held")
		}

		// Never return a potentially locked PostgreSQL session to the pool.
		unlockErr = errors.Join(unlockErr, discardLeaseConnection(conn))

		resultErr = errors.Join(resultErr, fmt.Errorf("release reconciliation lease: %w", unlockErr))
	}()

	return reconcile(ctx)
}

func discardLeaseConnection(conn *sql.Conn) error {
	err := conn.Raw(func(any) error { return driver.ErrBadConn })
	if errors.Is(err, driver.ErrBadConn) || errors.Is(err, sql.ErrConnDone) {
		return nil
	}

	return err
}
