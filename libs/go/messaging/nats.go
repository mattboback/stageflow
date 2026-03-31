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

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/mattboback/stageflow/libs/go/config"
	"github.com/mattboback/stageflow/libs/go/logging"
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

const (
	StreamJobs       = "jobs"
	StreamExtraction = "extraction"
	StreamScan       = "scan"
)

const (
	defaultStreamMaxAge       = 72 * time.Hour
	defaultConsumerAckWait    = 10 * time.Minute
	defaultConsumerMaxDeliver = 10
	defaultConsumerNAKDelay   = 5 * time.Second
)

const (
	SubjectJobCreated        = "jobs.events.created"
	SubjectJobCompleted      = "jobs.events.completed"
	SubjectJobFailed         = "jobs.events.failed"
	SubjectExtractionReady   = "extraction.events.ready"
	SubjectExtractionFailed  = "extraction.events.failed"
	SubjectScanPageCompleted = "scan.events.page.completed"
	SubjectScanCompleted     = "scan.events.completed"
	SubjectScanFailed        = "scan.events.failed"
)

type Client struct {
	nc *nats.Conn
	js jetstream.JetStream
}

type Config = config.NATSConfig

var (
	ErrNilClient       = errors.New("messaging: client is nil")
	ErrNilContext      = errors.New("messaging: context is nil")
	ErrNotConnected    = errors.New("messaging: client is not connected")
	ErrNilHandler      = errors.New("messaging: handler is nil")
	ErrBadSubscription = errors.New("messaging: invalid subscription")
	ErrBadEnvelope     = errors.New("messaging: invalid envelope")
)

func DefaultConfig() *Config {
	return &Config{
		URL:            nats.DefaultURL,
		MaxReconnects:  10,
		ReconnectWait:  2 * time.Second,
		ConnectTimeout: 10 * time.Second,
	}
}

func NewClient(cfg *Config) (*Client, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Treat zero-values as "not set" (common when config is partially specified).
	normalized := *cfg
	if normalized.URL == "" {
		normalized.URL = nats.DefaultURL
	}

	if normalized.MaxReconnects == 0 {
		normalized.MaxReconnects = 10
	}

	if normalized.ReconnectWait == 0 {
		normalized.ReconnectWait = 2 * time.Second
	}

	if normalized.ConnectTimeout == 0 {
		normalized.ConnectTimeout = 10 * time.Second
	}

	opts := []nats.Option{
		nats.Name("stageflow"),
		nats.MaxReconnects(normalized.MaxReconnects),
		nats.ReconnectWait(normalized.ReconnectWait),
		nats.Timeout(normalized.ConnectTimeout),

		nats.ErrorHandler(func(_ *nats.Conn, sub *nats.Subscription, err error) {
			// sub can be nil in some error cases
			if sub != nil {
				slog.Error("NATS async error", "subject", sub.Subject, "queue", sub.Queue, "error", err)
			} else {
				slog.Error("NATS async error", "error", err)
			}
		}),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			if err != nil {
				slog.Warn("NATS disconnected", "error", err)
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			slog.Info("NATS reconnected", "url", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			slog.Info("NATS connection closed", "url", nc.ConnectedUrl())
		}),
	}

	nc, err := nats.Connect(normalized.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()

		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	return &Client{
		nc: nc,
		js: js,
	}, nil
}

func (c *Client) ensureReady() error {
	if c == nil {
		return ErrNilClient
	}

	if c.nc == nil || c.nc.IsClosed() {
		return ErrNotConnected
	}

	if c.js == nil {
		return ErrNotConnected
	}

	return nil
}

func (c *Client) Close() error {
	if c == nil {
		return nil
	}

	if c.nc != nil {
		c.nc.Close()
	}

	c.nc = nil
	c.js = nil

	return nil
}

func (c *Client) EnsureStreams(ctx context.Context) error {
	if c == nil {
		return ErrNilClient
	}

	if ctx == nil {
		return ErrNilContext
	}

	if err := c.ensureReady(); err != nil {
		return err
	}

	streams := []struct {
		name     string
		subjects []string
	}{
		{
			name: StreamJobs,
			subjects: []string{
				SubjectJobCreated,
				SubjectJobCompleted,
				SubjectJobFailed,
			},
		},
		{
			name: StreamExtraction,
			subjects: []string{
				SubjectExtractionReady,
				SubjectExtractionFailed,
			},
		},
		{
			name: StreamScan,
			subjects: []string{
				SubjectScanPageCompleted,
				SubjectScanCompleted,
				SubjectScanFailed,
			},
		},
	}

	for _, s := range streams {
		_, err := c.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
			Name:      s.name,
			Subjects:  s.subjects,
			Retention: jetstream.LimitsPolicy,
			MaxAge:    defaultStreamMaxAge,
			Storage:   jetstream.FileStorage,
		})
		if err != nil {
			return fmt.Errorf("failed to create stream %s: %w", s.name, err)
		}
	}

	return nil
}

func (c *Client) Publish(ctx context.Context, subject string, data any) error {
	if err := c.ensureReady(); err != nil {
		return err
	}

	if ctx == nil {
		return ErrNilContext
	}

	if subject == "" {
		return errors.New("messaging: subject is required")
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	_, err = c.js.Publish(ctx, subject, payload)
	if err != nil {
		return fmt.Errorf("failed to publish to %s: %w", subject, err)
	}

	return nil
}

func (c *Client) Subscribe(
	ctx context.Context,
	stream, subject, consumerName string,
	handler func([]byte) error,
) error {
	if err := c.ensureReady(); err != nil {
		return err
	}

	if ctx == nil {
		return ErrNilContext
	}

	if handler == nil {
		return ErrNilHandler
	}

	cons, err := c.js.CreateOrUpdateConsumer(ctx, stream, jetstream.ConsumerConfig{
		Durable:       consumerName,
		FilterSubject: subject,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    defaultConsumerMaxDeliver,
		AckWait:       defaultConsumerAckWait,
	})
	if err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
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

	// Ensure consumption stops when the context is canceled (avoid leaking consumers).
	if done := ctx.Done(); done != nil {
		go func() {
			<-done
			consumeCtx.Stop()
		}()
	}

	return nil
}

func (c *Client) SubscribeWithContext(
	ctx context.Context,
	stream, subject, consumerName string,
	handler func(context.Context, jetstream.Msg) error,
) error {
	if err := c.ensureReady(); err != nil {
		return err
	}

	if ctx == nil {
		return ErrNilContext
	}

	if handler == nil {
		return ErrNilHandler
	}

	cons, err := c.js.CreateOrUpdateConsumer(ctx, stream, jetstream.ConsumerConfig{
		Durable:       consumerName,
		FilterSubject: subject,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    defaultConsumerMaxDeliver,
		AckWait:       defaultConsumerAckWait,
	})
	if err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
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

	if done := ctx.Done(); done != nil {
		go func() {
			<-done
			consumeCtx.Stop()
		}()
	}

	return nil
}

func (c *Client) PublishEvent(ctx context.Context, subject string, envelope any) error {
	if err := c.ensureReady(); err != nil {
		return err
	}

	if ctx == nil {
		return ErrNilContext
	}

	if err := validatePublishEventEnvelope(envelope); err != nil {
		return err
	}

	return c.Publish(ctx, subject, envelope)
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
