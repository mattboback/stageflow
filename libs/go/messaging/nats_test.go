package messaging

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/mattboback/stageflow/libs/go/events"
)

func TestStreamNames_NonEmpty(t *testing.T) {
	t.Parallel()

	streams := []string{StreamJobs, StreamExtraction, StreamScan}
	for _, s := range streams {
		if s == "" {
			t.Fatal("stream name should not be empty")
		}
	}
}

func TestSubjectNames_NonEmpty(t *testing.T) {
	t.Parallel()

	subjects := []string{
		SubjectJobCreated,
		SubjectJobCompleted,
		SubjectJobFailed,
		SubjectExtractionReady,
		SubjectExtractionFailed,
		SubjectScanPageCompleted,
		SubjectScanCompleted,
		SubjectScanFailed,
	}

	for _, s := range subjects {
		if s == "" {
			t.Fatal("subject name should not be empty")
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()
	if cfg == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	if cfg.URL != nats.DefaultURL {
		t.Fatalf("expected URL %s, got %s", nats.DefaultURL, cfg.URL)
	}

	if cfg.MaxReconnects != 10 {
		t.Fatalf("expected MaxReconnects 10, got %d", cfg.MaxReconnects)
	}

	if cfg.ReconnectWait != 2*time.Second {
		t.Fatalf("expected ReconnectWait 2s, got %v", cfg.ReconnectWait)
	}

	if cfg.ConnectTimeout != 10*time.Second {
		t.Fatalf("expected ConnectTimeout 10s, got %v", cfg.ConnectTimeout)
	}
}

func TestUnmarshalStrict(t *testing.T) {
	t.Parallel()

	t.Run("valid JSON", func(t *testing.T) {
		t.Parallel()

		data := []byte(`{"name":"test","value":42}`)

		var result struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}

		if err := unmarshalStrict(data, &result); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Name != "test" || result.Value != 42 {
			t.Fatalf("unexpected result: %+v", result)
		}
	})

	t.Run("accepts trailing whitespace", func(t *testing.T) {
		t.Parallel()

		data := []byte("{\"name\":\"test\"} \n\t ")

		var result struct {
			Name string `json:"name"`
		}
		if err := unmarshalStrict(data, &result); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("rejects unknown fields", func(t *testing.T) {
		t.Parallel()

		data := []byte(`{"name":"test","unknown_field":"value"}`)

		var result struct {
			Name string `json:"name"`
		}
		if err := unmarshalStrict(data, &result); err == nil {
			t.Fatal("expected error for unknown field")
		}
	})

	t.Run("rejects trailing JSON value", func(t *testing.T) {
		t.Parallel()

		data := []byte(`{"name":"test"}{"extra":"data"}`)

		var result struct {
			Name string `json:"name"`
		}
		if err := unmarshalStrict(data, &result); err == nil {
			t.Fatal("expected error for trailing JSON")
		}
	})

	t.Run("rejects trailing garbage", func(t *testing.T) {
		t.Parallel()

		data := []byte(`{"name":"test"}!!!`)

		var result struct {
			Name string `json:"name"`
		}
		if err := unmarshalStrict(data, &result); err == nil {
			t.Fatal("expected error for trailing garbage")
		}
	})
}

func TestUnmarshalLenient(t *testing.T) {
	t.Parallel()

	t.Run("accepts unknown fields", func(t *testing.T) {
		t.Parallel()

		data := []byte(`{"name":"test","unknown_field":"value"}`)

		var result struct {
			Name string `json:"name"`
		}
		if err := unmarshalLenient(data, &result); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Name != "test" {
			t.Fatalf("unexpected result: %+v", result)
		}
	})

	t.Run("rejects trailing JSON value", func(t *testing.T) {
		t.Parallel()

		data := []byte(`{"name":"test"}{"extra":"data"}`)

		var result struct {
			Name string `json:"name"`
		}
		if err := unmarshalLenient(data, &result); err == nil {
			t.Fatal("expected error for trailing JSON")
		}
	})

	t.Run("rejects trailing garbage", func(t *testing.T) {
		t.Parallel()

		data := []byte(`{"name":"test"}!!!`)

		var result struct {
			Name string `json:"name"`
		}
		if err := unmarshalLenient(data, &result); err == nil {
			t.Fatal("expected error for trailing garbage")
		}
	})
}

func TestReceivedEventMeta_ContextHelpers(t *testing.T) {
	t.Parallel()

	base := context.Background()

	// nil meta should be a no-op
	ctx := WithReceivedEventMeta(base, nil)
	if _, ok := ReceivedEventMetaFromContext(ctx); ok {
		t.Fatal("expected no meta in context")
	}

	meta := &ReceivedEventMeta{Event: "x", JobID: "job-1"}
	ctx2 := WithReceivedEventMeta(base, meta)

	got, ok := ReceivedEventMetaFromContext(ctx2)
	if !ok {
		t.Fatal("expected meta to be present")
	}

	if got.Event != "x" || got.JobID != "job-1" {
		t.Fatalf("unexpected meta: %+v", got)
	}
}

func TestClient_MethodGuards(t *testing.T) {
	t.Parallel()

	var c *Client
	if err := c.Publish(context.Background(), "subj", map[string]any{}); err == nil {
		t.Fatal("expected error for nil client")
	}

	// A client with no connection / js should not panic.
	c2 := &Client{}
	if err := c2.Publish(context.Background(), "subj", map[string]any{}); err == nil {
		t.Fatal("expected error for not-connected client")
	}

	if err := c2.EnsureStreams(context.Background()); err == nil {
		t.Fatal("expected error for not-connected client")
	}

	if err := c2.Subscribe(context.Background(), "s", "subj", "dur", func([]byte) error { return nil }); err == nil {
		t.Fatal("expected error for not-connected client")
	}
}

func TestClient_Close(t *testing.T) {
	t.Parallel()

	client := &Client{nc: nil}
	if err := client.Close(); err != nil {
		t.Fatalf("Close() on nil connection should not error, got: %v", err)
	}
}

func TestValidatePublishEventEnvelope(t *testing.T) {
	t.Parallel()

	t.Run("rejects non-validatable envelope", func(t *testing.T) {
		t.Parallel()

		err := validatePublishEventEnvelope(map[string]any{"x": 1})
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if !errors.Is(err, ErrBadEnvelope) {
			t.Fatalf("expected ErrBadEnvelope, got %v", err)
		}
	})

	t.Run("rejects invalid envelope", func(t *testing.T) {
		t.Parallel()

		err := validatePublishEventEnvelope(&events.Envelope{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if !errors.Is(err, ErrBadEnvelope) {
			t.Fatalf("expected ErrBadEnvelope, got %v", err)
		}
	})

	t.Run("accepts valid envelope", func(t *testing.T) {
		t.Parallel()

		env := events.NewEnvelope("test.event", "job-1", "svc", map[string]any{"a": "b"})
		if err := validatePublishEventEnvelope(env); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("handles typed nil envelope", func(t *testing.T) {
		t.Parallel()

		var env *events.Envelope

		err := validatePublishEventEnvelope(env)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if !errors.Is(err, ErrBadEnvelope) {
			t.Fatalf("expected ErrBadEnvelope, got %v", err)
		}
	})
}
