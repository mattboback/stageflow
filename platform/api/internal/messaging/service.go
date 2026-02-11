// Package messaging wires NATS messaging for the web server.
package messaging

import (
	"context"

	"github.com/mattboback/stageflow/packages/shared-go/events"
	sharedmsg "github.com/mattboback/stageflow/packages/shared-go/messaging"
	"github.com/mattboback/stageflow/platform/api/internal/sse"
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

// SSEBroadcastHandler broadcasts lifecycle updates to SSE clients.
type SSEBroadcastHandler struct {
	sseHub *sse.Hub
}

// NewSSEBroadcastHandler creates a new handler that broadcasts to SSE clients.
func NewSSEBroadcastHandler(sseHub *sse.Hub) *SSEBroadcastHandler {
	return &SSEBroadcastHandler{
		sseHub: sseHub,
	}
}

func (h *SSEBroadcastHandler) HandleJobCreated(_ context.Context, p *events.JobCreatedPayload) error {
	h.sseHub.Broadcast(p.JobID, map[string]any{
		"type":  "status",
		"state": "PENDING",
	})

	return nil
}

func (h *SSEBroadcastHandler) HandleExtractionReady(_ context.Context, p *events.ExtractionReadyPayload) error {
	h.sseHub.Broadcast(p.JobID, map[string]any{
		"type":       "status",
		"state":      "READY_TO_SCAN",
		"totalPages": p.TotalPages,
	})

	return nil
}

func (h *SSEBroadcastHandler) HandleExtractionFailed(_ context.Context, p *events.ExtractionFailedPayload) error {
	h.sseHub.Broadcast(p.JobID, map[string]any{
		"type":          "failed",
		"state":         "FAILED",
		"error":         p.Error,
		"error_details": p.ErrorDetails,
		"stage":         "extraction",
	})

	return nil
}

func (h *SSEBroadcastHandler) HandleScanPageCompleted(_ context.Context, p *events.ScanPageCompletedPayload) error {
	h.sseHub.Broadcast(p.JobID, map[string]any{
		"type":  "progress",
		"state": "SCANNING",
		"progress": map[string]int{
			"currentPage": p.PageIndex,
			"totalPages":  p.TotalPages,
		},
	})

	return nil
}

func (h *SSEBroadcastHandler) HandleScanCompleted(_ context.Context, p *events.ScanCompletedPayload) error {
	h.sseHub.Broadcast(p.JobID, map[string]any{
		"type":  "status",
		"state": "COMPLETING",
	})

	return nil
}

func (h *SSEBroadcastHandler) HandleScanFailed(_ context.Context, p *events.ScanFailedPayload) error {
	h.sseHub.Broadcast(p.JobID, map[string]any{
		"type":  "failed",
		"state": "FAILED",
		"error": p.Error,
	})

	return nil
}

func (h *SSEBroadcastHandler) HandleJobCompleted(_ context.Context, p *events.JobCompletedPayload) error {
	h.sseHub.Broadcast(p.JobID, map[string]any{
		"type":  "complete",
		"state": "DONE",
	})

	return nil
}

func (h *SSEBroadcastHandler) HandleJobFailed(_ context.Context, p *events.JobFailedPayload) error {
	h.sseHub.Broadcast(p.JobID, map[string]any{
		"type":  "failed",
		"state": "FAILED",
		"error": p.Error,
	})

	return nil
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
