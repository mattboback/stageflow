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

func TestContractFixtures_ScanEventsDecodeStrict(t *testing.T) {
	t.Parallel()

	type envelope struct {
		Event     string          `json:"event"`
		JobID     string          `json:"job_id"`
		RequestID string          `json:"request_id,omitempty"`
		RunID     string          `json:"run_id,omitempty"`
		Timestamp time.Time       `json:"timestamp"`
		Producer  string          `json:"producer"`
		Payload   json.RawMessage `json:"payload"`
	}

	readFixture := func(t *testing.T, name string) []byte {
		t.Helper()
		path := filepath.Join("..", "..", "contracts", "events", "fixtures", name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read fixture %s: %v", name, err)
		}
		return data
	}

	unmarshalStrict := func(t *testing.T, data []byte, target any) {
		t.Helper()
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.DisallowUnknownFields()

		if err := dec.Decode(target); err != nil {
			t.Fatalf("strict decode: %v", err)
		}
		// Ensure there is exactly one JSON value (allow trailing whitespace only).
		if err := dec.Decode(&struct{}{}); err != io.EOF {
			if err == nil {
				t.Fatalf("strict decode: unexpected trailing JSON")
			}
			t.Fatalf("strict decode: trailing content: %v", err)
		}
	}

	t.Run("scan.page.completed", func(t *testing.T) {
		t.Parallel()

		raw := readFixture(t, "scan.page.completed.json")

		var env envelope
		unmarshalStrict(t, raw, &env)

		if env.Event != EventScanPageCompleted {
			t.Fatalf("unexpected event: want=%q got=%q", EventScanPageCompleted, env.Event)
		}
		if env.JobID == "" || env.Producer == "" || env.Timestamp.IsZero() {
			t.Fatalf("unexpected envelope: %+v", env)
		}

		var payload ScanPageCompletedPayload
		unmarshalStrict(t, env.Payload, &payload)

		if err := payload.Validate(); err != nil {
			t.Fatalf("payload Validate(): %v", err)
		}
	})

	t.Run("scan.completed", func(t *testing.T) {
		t.Parallel()

		raw := readFixture(t, "scan.completed.json")

		var env envelope
		unmarshalStrict(t, raw, &env)

		if env.Event != EventScanCompleted {
			t.Fatalf("unexpected event: want=%q got=%q", EventScanCompleted, env.Event)
		}
		if env.JobID == "" || env.Producer == "" || env.Timestamp.IsZero() {
			t.Fatalf("unexpected envelope: %+v", env)
		}

		var payload ScanCompletedPayload
		unmarshalStrict(t, env.Payload, &payload)

		if err := payload.Validate(); err != nil {
			t.Fatalf("payload Validate(): %v", err)
		}

		// Contract-specific expectations from fixtures:
		if payload.Summary.TotalViolations <= 0 {
			t.Fatalf("unexpected summary: %+v", payload.Summary)
		}
		if len(payload.Summary.BySeverity) == 0 {
			t.Fatalf("expected by_severity to be present: %+v", payload.Summary)
		}
		if payload.Timing == nil || payload.Timing.TotalMs <= 0 {
			t.Fatalf("expected timing to decode: %+v", payload.Timing)
		}
	})

	t.Run("scan.failed", func(t *testing.T) {
		t.Parallel()

		raw := readFixture(t, "scan.failed.json")

		var env envelope
		unmarshalStrict(t, raw, &env)

		if env.Event != EventScanFailed {
			t.Fatalf("unexpected event: want=%q got=%q", EventScanFailed, env.Event)
		}
		if env.JobID == "" || env.Producer == "" || env.Timestamp.IsZero() {
			t.Fatalf("unexpected envelope: %+v", env)
		}

		var payload ScanFailedPayload
		unmarshalStrict(t, env.Payload, &payload)

		if err := payload.Validate(); err != nil {
			t.Fatalf("payload Validate(): %v", err)
		}
	})
}
