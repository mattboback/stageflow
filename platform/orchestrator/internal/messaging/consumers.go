// Package messaging wires orchestrator event consumers.
package messaging

import (
	"context"
	"log/slog"

	"github.com/mattboback/stageflow/packages/shared-go/events"
	sharedmsg "github.com/mattboback/stageflow/packages/shared-go/messaging"
)

// EventHandler defines orchestrator callbacks for lifecycle events.
type EventHandler interface {
	HandleJobCreated(ctx context.Context, payload *events.JobCreatedPayload) error
	HandleExtractionReady(ctx context.Context, payload *events.ExtractionReadyPayload) error
	HandleExtractionFailed(ctx context.Context, payload *events.ExtractionFailedPayload) error
	HandleScanCompleted(ctx context.Context, payload *events.ScanCompletedPayload) error
	HandleScanFailed(ctx context.Context, payload *events.ScanFailedPayload) error
}

// Consumer subscribes orchestrator handlers to JetStream subjects.
type Consumer struct {
	natsClient *sharedmsg.Client
	handler    EventHandler
}

// NewConsumer wires a shared JetStream client and handler into a Consumer.
func NewConsumer(natsClient *sharedmsg.Client, handler EventHandler) *Consumer {
	return &Consumer{
		natsClient: natsClient,
		handler:    handler,
	}
}

// Start starts consuming events from all streams.
func (c *Consumer) Start(ctx context.Context) error {
	subscriptions := []func(context.Context) error{
		c.subscribeJobCreated,
		c.subscribeExtractionReady,
		c.subscribeExtractionFailed,
		c.subscribeScanCompleted,
		c.subscribeScanFailed,
	}

	for _, sub := range subscriptions {
		if err := sub(ctx); err != nil {
			return err
		}
	}

	slog.Info("Started consuming events from NATS")

	return nil
}

func (c *Consumer) subscribeJobCreated(ctx context.Context) error {
	return sharedmsg.SubscribeTyped(ctx, c.natsClient, sharedmsg.Subscription[events.JobCreatedPayload]{
		Stream:  sharedmsg.StreamJobs,
		Subject: sharedmsg.SubjectJobCreated,
		Durable: "orchestrator-job-created",
		Handler: c.handler.HandleJobCreated,
	})
}

func (c *Consumer) subscribeExtractionReady(ctx context.Context) error {
	return sharedmsg.SubscribeTyped(ctx, c.natsClient, sharedmsg.Subscription[events.ExtractionReadyPayload]{
		Stream:  sharedmsg.StreamExtraction,
		Subject: sharedmsg.SubjectExtractionReady,
		Durable: "orchestrator-extraction-ready",
		Handler: c.handler.HandleExtractionReady,
	})
}

func (c *Consumer) subscribeExtractionFailed(ctx context.Context) error {
	return sharedmsg.SubscribeTyped(ctx, c.natsClient, sharedmsg.Subscription[events.ExtractionFailedPayload]{
		Stream:  sharedmsg.StreamExtraction,
		Subject: sharedmsg.SubjectExtractionFailed,
		Durable: "orchestrator-extraction-failed",
		Handler: c.handler.HandleExtractionFailed,
	})
}

func (c *Consumer) subscribeScanCompleted(ctx context.Context) error {
	return sharedmsg.SubscribeTyped(ctx, c.natsClient, sharedmsg.Subscription[events.ScanCompletedPayload]{
		Stream:  sharedmsg.StreamScan,
		Subject: sharedmsg.SubjectScanCompleted,
		Durable: "orchestrator-scan-completed",
		Handler: c.handler.HandleScanCompleted,
	})
}

func (c *Consumer) subscribeScanFailed(ctx context.Context) error {
	return sharedmsg.SubscribeTyped(ctx, c.natsClient, sharedmsg.Subscription[events.ScanFailedPayload]{
		Stream:  sharedmsg.StreamScan,
		Subject: sharedmsg.SubjectScanFailed,
		Durable: "orchestrator-scan-failed",
		Handler: c.handler.HandleScanFailed,
	})
}
