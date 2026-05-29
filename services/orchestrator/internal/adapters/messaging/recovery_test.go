package messaging

import (
	"context"
	"strings"
	"testing"

	"github.com/mattboback/stageflow/libs/go/events"
	sharedmsg "github.com/mattboback/stageflow/libs/go/messaging"
)

func TestRecoveringHandlerReturnsErrorOnPanic(t *testing.T) {
	t.Parallel()

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
		panic("boom")
	})

	err := handler(ctx, &events.JobCreatedPayload{JobID: "job-panic"})
	if err == nil {
		t.Fatal("recoveringHandler() error = nil, want panic error")
	}

	if !strings.Contains(err.Error(), "panic handling "+sharedmsg.SubjectJobCreated) ||
		!strings.Contains(err.Error(), "boom") {
		t.Fatalf("recoveringHandler() error = %q, want panic details", err.Error())
	}
}
