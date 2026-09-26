package messaging

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/mattboback/stageflow/libs/go/events"
	sharedmsg "github.com/mattboback/stageflow/libs/go/messaging"
	"github.com/mattboback/stageflow/libs/go/models"
)

func TestRecoveringHandlerReturnsErrorOnPanic(t *testing.T) {
	const secret = "panic-secret-canary-51c9"

	var logs bytes.Buffer

	previous := slog.Default()

	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	ctx := sharedmsg.WithReceivedEventMeta(context.Background(), &sharedmsg.ReceivedEventMeta{
		Event:       sharedmsg.SubjectJobCreated,
		JobID:       "job-panic",
		Subject:     sharedmsg.SubjectJobCreated,
		Stream:      sharedmsg.StreamJobs,
		Consumer:    "orchestrator-job-created",
		Deliveries:  2,
		StreamSeq:   10,
		ConsumerSeq: 3,
	})

	handler := recoveringHandler(sharedmsg.SubjectJobCreated, func(context.Context, *events.JobCreatedPayload) error {
		panic(secret)
	})

	err := handler(ctx, &events.JobCreatedPayload{
		JobID:     "job-panic",
		InputType: events.InputTypeURLs,
		URLs:      []string{"https://example.com"},
		Config:    models.JobConfig{Modules: []string{"axe"}},
	})
	if err == nil {
		t.Fatal("recoveringHandler() error = nil, want panic error")
	}

	if got, want := err.Error(), "panic handling "+sharedmsg.SubjectJobCreated; got != want {
		t.Fatalf("recoveringHandler() error = %q, want %q", got, want)
	}

	if strings.Contains(err.Error(), secret) || strings.Contains(logs.String(), secret) {
		t.Fatal("panic recovery exposed the secret canary")
	}

	for _, expected := range []string{`"panic_type":"string"`, `"job_id":"job-panic"`, `"stack":`} {
		if !strings.Contains(logs.String(), expected) {
			t.Fatalf("recovery log missing %s", expected)
		}
	}
}
