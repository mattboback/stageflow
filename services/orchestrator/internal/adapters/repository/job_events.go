package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

// JobEvent represents a job event.
type JobEvent struct {
	ID           int64
	JobID        string
	PayloadJobID string
	Event        string
	Timestamp    time.Time
	Payload      string

	RequestID string
	RunID     string
	Producer  string

	NATSSubject     string
	NATSStream      string
	NATSConsumer    string
	NATSStreamSeq   int64
	NATSConsumerSeq int64
	NATSDeliveries  int64
	NATSStoredAt    *time.Time

	HandlerStatus string
	HandlerError  string
	DurationMs    int64
}

// JobEventInsert captures the insertable fields for a job event record.
type JobEventInsert struct {
	JobID     string
	Event     string
	Timestamp time.Time
	Payload   string

	RequestID string
	RunID     string
	Producer  string

	NATSSubject     string
	NATSStream      string
	NATSConsumer    string
	NATSStreamSeq   *uint64
	NATSConsumerSeq *uint64
	NATSDeliveries  *uint64
	NATSStoredAt    *time.Time

	HandlerStatus string
	HandlerError  string
	DurationMs    *int64
}

// InsertJobEvent inserts a job event record.
func (d *Database) InsertJobEvent(ctx context.Context, e *JobEventInsert) error {
	if e == nil {
		return errors.New("job event is nil")
	}

	if e.Event == "" {
		return errors.New("event is required")
	}

	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now().UTC()
	}

	query := `
		INSERT INTO job_events (
			job_id, payload_job_id, event, timestamp, payload_json,
			request_id, run_id, producer,
			nats_subject, nats_stream, nats_consumer, nats_stream_seq, nats_consumer_seq, nats_deliveries, nats_stored_at,
			handler_status, handler_error, duration_ms
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var streamSeq any

	if e.NATSStreamSeq != nil {
		if *e.NATSStreamSeq > uint64(math.MaxInt64) {
			return fmt.Errorf("nats_stream_seq overflows int64: %d", *e.NATSStreamSeq)
		}

		streamSeq = int64(*e.NATSStreamSeq)
	}

	var consumerSeq any

	if e.NATSConsumerSeq != nil {
		if *e.NATSConsumerSeq > uint64(math.MaxInt64) {
			return fmt.Errorf("nats_consumer_seq overflows int64: %d", *e.NATSConsumerSeq)
		}

		consumerSeq = int64(*e.NATSConsumerSeq)
	}

	var deliveries any

	if e.NATSDeliveries != nil {
		if *e.NATSDeliveries > uint64(math.MaxInt64) {
			return fmt.Errorf("nats_deliveries overflows int64: %d", *e.NATSDeliveries)
		}

		deliveries = int64(*e.NATSDeliveries)
	}

	var storedAt any
	if e.NATSStoredAt != nil {
		storedAt = *e.NATSStoredAt
	}

	var duration any
	if e.DurationMs != nil {
		duration = *e.DurationMs
	}

	insert := func(jobID any) error {
		_, execErr := d.execContext(
			ctx,
			query,
			jobID,
			nullIfEmpty(e.JobID),
			e.Event,
			e.Timestamp,
			nullIfEmpty(e.Payload),
			nullIfEmpty(e.RequestID),
			nullIfEmpty(e.RunID),
			nullIfEmpty(e.Producer),
			nullIfEmpty(e.NATSSubject),
			nullIfEmpty(e.NATSStream),
			nullIfEmpty(e.NATSConsumer),
			streamSeq,
			consumerSeq,
			deliveries,
			storedAt,
			nullIfEmpty(e.HandlerStatus),
			nullIfEmpty(e.HandlerError),
			duration,
		)

		return execErr
	}

	err := insert(nullIfEmpty(e.JobID))
	if err != nil && isForeignKeyFailure(err) {
		err = insert(nil)
	}

	if err != nil {
		return fmt.Errorf("failed to insert job event: %w", err)
	}

	return nil
}

func isForeignKeyFailure(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())

	return strings.Contains(msg, "foreign key")
}

// ListJobEventsOptions controls paging for ListJobEvents.
type ListJobEventsOptions struct {
	Limit  int
	Offset int
}

// ListJobEvents retrieves events for a job ordered by timestamp ASC.
func (d *Database) ListJobEvents(ctx context.Context, jobID string, opts ListJobEventsOptions) ([]JobEvent, error) {
	if opts.Limit <= 0 {
		opts.Limit = 500
	}

	if opts.Limit > 5000 {
		opts.Limit = 5000
	}

	if opts.Offset < 0 {
		opts.Offset = 0
	}

	query := `
		SELECT
			id, COALESCE(job_id, ''), COALESCE(payload_job_id, ''), event, timestamp, payload_json,
			COALESCE(request_id, ''), COALESCE(run_id, ''), COALESCE(producer, ''),
			COALESCE(nats_subject, ''), COALESCE(nats_stream, ''), COALESCE(nats_consumer, ''),
			nats_stream_seq, nats_consumer_seq, nats_deliveries, nats_stored_at,
			COALESCE(handler_status, ''), COALESCE(handler_error, ''), duration_ms
		FROM job_events
		WHERE job_id = ?
		ORDER BY timestamp ASC
		LIMIT ? OFFSET ?
	`

	rows, err := d.queryContext(ctx, query, jobID, opts.Limit, opts.Offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get job events: %w", err)
	}

	defer func() {
		_ = rows.Close()
	}()

	var events []JobEvent

	for rows.Next() {
		var (
			event       JobEvent
			payloadJSON sql.NullString
			streamSeq   sql.NullInt64
			consumerSeq sql.NullInt64
			deliveries  sql.NullInt64
			storedAt    sql.NullTime
			duration    sql.NullInt64
		)

		if scanErr := rows.Scan(
			&event.ID,
			&event.JobID,
			&event.PayloadJobID,
			&event.Event,
			&event.Timestamp,
			&payloadJSON,
			&event.RequestID,
			&event.RunID,
			&event.Producer,
			&event.NATSSubject,
			&event.NATSStream,
			&event.NATSConsumer,
			&streamSeq,
			&consumerSeq,
			&deliveries,
			&storedAt,
			&event.HandlerStatus,
			&event.HandlerError,
			&duration,
		); scanErr != nil {
			return nil, fmt.Errorf("failed to scan event: %w", scanErr)
		}

		if payloadJSON.Valid {
			event.Payload = payloadJSON.String
		}

		if streamSeq.Valid {
			event.NATSStreamSeq = streamSeq.Int64
		}

		if consumerSeq.Valid {
			event.NATSConsumerSeq = consumerSeq.Int64
		}

		if deliveries.Valid {
			event.NATSDeliveries = deliveries.Int64
		}

		if storedAt.Valid {
			t := storedAt.Time
			event.NATSStoredAt = &t
		}

		if duration.Valid {
			event.DurationMs = duration.Int64
		}

		events = append(events, event)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("failed iterating job events: %w", rowsErr)
	}

	return events, nil
}

func nullIfEmpty(v string) any {
	if v == "" {
		return nil
	}

	return v
}
