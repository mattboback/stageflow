// Package messaging wires NATS messaging for the web server.
package messaging

import (
	"context"

	"github.com/mattboback/stageflow/libs/go/events"
	sharedmsg "github.com/mattboback/stageflow/libs/go/messaging"
)

// Service wraps messaging operations for the web server.
type Service struct {
	natsClient *sharedmsg.Client
}

// NewService creates a new messaging service.
func NewService(natsClient *sharedmsg.Client) *Service {
	return &Service{
		natsClient: natsClient,
	}
}

// PublishJobCreated publishes a job.created event.
func (s *Service) PublishJobCreated(ctx context.Context, envelope *events.Envelope) error {
	return s.natsClient.PublishEvent(ctx, sharedmsg.SubjectJobCreated, envelope)
}
