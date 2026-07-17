package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"github.com/mattboback/stageflow/libs/go/logging"
	sharedmsg "github.com/mattboback/stageflow/libs/go/messaging"
	db "github.com/mattboback/stageflow/services/orchestrator/internal/adapters/repository"
)

func (o *Orchestrator) backgroundWithCorrelation(ctx context.Context) context.Context {
	bg := context.Background()
	if o != nil && o.lifecycleCtx != nil {
		bg = o.lifecycleCtx
	}

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
		fallback, fallbackErr := json.Marshal(map[string]string{"marshal_error": "payload could not be marshaled"})
		if fallbackErr != nil {
			return `{"marshal_error":"payload could not be marshaled"}`
		}

		return string(fallback)
	}

	return string(b)
}

// redactPayloadForAudit converts payloads to their JSON representation and
// removes execution-only values before the audit row is persisted. Doing this
// generically protects new producers and payload shapes without requiring every
// caller to remember a concrete event type.
func redactPayloadForAudit(v any) any {
	raw, err := json.Marshal(v)
	if err != nil {
		return v
	}

	var decoded any
	if unmarshalErr := json.Unmarshal(raw, &decoded); unmarshalErr != nil {
		return map[string]any{"redacted": true}
	}

	secrets := make([]string, 0)
	collectAuditSecrets(decoded, &secrets)

	return redactAuditSecretDuplicates(redactAuditValue(decoded), uniqueAuditSecrets(secrets))
}

func collectAuditSecrets(value any, secrets *[]string) {
	switch typed := value.(type) {
	case []any:
		for _, entry := range typed {
			collectAuditSecrets(entry, secrets)
		}
	case map[string]any:
		for key, entry := range typed {
			switch normalizeAuditKey(key) {
			case "auth", "prescanactions":
				collectAuditActionValues(entry, secrets)
			case "inputvalues", "contentb64", "password", "passwd", "username":
				collectAuditStringValues(entry, secrets)
			default:
				collectAuditSecrets(entry, secrets)
			}
		}
	}
}

func collectAuditActionValues(value any, secrets *[]string) {
	switch typed := value.(type) {
	case []any:
		for _, entry := range typed {
			collectAuditActionValues(entry, secrets)
		}
	case map[string]any:
		for key, entry := range typed {
			switch normalizeAuditKey(key) {
			case "value", "contentb64", "password", "passwd", "username":
				collectAuditStringValues(entry, secrets)
			default:
				collectAuditActionValues(entry, secrets)
			}
		}
	}
}

func collectAuditStringValues(value any, secrets *[]string) {
	switch typed := value.(type) {
	case string:
		if typed != "" && typed != "[REDACTED]" {
			*secrets = append(*secrets, typed)
		}
	case []any:
		for _, entry := range typed {
			collectAuditStringValues(entry, secrets)
		}
	case map[string]any:
		for _, entry := range typed {
			collectAuditStringValues(entry, secrets)
		}
	}
}

func uniqueAuditSecrets(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))

	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}

		seen[value] = struct{}{}
		out = append(out, value)
	}

	sort.Slice(out, func(i, j int) bool { return len(out[i]) > len(out[j]) })

	return out
}

func redactAuditSecretDuplicates(value any, secrets []string) any {
	switch typed := value.(type) {
	case string:
		for _, secret := range secrets {
			typed = strings.ReplaceAll(typed, secret, "[REDACTED]")
		}

		return typed
	case []any:
		for index, entry := range typed {
			typed[index] = redactAuditSecretDuplicates(entry, secrets)
		}

		return typed
	case map[string]any:
		for key, entry := range typed {
			typed[key] = redactAuditSecretDuplicates(entry, secrets)
		}

		return typed
	default:
		return value
	}
}

func redactAuditValue(value any) any {
	switch typed := value.(type) {
	case []any:
		out := make([]any, len(typed))
		for index, entry := range typed {
			out[index] = redactAuditValue(entry)
		}

		return out
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, entry := range typed {
			switch normalizeAuditKey(key) {
			case "auth":
				out[key] = summarizeAuthForAudit(entry)
			case "error", "errordetails":
				// Producer failures may echo browser URLs or submitted values.
				// The event type and handler outcome retain enough diagnostic
				// signal without persisting arbitrary execution text for 30 days.
				out[key] = "[REDACTED]"
			case "inputvalues":
				out[key] = redactInputValuesForAudit(entry)
			case "prescanactions":
				out[key] = map[string]any{"configured": true, "redacted": true}
			case "contentb64", "password", "passwd", "username":
				out[key] = "[REDACTED]"
			default:
				out[key] = redactAuditValue(entry)
			}
		}

		return out
	default:
		return value
	}
}

func summarizeAuthForAudit(value any) map[string]any {
	summary := map[string]any{"configured": true}

	if auth, isAuth := value.(map[string]any); isAuth {
		if mode, hasMode := auth["mode"].(string); hasMode && mode != "" {
			summary["mode"] = mode
		}
	}

	return summary
}

func redactInputValuesForAudit(value any) any {
	values, ok := value.(map[string]any)
	if !ok {
		return map[string]any{"redacted": true}
	}

	redacted := make(map[string]any, len(values))
	for key := range values {
		redacted[key] = "[REDACTED]"
	}

	return redacted
}

func normalizeAuditKey(key string) string {
	replacer := strings.NewReplacer("_", "", "-", "", " ", "")

	return replacer.Replace(strings.ToLower(key))
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
			err = fmt.Errorf("panic handling %s", eventName)
			handlerStatus = "panic"

			slog.Error(
				"Recovered panic in orchestrator event handler",
				"event",
				eventName,
				"job_id",
				jobID,
				"panic_type",
				fmt.Sprintf("%T", panicValue),
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

			// Handler errors can echo browser input values or URLs containing
			// credentials. Keep the status useful without persisting arbitrary
			// execution text into the 30-day audit table.
			insert.HandlerError = "handler returned an error"
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
