package events

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewEnvelope_SetsExpectedFieldsAndTimestampUTC(t *testing.T) {
	t.Parallel()

	event := "test.event"
	jobID := "job-123"
	producer := "test-service"
	payload := map[string]string{"key": "value"}

	start := time.Now().UTC()
	env := NewEnvelope(event, jobID, producer, payload)
	end := time.Now().UTC()

	if env.Event != event {
		t.Fatalf("event mismatch: want=%q got=%q", event, env.Event)
	}

	if env.JobID != jobID {
		t.Fatalf("job_id mismatch: want=%q got=%q", jobID, env.JobID)
	}

	if env.Producer != producer {
		t.Fatalf("producer mismatch: want=%q got=%q", producer, env.Producer)
	}

	if env.Payload == nil {
		t.Fatalf("expected payload to be non-nil")
	}

	if env.Timestamp.IsZero() {
		t.Fatalf("expected timestamp to be set")
	}

	if env.Timestamp.Location() != time.UTC {
		t.Fatalf("expected timestamp to be UTC, got %v", env.Timestamp.Location())
	}

	// Allow some tolerance for clock adjustments, but the timestamp should be "around" the call.
	if env.Timestamp.Before(start.Add(-1*time.Second)) || env.Timestamp.After(end.Add(1*time.Second)) {
		t.Fatalf("timestamp out of expected range: start=%s ts=%s end=%s", start, env.Timestamp, end)
	}

	if err := env.Validate(); err != nil {
		t.Fatalf("expected Validate() to succeed: %v", err)
	}

	// omitempty behavior for RequestID/RunID
	b, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]any
	if unmarshalErr := json.Unmarshal(b, &m); unmarshalErr != nil {
		t.Fatalf("unmarshal: %v", unmarshalErr)
	}

	if _, ok := m["request_id"]; ok {
		t.Fatalf("expected request_id to be omitted when empty, JSON=%s", string(b))
	}

	if _, ok := m["run_id"]; ok {
		t.Fatalf("expected run_id to be omitted when empty, JSON=%s", string(b))
	}

	if _, ok := m["timestamp"]; !ok {
		t.Fatalf("expected timestamp to be present, JSON=%s", string(b))
	}
}

func TestNewEnvelopeAt_NormalizesTimestampToUTC(t *testing.T) {
	t.Parallel()

	// A non-UTC timestamp should be normalized to UTC.
	nonUTC := time.Date(2025, 12, 25, 10, 0, 0, 0, time.FixedZone("X", 3600))

	env := NewEnvelopeAt("test.event", "job-123", "svc", map[string]string{"a": "b"}, nonUTC)
	if env.Timestamp.Location() != time.UTC {
		t.Fatalf("expected UTC location, got %v", env.Timestamp.Location())
	}

	if !env.Timestamp.Equal(nonUTC.UTC()) {
		t.Fatalf("expected timestamp %s, got %s", nonUTC.UTC(), env.Timestamp)
	}
}

func TestEnvelope_Validate(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()

	cases := []struct {
		name    string
		env     *Envelope
		wantErr bool
	}{
		{"nil", nil, true},
		{"missing event", &Envelope{JobID: "job", Producer: "svc", Timestamp: now}, true},
		{"missing job", &Envelope{Event: "x", Producer: "svc", Timestamp: now}, true},
		{"missing producer", &Envelope{Event: "x", JobID: "job", Timestamp: now}, true},
		{"missing timestamp", &Envelope{Event: "x", JobID: "job", Producer: "svc"}, true},
		{"non-utc timestamp", &Envelope{Event: "x", JobID: "job", Producer: "svc", Timestamp: time.Now()}, true},
		{
			"valid",
			&Envelope{Event: "x", JobID: "job", Producer: "svc", Timestamp: now, Payload: map[string]any{}},
			false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.env.Validate()
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}

			if !tc.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}
