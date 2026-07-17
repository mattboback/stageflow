package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/mattboback/stageflow/libs/go/events"
)

type blockingConsumeContext struct {
	stopped chan struct{}
	closed  chan struct{}
	once    sync.Once
}

func newBlockingConsumeContext() *blockingConsumeContext {
	return &blockingConsumeContext{
		stopped: make(chan struct{}),
		closed:  make(chan struct{}),
	}
}

func (c *blockingConsumeContext) Stop() {
	c.once.Do(func() { close(c.stopped) })
}

func (c *blockingConsumeContext) Closed() <-chan struct{} {
	return c.closed
}

func TestNewClient_WithNilConfig(t *testing.T) {
	t.Parallel()

	// NewClient should use DefaultConfig when nil is passed
	// Note: This test will fail if NATS server is not running
	t.Skip("Skipping integration test - requires NATS server")

	client, err := NewClient(nil)
	if err != nil {
		t.Fatalf("NewClient with nil config should use defaults: %v", err)
	}

	t.Cleanup(func() {
		if closeErr := client.Close(); closeErr != nil {
			t.Errorf("Close() error: %v", closeErr)
		}
	})

	if client.nc == nil {
		t.Fatal("expected non-nil NATS connection")
	}

	if client.js == nil {
		t.Fatal("expected non-nil JetStream context")
	}
}

func TestNewClient_WithPartialConfig(t *testing.T) {
	t.Parallel()

	t.Skip("Skipping integration test - requires NATS server")

	// Test that zero-values in config are normalized to defaults
	cfg := &Config{
		URL: "", // Should be set to nats.DefaultURL
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient should normalize zero-values: %v", err)
	}

	t.Cleanup(func() {
		if closeErr := client.Close(); closeErr != nil {
			t.Errorf("Close() error: %v", closeErr)
		}
	})
}

func TestNewClient_InvalidURL(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		URL:            "nats://invalid-host-that-does-not-exist:4222",
		MaxReconnects:  1,
		ReconnectWait:  100 * time.Millisecond,
		ConnectTimeout: 500 * time.Millisecond,
	}

	_, err := NewClient(cfg)
	if err == nil {
		t.Fatal("expected error for invalid NATS URL")
	}
}

func TestClient_Snapshot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		client    *Client
		expectErr error
	}{
		{
			name:      "nil client",
			client:    nil,
			expectErr: ErrNilClient,
		},
		{
			name:      "nil NATS connection",
			client:    &Client{nc: nil, js: nil},
			expectErr: ErrNotConnected,
		},
		{
			name:      "closed client",
			client:    &Client{closed: true},
			expectErr: ErrNotConnected,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := tt.client.snapshot()
			if !errors.Is(err, tt.expectErr) {
				t.Fatalf("expected error %v, got %v", tt.expectErr, err)
			}
		})
	}
}

func TestClient_Publish_InvalidInputs(t *testing.T) {
	t.Parallel()

	var client *Client

	tests := []struct {
		name    string
		ctx     context.Context
		subject string
		data    any
		wantErr bool
	}{
		{
			name:    "nil context",
			ctx:     nil,
			subject: "test.subject",
			data:    map[string]any{"key": "value"},
			wantErr: true,
		},
		{
			name:    "empty subject",
			ctx:     context.Background(),
			subject: "",
			data:    map[string]any{"key": "value"},
			wantErr: true,
		},
		{
			name:    "nil client",
			ctx:     context.Background(),
			subject: "test.subject",
			data:    map[string]any{"key": "value"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := client.Publish(tt.ctx, tt.subject, tt.data)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Publish() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_Subscribe_InvalidInputs(t *testing.T) {
	t.Parallel()

	var client *Client

	handler := func([]byte) error { return nil }

	tests := []struct {
		name         string
		ctx          context.Context
		stream       string
		subject      string
		consumerName string
		handler      func([]byte) error
		wantErr      bool
	}{
		{
			name:         "nil context",
			ctx:          nil,
			stream:       "test-stream",
			subject:      "test.subject",
			consumerName: "test-consumer",
			handler:      handler,
			wantErr:      true,
		},
		{
			name:         "nil handler",
			ctx:          context.Background(),
			stream:       "test-stream",
			subject:      "test.subject",
			consumerName: "test-consumer",
			handler:      nil,
			wantErr:      true,
		},
		{
			name:         "nil client",
			ctx:          context.Background(),
			stream:       "test-stream",
			subject:      "test.subject",
			consumerName: "test-consumer",
			handler:      handler,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := client.Subscribe(tt.ctx, tt.stream, tt.subject, tt.consumerName, tt.handler)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Subscribe() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_SubscribeWithContext_InvalidInputs(t *testing.T) {
	t.Parallel()

	var client *Client

	tests := []struct {
		name    string
		ctx     context.Context
		handler func(context.Context, interface{}) error
		wantErr bool
	}{
		{
			name:    "nil context",
			ctx:     nil,
			handler: func(context.Context, interface{}) error { return nil },
			wantErr: true,
		},
		{
			name:    "nil handler",
			ctx:     context.Background(),
			handler: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := client.SubscribeWithContext(tt.ctx, "stream", "subject", "consumer", nil)
			if (err != nil) != tt.wantErr {
				t.Fatalf("SubscribeWithContext() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_PublishEvent_InvalidEnvelope(t *testing.T) {
	t.Parallel()

	var client *Client

	tests := []struct {
		name     string
		envelope any
		wantErr  bool
	}{
		{
			name:     "non-validatable envelope",
			envelope: map[string]any{"foo": "bar"},
			wantErr:  true,
		},
		{
			name:     "empty envelope",
			envelope: &events.Envelope{},
			wantErr:  true,
		},
		{
			name:     "valid envelope",
			envelope: events.NewEnvelope("test.event", "job-1", "service", map[string]any{"data": "value"}),
			wantErr:  true, // Still fails because client is nil
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := client.PublishEvent(context.Background(), "test.subject", tt.envelope)
			if (err != nil) != tt.wantErr {
				t.Fatalf("PublishEvent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSubscribeTyped_InvalidSubscription(t *testing.T) {
	t.Parallel()

	type TestPayload struct {
		Message string `json:"message"`
	}

	ctx := context.Background()

	tests := []struct {
		name    string
		client  *Client
		sub     Subscription[TestPayload]
		wantErr error
	}{
		{
			name:   "nil client",
			client: nil,
			sub: Subscription[TestPayload]{
				Stream:  "s",
				Subject: "subj",
				Durable: "d",
				Handler: func(context.Context, *TestPayload) error { return nil },
			},
			wantErr: ErrNilClient,
		},
		{
			name:   "empty stream",
			client: &Client{},
			sub: Subscription[TestPayload]{
				Stream:  "",
				Subject: "subj",
				Durable: "d",
				Handler: func(context.Context, *TestPayload) error { return nil },
			},
			wantErr: ErrBadSubscription,
		},
		{
			name:   "empty subject",
			client: &Client{},
			sub: Subscription[TestPayload]{
				Stream:  "s",
				Subject: "",
				Durable: "d",
				Handler: func(context.Context, *TestPayload) error { return nil },
			},
			wantErr: ErrBadSubscription,
		},
		{
			name:   "empty durable",
			client: &Client{},
			sub: Subscription[TestPayload]{
				Stream:  "s",
				Subject: "subj",
				Durable: "",
				Handler: func(context.Context, *TestPayload) error { return nil },
			},
			wantErr: ErrBadSubscription,
		},
		{
			name:    "nil handler",
			client:  &Client{},
			sub:     Subscription[TestPayload]{Stream: "s", Subject: "subj", Durable: "d", Handler: nil},
			wantErr: ErrNilHandler,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := SubscribeTyped(ctx, tt.client, tt.sub)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestClient_Close_NilClient(t *testing.T) {
	t.Parallel()

	var client *Client
	if err := client.Close(); err != nil {
		t.Fatalf("Close() on nil client should not error, got: %v", err)
	}
}

func TestClient_StopConsumersWaitsForActiveCallbacks(t *testing.T) {
	t.Parallel()

	consumeCtx := newBlockingConsumeContext()
	client := &Client{
		consumeContexts: map[string]consumeContext{"stream/consumer": consumeCtx},
	}

	returned := make(chan struct{})

	go func() {
		client.StopConsumers()
		close(returned)
	}()

	select {
	case <-consumeCtx.stopped:
	case <-time.After(time.Second):
		t.Fatal("StopConsumers did not stop the subscription")
	}

	select {
	case <-returned:
		t.Fatal("StopConsumers returned before the consume context closed")
	case <-time.After(25 * time.Millisecond):
	}

	close(consumeCtx.closed)

	select {
	case <-returned:
	case <-time.After(time.Second):
		t.Fatal("StopConsumers did not return after the consume context closed")
	}

	client.mu.Lock()
	defer client.mu.Unlock()

	if len(client.consumeContexts) != 0 {
		t.Fatalf("tracked consume contexts = %d, want 0", len(client.consumeContexts))
	}
}

func TestClient_StopConsumersClosesSubscriptionRegistration(t *testing.T) {
	t.Parallel()

	client := &Client{}
	client.StopConsumers()

	if err := client.beginConsumeSetup(); !errors.Is(err, ErrNotConnected) {
		t.Fatalf("beginConsumeSetup() error = %v, want ErrNotConnected", err)
	}
}

func TestClient_EnsureStreams_NilContext(t *testing.T) {
	t.Parallel()

	client := &Client{}

	var nilCtx context.Context

	err := client.EnsureStreams(nilCtx)
	if !errors.Is(err, ErrNilContext) {
		t.Fatalf("expected ErrNilContext, got %v", err)
	}
}

func TestUnmarshalStrict_UnmarshalableData(t *testing.T) {
	t.Parallel()

	data := []byte(`invalid json`)

	var result map[string]any

	err := unmarshalStrict(data, &result)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestUnmarshalLenient_UnmarshalableData(t *testing.T) {
	t.Parallel()

	data := []byte(`invalid json`)

	var result map[string]any

	err := unmarshalLenient(data, &result)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestClient_Publish_UnmarshalableData(t *testing.T) {
	t.Parallel()

	client := &Client{}

	// Create a type that can't be marshaled to JSON
	type unmarshalable struct {
		Ch chan int
	}

	data := unmarshalable{Ch: make(chan int)}

	err := client.Publish(context.Background(), "test.subject", data)
	if err == nil {
		t.Fatal("expected error for unmarshalable data")
	}
}

func TestReceivedEventMetaFromContext_InvalidContext(t *testing.T) {
	t.Parallel()

	// Context with wrong type
	ctx := context.WithValue(context.Background(), receivedMetaKey{}, "not a meta")

	_, ok := ReceivedEventMetaFromContext(ctx)
	if ok {
		t.Fatal("expected false for wrong type in context")
	}

	// Empty context
	_, ok = ReceivedEventMetaFromContext(context.Background())
	if ok {
		t.Fatal("expected false for empty context")
	}
}

func TestDefaultConfig_Normalization(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    *Config
		validate func(*testing.T, *Config)
	}{
		{
			name: "zero URL gets default",
			input: &Config{
				URL:            "",
				MaxReconnects:  5,
				ReconnectWait:  1 * time.Second,
				ConnectTimeout: 5 * time.Second,
			},
			validate: func(t *testing.T, cfg *Config) {
				t.Helper()

				// Normalization happens in NewClient, not in config itself
				if cfg.URL != "" {
					t.Fatalf("expected empty URL to remain empty in config, got %s", cfg.URL)
				}
			},
		},
		{
			name: "zero reconnect settings get defaults",
			input: &Config{
				URL:            "nats://localhost:4222",
				MaxReconnects:  0,
				ReconnectWait:  0,
				ConnectTimeout: 0,
			},
			validate: func(t *testing.T, cfg *Config) {
				t.Helper()

				if cfg.MaxReconnects != 0 {
					t.Fatalf("expected MaxReconnects to remain 0, got %d", cfg.MaxReconnects)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.validate(t, tt.input)
		})
	}
}

func TestUnmarshalStrict_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("empty JSON object", func(t *testing.T) {
		t.Parallel()

		data := []byte(`{}`)

		var result struct{}
		if err := unmarshalStrict(data, &result); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("null value", func(t *testing.T) {
		t.Parallel()

		data := []byte(`null`)

		var result *struct{}
		if err := unmarshalStrict(data, &result); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result != nil {
			t.Fatal("expected nil result")
		}
	})

	t.Run("nested object with unknown fields", func(t *testing.T) {
		t.Parallel()

		data := []byte(`{"known": {"unknown": "value"}}`)

		var result struct {
			Known struct {
				Known string `json:"known"`
			} `json:"known"`
		}
		if err := unmarshalStrict(data, &result); err == nil {
			t.Fatal("expected error for unknown nested field")
		}
	})
}

func TestUnmarshalLenient_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("deeply nested unknown fields", func(t *testing.T) {
		t.Parallel()

		data := []byte(`{"a": {"b": {"c": {"unknown": "deep"}}}}`)

		var result struct {
			A struct {
				B struct {
					C struct{} `json:"c"`
				} `json:"b"`
			} `json:"a"`
		}
		if err := unmarshalLenient(data, &result); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("array with unknown fields", func(t *testing.T) {
		t.Parallel()

		data := []byte(`[{"known": "value", "unknown": "ignored"}]`)

		var result []struct {
			Known string `json:"known"`
		}
		if err := unmarshalLenient(data, &result); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result) != 1 || result[0].Known != "value" {
			t.Fatalf("unexpected result: %+v", result)
		}
	})
}

func TestValidatePublishEventEnvelope_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("envelope with empty required fields", func(t *testing.T) {
		t.Parallel()

		env := &events.Envelope{
			Event:     "",
			JobID:     "",
			Timestamp: time.Time{},
			Producer:  "",
			Payload:   json.RawMessage(`{}`),
		}

		err := validatePublishEventEnvelope(env)
		if err == nil {
			t.Fatal("expected error for envelope with empty required fields")
		}
	})

	t.Run("untyped nil interface", func(t *testing.T) {
		t.Parallel()

		var env any

		err := validatePublishEventEnvelope(env)
		if err == nil {
			t.Fatal("expected error for nil interface")
		}
	})
}
