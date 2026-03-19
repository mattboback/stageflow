package events

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type contractEnvelope struct {
	Event     string          `json:"event"`
	JobID     string          `json:"job_id"`
	RequestID string          `json:"request_id,omitempty"`
	RunID     string          `json:"run_id,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
	Producer  string          `json:"producer"`
	Payload   json.RawMessage `json:"payload"`
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()

	path := filepath.Join("..", "..", "contracts", "events", "fixtures", name)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}

	return data
}

func unmarshalStrict(t *testing.T, data []byte, target any) {
	t.Helper()

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()

	if err := dec.Decode(target); err != nil {
		t.Fatalf("strict decode: %v", err)
	}

	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			t.Fatalf("strict decode: unexpected trailing JSON")
		}

		t.Fatalf("strict decode: trailing content: %v", err)
	}
}

func readEnvelopeFixture(t *testing.T, name string) contractEnvelope {
	t.Helper()

	raw := readFixture(t, name)

	var env contractEnvelope
	unmarshalStrict(t, raw, &env)

	return env
}

func assertBaseEnvelopeFields(t *testing.T, env contractEnvelope, expectedEvent string) {
	t.Helper()

	if env.Event != expectedEvent {
		t.Fatalf("unexpected event: want=%q got=%q", expectedEvent, env.Event)
	}

	if env.JobID == "" || env.Producer == "" || env.Timestamp.IsZero() {
		t.Fatalf("unexpected envelope: %+v", env)
	}
}

func TestContractFixtures_ScanPageCompleted_DecodeStrict(t *testing.T) {
	t.Parallel()

	env := readEnvelopeFixture(t, "scan.page.completed.json")
	assertBaseEnvelopeFields(t, env, EventScanPageCompleted)

	var payload ScanPageCompletedPayload
	unmarshalStrict(t, env.Payload, &payload)

	if err := payload.Validate(); err != nil {
		t.Fatalf("payload Validate(): %v", err)
	}
}

func TestContractFixtures_ScanCompleted_DecodeStrict(t *testing.T) {
	t.Parallel()

	env := readEnvelopeFixture(t, "scan.completed.json")
	assertBaseEnvelopeFields(t, env, EventScanCompleted)

	var payload ScanCompletedPayload
	unmarshalStrict(t, env.Payload, &payload)

	if err := payload.Validate(); err != nil {
		t.Fatalf("payload Validate(): %v", err)
	}

	if payload.Summary.TotalViolations <= 0 {
		t.Fatalf("unexpected summary: %+v", payload.Summary)
	}

	if len(payload.Summary.BySeverity) == 0 {
		t.Fatalf("expected by_severity to be present: %+v", payload.Summary)
	}

	if payload.Timing == nil || payload.Timing.TotalMs <= 0 {
		t.Fatalf("expected timing to decode: %+v", payload.Timing)
	}
}

func TestContractFixtures_ScanFailed_DecodeStrict(t *testing.T) {
	t.Parallel()

	env := readEnvelopeFixture(t, "scan.failed.json")
	assertBaseEnvelopeFields(t, env, EventScanFailed)

	var payload ScanFailedPayload
	unmarshalStrict(t, env.Payload, &payload)

	if err := payload.Validate(); err != nil {
		t.Fatalf("payload Validate(): %v", err)
	}
}
