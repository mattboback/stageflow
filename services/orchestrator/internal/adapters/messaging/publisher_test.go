package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/mattboback/stageflow/libs/go/events"
	sharedmsg "github.com/mattboback/stageflow/libs/go/messaging"
)

func TestNewPublisher(t *testing.T) {
	t.Run("creates publisher with nil client", func(t *testing.T) {
		pub := NewPublisher(nil)

		if pub == nil {
			t.Fatal("NewPublisher returned nil")
		}

		if pub.natsClient != nil {
			t.Error("expected natsClient to be nil")
		}
	})
}

// TestEnvelopeConstruction tests that our Publisher correctly constructs envelopes
// by verifying the events package integration.
//
//nolint:gocognit // Test function covers multiple envelope construction scenarios
func TestEnvelopeConstruction(t *testing.T) {
	t.Run("JobCompletedPayload creates valid envelope", func(t *testing.T) {
		payload := &events.JobCompletedPayload{
			JobID:  "job-123",
			Status: "success",
			Artifacts: events.ArtifactLocations{
				ReportJSON: "job-123/report.json",
				ReportHTML: "job-123/report.html",
			},
		}

		envelope := events.NewEnvelope(events.EventJobCompleted, payload.JobID, "orchestrator", payload)

		if envelope.Event != events.EventJobCompleted {
			t.Errorf("Event = %q, want %q", envelope.Event, events.EventJobCompleted)
		}

		if envelope.JobID != "job-123" {
			t.Errorf("JobID = %q, want %q", envelope.JobID, "job-123")
		}

		if envelope.Producer != "orchestrator" {
			t.Errorf("Producer = %q, want %q", envelope.Producer, "orchestrator")
		}

		// Verify payload is correctly attached
		if envelope.Payload == nil {
			t.Fatal("Payload is nil")
		}

		// Verify it can be marshaled
		data, err := json.Marshal(envelope)
		if err != nil {
			t.Fatalf("failed to marshal envelope: %v", err)
		}

		if len(data) == 0 {
			t.Error("marshaled envelope is empty")
		}
	})

	t.Run("JobFailedPayload creates valid envelope", func(t *testing.T) {
		payload := &events.JobFailedPayload{
			JobID:        "job-456",
			Status:       "failed",
			Stage:        "scanning",
			Error:        "scanner timeout",
			ErrorDetails: "Connection refused after 30s",
		}

		envelope := events.NewEnvelope(events.EventJobFailed, payload.JobID, "orchestrator", payload)

		if envelope.Event != events.EventJobFailed {
			t.Errorf("Event = %q, want %q", envelope.Event, events.EventJobFailed)
		}

		if envelope.JobID != "job-456" {
			t.Errorf("JobID = %q, want %q", envelope.JobID, "job-456")
		}

		// Verify payload roundtrip
		data, err := json.Marshal(envelope)
		if err != nil {
			t.Fatalf("failed to marshal envelope: %v", err)
		}

		var decoded struct {
			Event   string `json:"event"`
			JobID   string `json:"job_id"`
			Payload struct {
				JobID   string `json:"job_id"`
				Status  string `json:"status"`
				Error   string `json:"error"`
				Details string `json:"details"`
			} `json:"payload"`
		}

		if unmarshalErr := json.Unmarshal(data, &decoded); unmarshalErr != nil {
			t.Fatalf("failed to unmarshal envelope: %v", unmarshalErr)
		}

		if decoded.Payload.Error != "scanner timeout" {
			t.Errorf("Payload.Error = %q, want %q", decoded.Payload.Error, "scanner timeout")
		}
	})
}

// TestPublisherMethodSignatures verifies the Publisher methods exist with expected signatures.
// Actual publish behavior requires integration tests with NATS.
func TestPublisherMethodSignatures(t *testing.T) {
	pub := NewPublisher(nil)
	ctx := context.Background()

	t.Run("PublishJobCompleted returns error with nil client", func(t *testing.T) {
		payload := &events.JobCompletedPayload{
			JobID:  "test",
			Status: "success",
		}

		err := pub.PublishJobCompleted(ctx, payload)
		if err == nil {
			t.Fatal("expected error with nil client, got nil")
		}

		if !errors.Is(err, sharedmsg.ErrNilClient) {
			t.Fatalf("expected ErrNilClient, got %v", err)
		}
	})

	t.Run("PublishJobFailed returns error with nil client", func(t *testing.T) {
		payload := &events.JobFailedPayload{
			JobID:  "test",
			Status: "failed",
			Stage:  "testing",
			Error:  "test error",
		}

		err := pub.PublishJobFailed(ctx, payload)
		if err == nil {
			t.Fatal("expected error with nil client, got nil")
		}

		if !errors.Is(err, sharedmsg.ErrNilClient) {
			t.Fatalf("expected ErrNilClient, got %v", err)
		}
	})
}
