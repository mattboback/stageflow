package messaging

import (
	"context"
	"errors"
	"time"
)

// Delivery disposition for messages whose handler returned an error.
//
// Every durable consumer is created with MaxDeliver, after which JetStream stops
// redelivering. Until now the subscriber NAK'd unconditionally, so a permanent
// failure simply stopped happening: no log at the boundary, no advisory, and any
// state the handler had not finished writing stayed as it was. A production job
// spent its whole delivery budget failing and was left in PENDING indefinitely.
//
// Two things address that here, and neither changes behavior for a handler that
// does nothing:
//
//   - The final delivery is terminated with a reason instead of NAK'd. That is a
//     no-op as far as redelivery goes -- JetStream was going to drop it anyway --
//     but it is loud, and the server publishes the reason on its terminated
//     advisory subject.
//   - Handlers that can record their own terminal outcome can ask whether they
//     are on the last attempt (IsFinalDelivery) or declare an error permanent
//     (Terminal), rather than burning the remaining budget on a retry that cannot
//     succeed.

// MaxDeliver is the delivery ceiling every durable consumer is created with.
// Exported so handlers can reason about their own retry budget.
const MaxDeliver = defaultConsumerMaxDeliver

// maxConsumerNAKDelay caps the escalating redelivery delay.
const maxConsumerNAKDelay = 2 * time.Minute

// terminalError marks an error as permanent.
type terminalError struct{ err error }

func (e *terminalError) Error() string { return e.err.Error() }
func (e *terminalError) Unwrap() error { return e.err }

// Terminal marks err as permanent, so the subscriber terminates the message
// instead of NAK-ing it for redelivery.
//
// Use it when retrying provably cannot help -- a malformed payload, or work whose
// terminal outcome the handler has already recorded. Retryable failures should be
// returned bare so they keep their delivery budget.
func Terminal(err error) error {
	if err == nil {
		return nil
	}

	return &terminalError{err: err}
}

// IsTerminal reports whether err was marked permanent with Terminal.
func IsTerminal(err error) bool {
	var terminal *terminalError

	return errors.As(err, &terminal)
}

// IsFinalDelivery reports whether the message being handled is on its last
// attempt, so a handler can record a terminal outcome instead of letting the
// message be dropped with the work half-done.
//
// It reports false when no delivery metadata is present, which is the right
// default: reconciliation sweeps and unit tests call the same handlers outside
// any delivery, and they should take the retryable path.
func IsFinalDelivery(ctx context.Context) bool {
	meta, ok := ReceivedEventMetaFromContext(ctx)
	if !ok || meta == nil {
		return false
	}

	return meta.Deliveries >= MaxDeliver
}

// nakDelay spaces redeliveries so a failing dependency gets time to recover.
//
// JetStream's own BackOff is not usable for this. It is ignored for messages that
// were explicitly NAK'd -- the server applies the NAK's delay and returns before
// reaching the backoff ladder -- and setting it overrides AckWait for the whole
// consumer, which would shrink the ack window every handler relies on. Computing
// the delay here avoids both problems.
//
// 5s, 10s, 20s, 40s, 80s, then 120s, turning a ten-delivery budget from about
// forty-five seconds into roughly thirteen minutes.
func nakDelay(deliveries uint64) time.Duration {
	delay := defaultConsumerNAKDelay
	if deliveries < 2 {
		return delay
	}

	for i := uint64(1); i < deliveries; i++ {
		delay *= 2
		if delay >= maxConsumerNAKDelay {
			return maxConsumerNAKDelay
		}
	}

	return delay
}

// disposition is the decision made about a failed message, split out from the
// JetStream types so it can be tested directly.
type disposition int

const (
	dispositionRetry disposition = iota
	dispositionTerminate
)

// decideDisposition terminates on the final delivery or on an explicitly terminal
// error, and retries otherwise.
func decideDisposition(deliveries uint64, handleErr error) disposition {
	if IsTerminal(handleErr) {
		return dispositionTerminate
	}

	// deliveries is 0 when metadata is unavailable; treat that as retryable rather
	// than guessing that the budget is spent.
	if deliveries >= MaxDeliver {
		return dispositionTerminate
	}

	return dispositionRetry
}
