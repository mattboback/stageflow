package main

import (
	"context"
	"errors"
	"os"
	"testing"
)

type orderedMonitorWaiter struct{ order *[]string }

func (s *orderedMonitorWaiter) WaitForMonitors() {
	*s.order = append(*s.order, "monitors")
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
		&orderedMonitorWaiter{order: &order},
	)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("shutdownService() error = %v, want %v", err, expectedErr)
	}

	want := []string{"cancel", "api", "monitors"}
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

	if checkErr1 := waitForShutdown(signals, apiErrors); !errors.Is(checkErr1, expected) {
		t.Fatalf("waitForShutdown() error = %v, want %v", checkErr1, expected)
	}
}

func TestWaitForShutdownAcceptsSignal(t *testing.T) {
	t.Parallel()

	signals := make(chan os.Signal, 1)
	apiErrors := make(chan error)

	signals <- os.Interrupt

	if checkErr2 := waitForShutdown(signals, apiErrors); checkErr2 != nil {
		t.Fatalf("waitForShutdown() error = %v, want nil", checkErr2)
	}
}

func TestWaitForShutdownRejectsUnexpectedCleanAPIStop(t *testing.T) {
	t.Parallel()

	signals := make(chan os.Signal)

	apiErrors := make(chan error, 1)
	apiErrors <- nil

	if checkErr3 := waitForShutdown(signals, apiErrors); checkErr3 == nil {
		t.Fatal("waitForShutdown() error = nil, want unexpected-stop error")
	}
}
