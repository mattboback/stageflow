package messaging

import (
	"context"
	"fmt"
	"time"
)

type receivedMetaKey struct{}

type ReceivedEventMeta struct {
	Event             string
	JobID             string
	RequestID         string
	RunID             string
	Producer          string
	EnvelopeTimestamp time.Time

	Subject     string
	Stream      string
	Consumer    string
	StreamSeq   uint64
	ConsumerSeq uint64
	Deliveries  uint64
	Pending     uint64
	StoredAt    time.Time
}

func WithReceivedEventMeta(ctx context.Context, meta *ReceivedEventMeta) context.Context {
	if meta == nil {
		return ctx
	}

	return context.WithValue(ctx, receivedMetaKey{}, meta)
}

func ReceivedEventMetaFromContext(ctx context.Context) (*ReceivedEventMeta, bool) {
	if v := ctx.Value(receivedMetaKey{}); v != nil {
		if meta, ok := v.(*ReceivedEventMeta); ok {
			return meta, true
		}
	}

	return nil, false
}

func validatePublishEventEnvelope(envelope any) error {
	v, ok := envelope.(interface{ Validate() error })
	if !ok {
		return fmt.Errorf("%w: envelope must implement Validate()", ErrBadEnvelope)
	}

	if err := v.Validate(); err != nil {
		return fmt.Errorf("%w: %w", ErrBadEnvelope, err)
	}

	return nil
}
