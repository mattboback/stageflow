package db

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

func TestReconcileLeaseExcludesConcurrentCallbacks(t *testing.T) {
	d := setupTestDB(t)
	ctx := context.Background()
	entered := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan error, 1)

	go func() {
		finished <- d.WithReconcileLease(ctx, func(context.Context) error {
			close(entered)
			<-release

			return nil
		})
	}()

	<-entered

	var competingCalls atomic.Int32

	competingErr := d.WithReconcileLease(ctx, func(context.Context) error {
		competingCalls.Add(1)
		return nil
	})

	close(release)

	if firstErr := <-finished; firstErr != nil {
		t.Fatal(firstErr)
	}

	if competingErr != nil || competingCalls.Load() != 0 {
		t.Fatalf("competing scheduler entered: calls=%d err=%v", competingCalls.Load(), competingErr)
	}

	if retryErr := d.WithReconcileLease(ctx, func(context.Context) error {
		competingCalls.Add(1)
		return nil
	}); retryErr != nil || competingCalls.Load() != 1 {
		t.Fatalf("released lease unavailable: calls=%d err=%v", competingCalls.Load(), retryErr)
	}
}

func TestReconcileLeaseUnlocksAfterCanceledTick(t *testing.T) {
	d := setupTestDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	callbackErr := errors.New("tick failed")

	leaseErr := d.WithReconcileLease(ctx, func(context.Context) error {
		cancel()
		return callbackErr
	})
	if !errors.Is(leaseErr, callbackErr) {
		t.Fatalf("callback error lost: %v", leaseErr)
	}

	var called bool
	if retryErr := d.WithReconcileLease(context.Background(), func(context.Context) error {
		called = true
		return nil
	}); retryErr != nil || !called {
		t.Fatalf("canceled tick retained lease: called=%v err=%v", called, retryErr)
	}
}

func TestReconcileLeaseDiscardsLostSession(t *testing.T) {
	d := setupTestDB(t)
	ctx := context.Background()

	leaseErr := d.WithReconcileLease(ctx, func(context.Context) error {
		var terminated bool

		terminateErr := d.db.QueryRowContext(ctx, `SELECT pg_terminate_backend(pid) FROM pg_locks WHERE locktype='advisory' AND classid=0 AND objid=$1 AND granted`, reconcileLeaseLock).
			Scan(&terminated)
		if terminateErr != nil {
			return terminateErr
		}

		if !terminated {
			return errors.New("lease session was not terminated")
		}

		return nil
	})
	if leaseErr == nil {
		t.Fatal("lost lease session was not reported")
	}

	var called bool
	if retryErr := d.WithReconcileLease(
		ctx,
		func(context.Context) error { called = true; return nil },
	); retryErr != nil ||
		!called {
		t.Fatalf("lost session poisoned pool: called=%v err=%v", called, retryErr)
	}
}
