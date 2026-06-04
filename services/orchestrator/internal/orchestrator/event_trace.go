package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/logging"
	sharedmsg "github.com/mattboback/stageflow/libs/go/messaging"
	db "github.com/mattboback/stageflow/services/orchestrator/internal/adapters/repository"
)

func backgroundWithCorrelation(ctx context.Context) context.Context {
	bg := context.Background()

	if jobID := logging.JobID(ctx); jobID != "" {
		bg = logging.WithJobID(bg, jobID)
	}

	if requestID := logging.RequestID(ctx); requestID != "" {
		bg = logging.WithRequestID(bg, requestID)
	}

	if runID := logging.RunID(ctx); runID != "" {
		bg = logging.WithRunID(bg, runID)
	}

	return bg
}

func marshalPayload(v any) string {
	if v == nil {
		return ""
	}

	b, err := json.Marshal(redactPayloadForAudit(v))
	if err != nil {
		fallback, fallbackErr := json.Marshal(map[string]string{"marshal_error": err.Error()})
		if fallbackErr != nil {
			return `{"marshal_error":"payload could not be marshaled"}`
		}

		return string(fallback)
	}

	return string(b)
}

// redactPayloadForAudit only handles JobCreatedPayload because it is the event
// payload that can carry inline auth material. Other event payloads are expected
// to be safe to persist verbatim in the job-event audit trail.
func redactPayloadForAudit(v any) any {
	switch payload := v.(type) {
	case *events.JobCreatedPayload:
		if payload == nil {
			return v
		}

		clone := *payload
		clone.Config.Auth = redactAuthForAudit(payload.Config.Auth)

		return &clone
	case events.JobCreatedPayload:
		clone := payload
		clone.Config.Auth = redactAuthForAudit(payload.Config.Auth)

		return clone
	default:
		return v
	}
}

func redactAuthForAudit(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return raw
	}

	var auth map[string]any
	if err := json.Unmarshal(raw, &auth); err != nil {
		return json.RawMessage(`{"redacted":true}`)
	}

	if _, ok := auth["content_b64"]; ok {
		delete(auth, "content_b64")
		auth["content_redacted"] = true
	}

	out, err := json.Marshal(auth)
	if err != nil {
		return json.RawMessage(`{"redacted":true}`)
	}

	return out
}

func (o *Orchestrator) recordJobEvent(ctx context.Context, e *db.JobEventInsert) {
	if o == nil || o.database == nil || e == nil {
		return
	}

	if err := o.database.InsertJobEvent(ctx, e); err != nil {
		slog.Warn("Failed to persist job event trace", "job_id", e.JobID, "event", e.Event, "error", err)
	}
}

func (o *Orchestrator) withInboundEvent(
	ctx context.Context,
	eventName, jobID string,
	payload any,
	fn func(context.Context) error,
) (err error) {
	start := time.Now()
	handlerStatus := "ok"

	defer func() {
		panicValue := recover()
		if panicValue != nil {
			err = fmt.Errorf("panic handling %s: %v", eventName, panicValue)
			handlerStatus = "panic"

			slog.Error(
				"Recovered panic in orchestrator event handler",
				"event",
				eventName,
				"job_id",
				jobID,
				"panic",
				panicValue,
				"stack",
				string(debug.Stack()),
			)
		}

		durationMs := time.Since(start).Milliseconds()

		insert := &db.JobEventInsert{
			JobID:         jobID,
			Event:         eventName,
			Timestamp:     time.Now().UTC(),
			Payload:       marshalPayload(payload),
			RequestID:     logging.RequestID(ctx),
			RunID:         logging.RunID(ctx),
			HandlerStatus: handlerStatus,
			DurationMs:    &durationMs,
		}

		if meta, ok := sharedmsg.ReceivedEventMetaFromContext(ctx); ok {
			applyReceivedEventMeta(insert, meta)
		}

		if err != nil {
			if handlerStatus == "ok" {
				insert.HandlerStatus = "error"
			}

			insert.HandlerError = err.Error()
		}

		o.metrics.ObserveEvent(insert.Event, insert.HandlerStatus, durationMs)
		o.recordJobEvent(ctx, insert)
	}()

	err = fn(ctx)

	return err
}

// applyReceivedEventMeta copies optional NATS delivery metadata onto a job
// event insert. Zero-valued sequence and timestamp fields are left nil so the
// audit row only records metadata the broker actually provided.
func applyReceivedEventMeta(insert *db.JobEventInsert, meta *sharedmsg.ReceivedEventMeta) {
	if meta == nil {
		return
	}

	if meta.Event != "" {
		insert.Event = meta.Event
	}

	insert.Producer = meta.Producer
	insert.NATSSubject = meta.Subject
	insert.NATSStream = meta.Stream
	insert.NATSConsumer = meta.Consumer

	if !meta.StoredAt.IsZero() {
		insert.NATSStoredAt = &meta.StoredAt
	}

	if meta.StreamSeq != 0 {
		insert.NATSStreamSeq = &meta.StreamSeq
	}

	if meta.ConsumerSeq != 0 {
		insert.NATSConsumerSeq = &meta.ConsumerSeq
	}

	if meta.Deliveries != 0 {
		insert.NATSDeliveries = &meta.Deliveries
	}
}

func (o *Orchestrator) recordInternalEvent(ctx context.Context, jobID, event string, payload any) {
	now := time.Now().UTC()
	durationMs := int64(0)

	insert := &db.JobEventInsert{
		JobID:         jobID,
		Event:         event,
		Timestamp:     now,
		Payload:       marshalPayload(payload),
		RequestID:     logging.RequestID(ctx),
		RunID:         logging.RunID(ctx),
		Producer:      "orchestrator",
		HandlerStatus: "ok",
		DurationMs:    &durationMs,
	}

	o.recordJobEvent(ctx, insert)
}
