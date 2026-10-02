package main

import (
	"context"
	"errors"
	"os"
	"testing"
)

type orderedReconcileWaiter struct{ order *[]string }

func (s *orderedReconcileWaiter) Wait() {
	*s.order = append(*s.order, "reconciler")
}

type orderedAPIShutdowner struct {
	order *[]string
	err   error
}

func (s *orderedAPIShutdowner) Shutdown(context.Context) error {
	*s.order = append(*s.order, "api")

	return s.err
}

func TestShutdownServiceCancelsBeforeHTTPAndJoinsAfterward(t *testing.T) {
	t.Parallel()

	var order []string

	cancel := func() { order = append(order, "cancel") }
	expectedErr := errors.New("HTTP drain timed out")

	err := shutdownService(
		context.Background(),
		cancel,
		&orderedAPIShutdowner{order: &order, err: expectedErr},
		&orderedReconcileWaiter{order: &order},
	)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("shutdownService() error = %v, want %v", err, expectedErr)
	}

	want := []string{"cancel", "api", "reconciler"}
	if len(order) != len(want) {
		t.Fatalf("shutdown order = %v, want %v", order, want)
	}

	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("shutdown order = %v, want %v", order, want)
		}
	}
}

func TestWaitForShutdownReturnsAPIServerFailure(t *testing.T) {
	t.Parallel()

	expected := errors.New("listen tcp: address already in use")
	signals := make(chan os.Signal)

	apiErrors := make(chan error, 1)
	apiErrors <- expected

	if err := waitForShutdown(signals, apiErrors); !errors.Is(err, expected) {
		t.Fatalf("waitForShutdown() error = %v, want %v", err, expected)
	}
}

func TestWaitForShutdownAcceptsSignal(t *testing.T) {
	t.Parallel()

	signals := make(chan os.Signal, 1)
	apiErrors := make(chan error)

	signals <- os.Interrupt

	if err := waitForShutdown(signals, apiErrors); err != nil {
		t.Fatalf("waitForShutdown() error = %v, want nil", err)
	}
}

func TestWaitForShutdownRejectsUnexpectedCleanAPIStop(t *testing.T) {
	t.Parallel()

	signals := make(chan os.Signal)

	apiErrors := make(chan error, 1)
	apiErrors <- nil

	if err := waitForShutdown(signals, apiErrors); err == nil {
		t.Fatal("waitForShutdown() error = nil, want unexpected-stop error")
	}
}
