package messaging

import (
	"context"
	"time"
)

// Failed deliveries retry with an escalating delay until MaxDeliver. The final
// delivery is terminated with a reason so exhausted jobs do not disappear
// silently. Handlers can use IsFinalDelivery to persist their terminal outcome.

// MaxDeliver is the delivery ceiling every durable consumer is created with.
// Exported so handlers can reason about their own retry budget.
const MaxDeliver = defaultConsumerMaxDeliver

// maxConsumerNAKDelay caps the escalating redelivery delay.
const maxConsumerNAKDelay = 2 * time.Minute

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

// decideDisposition terminates on the final delivery and retries otherwise.
func decideDisposition(deliveries uint64) disposition {
	// deliveries is 0 when metadata is unavailable; treat that as retryable rather
	// than guessing that the budget is spent.
	if deliveries >= MaxDeliver {
		return dispositionTerminate
	}

	return dispositionRetry
}
