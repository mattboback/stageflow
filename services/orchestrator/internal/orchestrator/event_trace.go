package orchestrator

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"time"

	"github.com/mattboback/stageflow/libs/go/logging"
	sharedmsg "github.com/mattboback/stageflow/libs/go/messaging"
	db "github.com/mattboback/stageflow/platform/orchestrator/internal/adapters/repository"
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

	b, err := json.Marshal(v)
	if err != nil {
		return `{"marshal_error":` + strconvQuote(err.Error()) + `}`
	}

	return string(b)
}

func strconvQuote(s string) string {
	return strconv.Quote(s)
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
) error {
	start := time.Now()
	err := fn(ctx)
	durationMs := time.Since(start).Milliseconds()

	meta, _ := sharedmsg.ReceivedEventMetaFromContext(ctx)

	insert := &db.JobEventInsert{
		JobID:         jobID,
		Event:         eventName,
		Timestamp:     time.Now().UTC(),
		Payload:       marshalPayload(payload),
		RequestID:     logging.RequestID(ctx),
		RunID:         logging.RunID(ctx),
		HandlerStatus: "ok",
		DurationMs:    &durationMs,
	}

	//nolint:nestif // Meta validation requires multiple optional field checks
	if meta != nil {
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

	if err != nil {
		insert.HandlerStatus = "error"
		insert.HandlerError = err.Error()
	}

	o.recordJobEvent(ctx, insert)

	return err
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
