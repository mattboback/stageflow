package events

import (
	"errors"
	"fmt"
	"time"
)

// Envelope is the canonical wrapper for all Stageflow events published over the bus.
//
// Timestamp is always stored in UTC.
type Envelope struct {
	Event     string    `json:"event"`
	JobID     string    `json:"job_id"`
	RequestID string    `json:"request_id,omitempty"`
	RunID     string    `json:"run_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Producer  string    `json:"producer"`
	Payload   any       `json:"payload"`
}

// NewEnvelope creates a new envelope with Timestamp set to time.Now().UTC().
//
// Use NewEnvelopeAt in tests (or when you need deterministic timestamps).
func NewEnvelope(event, jobID, producer string, payload any) *Envelope {
	return NewEnvelopeAt(event, jobID, producer, payload, time.Now().UTC())
}

// NewEnvelopeAt creates a new envelope with a caller-provided timestamp.
// The timestamp is normalized to UTC.
func NewEnvelopeAt(event, jobID, producer string, payload any, ts time.Time) *Envelope {
	return &Envelope{
		Event:     event,
		JobID:     jobID,
		Timestamp: ts.UTC(),
		Producer:  producer,
		Payload:   payload,
	}
}

// Validate performs basic sanity checks. It does not validate the payload shape.
func (e *Envelope) Validate() error {
	if e == nil {
		return errors.New("events: envelope is nil")
	}

	if e.Event == "" {
		return errors.New("events: envelope.event is required")
	}

	if e.JobID == "" {
		return errors.New("events: envelope.job_id is required")
	}

	if e.Producer == "" {
		return errors.New("events: envelope.producer is required")
	}

	if e.Timestamp.IsZero() {
		return errors.New("events: envelope.timestamp is required")
	}

	// Ensure we consistently store timestamps as UTC.
	if e.Timestamp.Location() != time.UTC {
		return fmt.Errorf("events: envelope.timestamp must be UTC (got %v)", e.Timestamp.Location())
	}

	return nil
}
