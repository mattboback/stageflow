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

// EventHandler handles lifecycle updates from messaging subscriptions.
type EventHandler interface {
	HandleJobCreated(ctx context.Context, payload *events.JobCreatedPayload) error
	HandleExtractionReady(ctx context.Context, payload *events.ExtractionReadyPayload) error
	HandleExtractionFailed(ctx context.Context, payload *events.ExtractionFailedPayload) error
	HandleScanPageCompleted(ctx context.Context, payload *events.ScanPageCompletedPayload) error
	HandleScanCompleted(ctx context.Context, payload *events.ScanCompletedPayload) error
	HandleScanFailed(ctx context.Context, payload *events.ScanFailedPayload) error
	HandleJobCompleted(ctx context.Context, payload *events.JobCompletedPayload) error
	HandleJobFailed(ctx context.Context, payload *events.JobFailedPayload) error
}

// PublishJobCreated publishes a job.created event.
func (s *Service) PublishJobCreated(ctx context.Context, envelope *events.Envelope) error {
	return s.natsClient.PublishEvent(ctx, sharedmsg.SubjectJobCreated, envelope)
}

// SubscribeToStatusEvents wires all lifecycle subjects to the projection handler.
func (s *Service) SubscribeToStatusEvents(ctx context.Context, handler EventHandler) error {
	if err := sharedmsg.SubscribeTyped(ctx, s.natsClient, sharedmsg.Subscription[events.JobCreatedPayload]{
		Stream: sharedmsg.StreamJobs, Subject: sharedmsg.SubjectJobCreated,
		Durable: "platform-api-job-created", Handler: handler.HandleJobCreated,
	}); err != nil {
		return err
	}

	if err := sharedmsg.SubscribeTyped(ctx, s.natsClient, sharedmsg.Subscription[events.ExtractionReadyPayload]{
		Stream: sharedmsg.StreamExtraction, Subject: sharedmsg.SubjectExtractionReady,
		Durable: "platform-api-extraction-ready", Handler: handler.HandleExtractionReady,
	}); err != nil {
		return err
	}

	if err := sharedmsg.SubscribeTyped(ctx, s.natsClient, sharedmsg.Subscription[events.ExtractionFailedPayload]{
		Stream: sharedmsg.StreamExtraction, Subject: sharedmsg.SubjectExtractionFailed,
		Durable: "platform-api-extraction-failed", Handler: handler.HandleExtractionFailed,
	}); err != nil {
		return err
	}

	if err := sharedmsg.SubscribeTyped(ctx, s.natsClient, sharedmsg.Subscription[events.ScanPageCompletedPayload]{
		Stream: sharedmsg.StreamScan, Subject: sharedmsg.SubjectScanPageCompleted,
		Durable: "platform-api-scan-page", Handler: handler.HandleScanPageCompleted,
	}); err != nil {
		return err
	}

	if err := sharedmsg.SubscribeTyped(ctx, s.natsClient, sharedmsg.Subscription[events.ScanCompletedPayload]{
		Stream: sharedmsg.StreamScan, Subject: sharedmsg.SubjectScanCompleted,
		Durable: "platform-api-scan-completed", Handler: handler.HandleScanCompleted,
	}); err != nil {
		return err
	}

	if err := sharedmsg.SubscribeTyped(ctx, s.natsClient, sharedmsg.Subscription[events.ScanFailedPayload]{
		Stream: sharedmsg.StreamScan, Subject: sharedmsg.SubjectScanFailed,
		Durable: "platform-api-scan-failed", Handler: handler.HandleScanFailed,
	}); err != nil {
		return err
	}

	if err := sharedmsg.SubscribeTyped(ctx, s.natsClient, sharedmsg.Subscription[events.JobCompletedPayload]{
		Stream: sharedmsg.StreamJobs, Subject: sharedmsg.SubjectJobCompleted,
		Durable: "platform-api-job-completed", Handler: handler.HandleJobCompleted,
	}); err != nil {
		return err
	}

	if err := sharedmsg.SubscribeTyped(ctx, s.natsClient, sharedmsg.Subscription[events.JobFailedPayload]{
		Stream: sharedmsg.StreamJobs, Subject: sharedmsg.SubjectJobFailed,
		Durable: "platform-api-job-failed", Handler: handler.HandleJobFailed,
	}); err != nil {
		return err
	}

	return nil
}
