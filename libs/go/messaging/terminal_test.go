package messaging

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestTerminalWrapsAndUnwraps(t *testing.T) {
	t.Parallel()

	base := errors.New("malformed payload")
	wrapped := Terminal(base)

	if !IsTerminal(wrapped) {
		t.Fatal("expected the wrapped error to be reported as terminal")
	}

	if !errors.Is(wrapped, base) {
		t.Fatal("expected the original error to remain unwrappable")
	}

	if wrapped.Error() != base.Error() {
		t.Fatalf("Error() = %q, want the original message %q", wrapped.Error(), base.Error())
	}

	if IsTerminal(base) {
		t.Fatal("a bare error must not be treated as terminal")
	}

	if Terminal(nil) != nil {
		t.Fatal("Terminal(nil) must stay nil so handlers can wrap unconditionally")
	}
}

func TestIsTerminalFindsAWrappedTerminalError(t *testing.T) {
	t.Parallel()

	// Handlers wrap errors on the way out; the marker has to survive that.
	err := errors.Join(errors.New("context"), Terminal(errors.New("permanent")))
	if !IsTerminal(err) {
		t.Fatal("expected a terminal error nested in a chain to be found")
	}
}

func TestIsFinalDelivery(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name       string
		deliveries uint64
		withMeta   bool
		want       bool
	}{
		{name: "no metadata at all", withMeta: false, want: false},
		{name: "first attempt", withMeta: true, deliveries: 1, want: false},
		{name: "one before the ceiling", withMeta: true, deliveries: MaxDeliver - 1, want: false},
		{name: "at the ceiling", withMeta: true, deliveries: MaxDeliver, want: true},
		{name: "past the ceiling", withMeta: true, deliveries: MaxDeliver + 1, want: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			if testCase.withMeta {
				ctx = WithReceivedEventMeta(ctx, &ReceivedEventMeta{Deliveries: testCase.deliveries})
			}

			if got := IsFinalDelivery(ctx); got != testCase.want {
				t.Fatalf("IsFinalDelivery = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestIsFinalDeliveryDefaultsToFalseOutsideADelivery(t *testing.T) {
	t.Parallel()

	// Reconciliation sweeps and unit tests invoke the same handlers with no
	// delivery metadata. They must take the retryable path, not the give-up path.
	if IsFinalDelivery(context.Background()) {
		t.Fatal("expected false without delivery metadata")
	}
}

func TestNakDelayEscalatesAndCaps(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		deliveries uint64
		want       time.Duration
	}{
		{deliveries: 0, want: 5 * time.Second},
		{deliveries: 1, want: 5 * time.Second},
		{deliveries: 2, want: 10 * time.Second},
		{deliveries: 3, want: 20 * time.Second},
		{deliveries: 4, want: 40 * time.Second},
		{deliveries: 5, want: 80 * time.Second},
		{deliveries: 6, want: maxConsumerNAKDelay},
		{deliveries: 10, want: maxConsumerNAKDelay},
		{deliveries: 100, want: maxConsumerNAKDelay},
	} {
		if got := nakDelay(testCase.deliveries); got != testCase.want {
			t.Errorf("nakDelay(%d) = %v, want %v", testCase.deliveries, got, testCase.want)
		}
	}
}

func TestNakDelayGivesTheBudgetRoomToRecover(t *testing.T) {
	t.Parallel()

	// The point of escalating: a flat 5s delay spent the whole ten-delivery budget
	// in under a minute, which is shorter than most transient outages.
	var total time.Duration
	for i := uint64(1); i < MaxDeliver; i++ {
		total += nakDelay(i)
	}

	if total < 10*time.Minute {
		t.Fatalf("total retry window = %v, want at least 10m", total)
	}
}

func TestDecideDisposition(t *testing.T) {
	t.Parallel()

	retryable := errors.New("podman socket unavailable")

	for _, testCase := range []struct {
		name       string
		deliveries uint64
		err        error
		want       disposition
	}{
		{name: "retryable, budget left", deliveries: 1, err: retryable, want: dispositionRetry},
		{name: "retryable, one left", deliveries: MaxDeliver - 1, err: retryable, want: dispositionRetry},
		{name: "retryable, budget spent", deliveries: MaxDeliver, err: retryable, want: dispositionTerminate},
		{name: "terminal on first attempt", deliveries: 1, err: Terminal(retryable), want: dispositionTerminate},
		{name: "no metadata is retryable", deliveries: 0, err: retryable, want: dispositionRetry},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := decideDisposition(testCase.deliveries, testCase.err); got != testCase.want {
				t.Fatalf("decideDisposition(%d) = %v, want %v", testCase.deliveries, got, testCase.want)
			}
		})
	}
}
