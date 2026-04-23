package messaging

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/mattboback/stageflow/libs/go/logging"
)

func (c *Client) Subscribe(
	ctx context.Context,
	stream, subject, consumerName string,
	handler func([]byte) error,
) error {
	if _, err := c.snapshot(); err != nil {
		return err
	}

	if ctx == nil {
		return ErrNilContext
	}

	if handler == nil {
		return ErrNilHandler
	}

	cons, err := c.createOrRefreshConsumer(ctx, stream, subject, consumerName)
	if err != nil {
		return err
	}

	consumeCtx, err := cons.Consume(func(msg jetstream.Msg) {
		if handleErr := handler(msg.Data()); handleErr != nil {
			slog.Error("Error handling message",
				"stream", stream,
				"subject", subject,
				"consumer", consumerName,
				"error", handleErr,
			)

			if nakErr := msg.NakWithDelay(defaultConsumerNAKDelay); nakErr != nil {
				slog.Warn("Failed to NAK message",
					"stream", stream,
					"subject", subject,
					"consumer", consumerName,
					"error", nakErr,
				)
			}

			return
		}

		if ackErr := msg.Ack(); ackErr != nil {
			slog.Warn("Failed to acknowledge message",
				"stream", stream,
				"subject", subject,
				"consumer", consumerName,
				"error", ackErr,
			)
		}
	})
	if err != nil {
		return err
	}

	c.trackConsumeContext(stream, consumerName, consumeCtx)

	// Ensure consumption stops when the context is canceled (avoid leaking consumers).
	if done := ctx.Done(); done != nil {
		go func() {
			<-done
			consumeCtx.Stop()
			c.untrackConsumeContext(stream, consumerName, consumeCtx)
		}()
	}

	return nil
}

func (c *Client) SubscribeWithContext(
	ctx context.Context,
	stream, subject, consumerName string,
	handler func(context.Context, jetstream.Msg) error,
) error {
	if _, err := c.snapshot(); err != nil {
		return err
	}

	if ctx == nil {
		return ErrNilContext
	}

	if handler == nil {
		return ErrNilHandler
	}

	cons, err := c.createOrRefreshConsumer(ctx, stream, subject, consumerName)
	if err != nil {
		return err
	}

	consumeCtx, err := cons.Consume(func(msg jetstream.Msg) {
		if handleErr := handler(ctx, msg); handleErr != nil {
			slog.Error("Error handling message",
				"stream", stream,
				"subject", subject,
				"consumer", consumerName,
				"error", handleErr,
			)

			if nakErr := msg.NakWithDelay(defaultConsumerNAKDelay); nakErr != nil {
				slog.Warn("Failed to NAK message",
					"stream", stream,
					"subject", subject,
					"consumer", consumerName,
					"error", nakErr,
				)
			}

			return
		}

		if ackErr := msg.Ack(); ackErr != nil {
			slog.Warn("Failed to acknowledge message",
				"stream", stream,
				"subject", subject,
				"consumer", consumerName,
				"error", ackErr,
			)
		}
	})
	if err != nil {
		return err
	}

	c.trackConsumeContext(stream, consumerName, consumeCtx)

	if done := ctx.Done(); done != nil {
		go func() {
			<-done
			consumeCtx.Stop()
			c.untrackConsumeContext(stream, consumerName, consumeCtx)
		}()
	}

	return nil
}

type Subscription[T any] struct {
	Stream  string
	Subject string
	Durable string
	Handler func(context.Context, *T) error
}

func SubscribeTyped[T any](ctx context.Context, client *Client, sub Subscription[T]) error {
	if client == nil {
		return ErrNilClient
	}

	if sub.Stream == "" || sub.Subject == "" || sub.Durable == "" {
		return fmt.Errorf("%w: stream/subject/durable are required", ErrBadSubscription)
	}

	if sub.Handler == nil {
		return ErrNilHandler
	}

	return client.SubscribeWithContext(ctx, sub.Stream, sub.Subject, sub.Durable,
		func(ctx context.Context, msg jetstream.Msg) error {
			var envelope struct {
				Event     string          `json:"event"`
				JobID     string          `json:"job_id"`
				RequestID string          `json:"request_id,omitempty"`
				RunID     string          `json:"run_id,omitempty"`
				Timestamp time.Time       `json:"timestamp"`
				Producer  string          `json:"producer"`
				Payload   json.RawMessage `json:"payload"`
			}

			// Allow additive fields in the envelope for forward-compatible event evolution.
			if err := unmarshalLenient(msg.Data(), &envelope); err != nil {
				return fmt.Errorf("unmarshal envelope: %w", err)
			}

			var payload T
			if err := unmarshalStrict(envelope.Payload, &payload); err != nil {
				return fmt.Errorf("unmarshal payload: %w", err)
			}

			meta := &ReceivedEventMeta{
				Event:             envelope.Event,
				JobID:             envelope.JobID,
				RequestID:         envelope.RequestID,
				RunID:             envelope.RunID,
				Producer:          envelope.Producer,
				EnvelopeTimestamp: envelope.Timestamp,
				Subject:           msg.Subject(),
			}

			if jsMeta, err := msg.Metadata(); err == nil && jsMeta != nil {
				meta.Stream = jsMeta.Stream
				meta.Consumer = jsMeta.Consumer
				meta.StreamSeq = jsMeta.Sequence.Stream
				meta.ConsumerSeq = jsMeta.Sequence.Consumer
				meta.Deliveries = jsMeta.NumDelivered
				meta.Pending = jsMeta.NumPending
				meta.StoredAt = jsMeta.Timestamp
			} else {
				meta.Stream = sub.Stream
				meta.Consumer = sub.Durable
			}

			// Attach common logging context.
			ctx = logging.WithJobID(ctx, envelope.JobID)
			if envelope.RequestID != "" {
				ctx = logging.WithRequestID(ctx, envelope.RequestID)
			}

			if envelope.RunID != "" {
				ctx = logging.WithRunID(ctx, envelope.RunID)
			}

			ctx = WithReceivedEventMeta(ctx, meta)

			if err := sub.Handler(ctx, &payload); err != nil {
				logging.Error(ctx, "Event handler failed",
					"event", meta.Event,
					"producer", meta.Producer,
					"subject", meta.Subject,
					"stream", meta.Stream,
					"consumer", meta.Consumer,
					"deliveries", meta.Deliveries,
					"stream_seq", meta.StreamSeq,
					"consumer_seq", meta.ConsumerSeq,
					"error", err,
				)

				return err
			}

			return nil
		},
	)
}

func unmarshalStrict(data []byte, target any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()

	if err := dec.Decode(target); err != nil {
		return err
	}

	// Ensure exactly one JSON value (whitespace allowed).
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("unexpected trailing JSON")
		}

		return err
	}

	return nil
}

func unmarshalLenient(data []byte, target any) error {
	dec := json.NewDecoder(bytes.NewReader(data))

	if err := dec.Decode(target); err != nil {
		return err
	}

	// Ensure exactly one JSON value (whitespace allowed).
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("unexpected trailing JSON")
		}

		return err
	}

	return nil
}
