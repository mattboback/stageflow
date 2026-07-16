package main

import (
	"context"
	"errors"
	"os"
	"testing"
)

type orderedStarter struct {
	order *[]string
	err   error
}

func (s *orderedStarter) Start(context.Context) {
	*s.order = append(*s.order, "reconcile")
}

func (s *orderedStarter) StartConsumer(context.Context) error {
	*s.order = append(*s.order, "consume")

	return s.err
}

type orderedConsumer struct{ *orderedStarter }

func (c *orderedConsumer) Start(ctx context.Context) error {
	return c.StartConsumer(ctx)
}

type orderedConsumerStopper struct{ order *[]string }

func (s *orderedConsumerStopper) StopConsumers() {
	*s.order = append(*s.order, "consumers")
}

type orderedDatabaseStopper struct{ order *[]string }

func (s *orderedDatabaseStopper) StopBackgroundTasks() {
	*s.order = append(*s.order, "database")
}

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

func TestStartEventProcessingReconcilesBeforeConsumer(t *testing.T) {
	t.Parallel()

	var order []string

	starter := &orderedStarter{order: &order}
	consumer := &orderedConsumer{orderedStarter: starter}

	if err := startEventProcessing(context.Background(), starter, consumer); err != nil {
		t.Fatalf("startEventProcessing() error = %v", err)
	}

	if got := len(order); got != 2 || order[0] != "reconcile" || order[1] != "consume" {
		t.Fatalf("startup order = %v, want [reconcile consume]", order)
	}
}

func TestStopEventProcessingDrainsDependenciesInOrder(t *testing.T) {
	t.Parallel()

	var order []string

	cancel := func() { order = append(order, "cancel") }

	stopEventProcessing(
		cancel,
		&orderedConsumerStopper{order: &order},
		&orderedDatabaseStopper{order: &order},
		&orderedMonitorWaiter{order: &order},
	)

	want := []string{"cancel", "consumers", "database", "monitors"}
	if len(order) != len(want) {
		t.Fatalf("shutdown order = %v, want %v", order, want)
	}

	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("shutdown order = %v, want %v", order, want)
		}
	}
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
		&orderedConsumerStopper{order: &order},
		&orderedDatabaseStopper{order: &order},
		&orderedMonitorWaiter{order: &order},
	)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("shutdownService() error = %v, want %v", err, expectedErr)
	}

	want := []string{"cancel", "api", "consumers", "database", "monitors"}
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
