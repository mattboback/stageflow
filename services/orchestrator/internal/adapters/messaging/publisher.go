package messaging

import (
	"context"

	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/logging"
	sharedmsg "github.com/mattboback/stageflow/libs/go/messaging"
)

// Publisher publishes orchestrator lifecycle envelopes to NATS.
type Publisher struct {
	natsClient *sharedmsg.Client
}

// NewPublisher wires a shared JetStream client into a Publisher.
func NewPublisher(natsClient *sharedmsg.Client) *Publisher {
	return &Publisher{
		natsClient: natsClient,
	}
}

// PublishJobCompleted publishes a job.completed envelope.
func (p *Publisher) PublishJobCompleted(ctx context.Context, payload *events.JobCompletedPayload) error {
	envelope := events.NewEnvelope(events.EventJobCompleted, payload.JobID, "orchestrator", payload)
	envelope.RequestID = logging.RequestID(ctx)
	envelope.RunID = logging.RunID(ctx)

	return p.natsClient.PublishEvent(ctx, sharedmsg.SubjectJobCompleted, envelope)
}

// PublishJobFailed publishes a job.failed envelope.
func (p *Publisher) PublishJobFailed(ctx context.Context, payload *events.JobFailedPayload) error {
	envelope := events.NewEnvelope(events.EventJobFailed, payload.JobID, "orchestrator", payload)
	envelope.RequestID = logging.RequestID(ctx)
	envelope.RunID = logging.RunID(ctx)

	return p.natsClient.PublishEvent(ctx, sharedmsg.SubjectJobFailed, envelope)
}

// PublishJobCreated publishes a job.created envelope.
// This is typically done by the platform-api, but the orchestrator test API also needs this.
func (p *Publisher) PublishJobCreated(ctx context.Context, payload *events.JobCreatedPayload) error {
	envelope := events.NewEnvelope(events.EventJobCreated, payload.JobID, "orchestrator", payload)
	envelope.RequestID = logging.RequestID(ctx)
	envelope.RunID = logging.RunID(ctx)

	return p.natsClient.PublishEvent(ctx, sharedmsg.SubjectJobCreated, envelope)
}
