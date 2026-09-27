package messaging

import (
	"context"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"

	"github.com/mattboback/stageflow/libs/go/events"
	sharedmsg "github.com/mattboback/stageflow/libs/go/messaging"
	"github.com/mattboback/stageflow/libs/go/models"
)

func TestRecoveredPanicIsRedelivered(t *testing.T) {
	client := newRecoveryTestClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := client.EnsureStreams(ctx); err != nil {
		t.Fatalf("ensure streams: %v", err)
	}

	deliveries := make(chan sharedmsg.ReceivedEventMeta, 2)

	handler := recoveringHandler(
		sharedmsg.SubjectJobCreated,
		func(ctx context.Context, _ *events.JobCreatedPayload) error {
			meta, ok := sharedmsg.ReceivedEventMetaFromContext(ctx)
			if !ok || meta == nil {
				return context.Canceled
			}

			deliveries <- *meta

			if meta.Deliveries == 1 {
				panic("panic-redelivery-secret-canary")
			}

			return nil
		},
	)
	if err := sharedmsg.SubscribeTyped(ctx, client, sharedmsg.Subscription[events.JobCreatedPayload]{
		Stream:  sharedmsg.StreamJobs,
		Subject: sharedmsg.SubjectJobCreated,
		Durable: "panic-redelivery-test",
		Handler: handler,
	}); err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	const jobID = "panic-redelivery-job"

	envelope := events.NewEnvelope(events.EventJobCreated, jobID, "messaging-test", &events.JobCreatedPayload{
		JobID:     jobID,
		InputType: events.InputTypeURLs,
		URLs:      []string{"https://example.com"},
		Config:    models.JobConfig{Modules: []string{"axe"}},
	})
	if err := client.PublishEvent(ctx, sharedmsg.SubjectJobCreated, envelope); err != nil {
		t.Fatalf("publish: %v", err)
	}

	var streamSequence uint64

	for attempt := uint64(1); attempt <= 2; attempt++ {
		select {
		case meta := <-deliveries:
			if meta.JobID != jobID || meta.Deliveries != attempt {
				t.Fatalf("delivery %d: unexpected metadata %+v", attempt, meta)
			}

			if attempt == 1 {
				streamSequence = meta.StreamSeq
			} else if meta.StreamSeq != streamSequence {
				t.Fatalf("redelivery sequence %d differs from original %d", meta.StreamSeq, streamSequence)
			}
		case <-ctx.Done():
			t.Fatalf("waiting for delivery %d: %v", attempt, ctx.Err())
		}
	}
}

func newRecoveryTestClient(t *testing.T) *sharedmsg.Client {
	t.Helper()

	server, err := natsserver.NewServer(&natsserver.Options{
		Host:      "127.0.0.1",
		Port:      -1,
		JetStream: true,
		StoreDir:  t.TempDir(),
		NoLog:     true,
		NoSigs:    true,
	})
	if err != nil {
		t.Fatalf("create embedded NATS: %v", err)
	}

	t.Cleanup(func() {
		server.Shutdown()
		server.WaitForShutdown()
	})

	go server.Start()

	if !server.ReadyForConnections(10 * time.Second) {
		t.Fatal("embedded NATS did not become ready")
	}

	client, err := sharedmsg.NewClient(&sharedmsg.Config{URL: server.ClientURL()})
	if err != nil {
		t.Fatalf("connect to embedded NATS: %v", err)
	}

	t.Cleanup(func() { _ = client.Close() })

	return client
}
