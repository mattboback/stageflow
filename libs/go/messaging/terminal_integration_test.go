package messaging

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/mattboback/stageflow/libs/go/events"
)

// A terminal error must stop redelivery immediately rather than spending the
// whole budget. Run against the embedded JetStream server so the assertion is
// about real consumer behavior, not a stub.
func TestTerminalErrorIsNotRedelivered(t *testing.T) {
	requireDefaultURLNATS(t)

	client, err := NewClient(&Config{URL: nats.DefaultURL})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	t.Cleanup(func() { _ = client.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if ensureErr := client.EnsureStreams(ctx); ensureErr != nil {
		t.Skipf("JetStream unavailable: %v", ensureErr)
	}

	var calls atomic.Int64

	subErr := SubscribeTyped(ctx, client, Subscription[map[string]any]{
		Stream:  StreamJobs,
		Subject: SubjectJobCreated,
		Durable: "terminal-error-test",
		Handler: func(_ context.Context, _ *map[string]any) error {
			calls.Add(1)

			return Terminal(errors.New("permanently undeliverable"))
		},
	})
	if subErr != nil {
		t.Fatalf("SubscribeTyped: %v", subErr)
	}

	envelope := events.NewEnvelope("job.created", "t-1", "messaging-test", map[string]any{"job_id": "t-1"})
	if pubErr := client.PublishEvent(ctx, SubjectJobCreated, envelope); pubErr != nil {
		t.Fatalf("PublishEvent: %v", pubErr)
	}

	deadline := time.Now().Add(5 * time.Second)
	for calls.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}

	if calls.Load() == 0 {
		t.Fatal("handler was never invoked")
	}

	// A NAK would redeliver after the escalating delay; a Term must not. Wait past
	// the first NAK interval so a regression here is caught rather than raced.
	time.Sleep(defaultConsumerNAKDelay + 2*time.Second)

	if got := calls.Load(); got != 1 {
		t.Fatalf("handler invocations = %d, want exactly 1: a terminal error must not be redelivered", got)
	}
}
