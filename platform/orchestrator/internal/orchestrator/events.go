package orchestrator

import (
	"context"

	"github.com/mattboback/stageflow/packages/shared-go/events"
)

func (o *Orchestrator) HandleJobCreated(ctx context.Context, payload *events.JobCreatedPayload) error {
	return o.withInboundEvent(ctx, events.EventJobCreated, payload.JobID, payload, func(ctx context.Context) error {
		return o.newService().HandleJobCreated(ctx, payload)
	})
}

// HandleExtractionReady handles extraction.ready events.
func (o *Orchestrator) HandleExtractionReady(ctx context.Context, payload *events.ExtractionReadyPayload) error {
	return o.withInboundEvent(
		ctx,
		events.EventExtractionReady,
		payload.JobID,
		payload,
		func(ctx context.Context) error {
			return o.newService().HandleExtractionReady(ctx, payload)
		},
	)
}

// HandleExtractionFailed handles extraction.failed events.
func (o *Orchestrator) HandleExtractionFailed(ctx context.Context, payload *events.ExtractionFailedPayload) error {
	return o.withInboundEvent(
		ctx,
		events.EventExtractionFailed,
		payload.JobID,
		payload,
		func(ctx context.Context) error {
			return o.newService().HandleExtractionFailed(ctx, payload)
		},
	)
}

// HandleScanPageCompleted persists scanner progress updates for job status reads.
func (o *Orchestrator) HandleScanPageCompleted(ctx context.Context, payload *events.ScanPageCompletedPayload) error {
	return o.withInboundEvent(
		ctx,
		events.EventScanPageCompleted,
		payload.JobID,
		payload,
		func(ctx context.Context) error {
			return o.newService().HandleScanPageCompleted(ctx, payload)
		},
	)
}

// HandleScanCompleted handles scan.completed events.
func (o *Orchestrator) HandleScanCompleted(ctx context.Context, payload *events.ScanCompletedPayload) error {
	return o.withInboundEvent(ctx, events.EventScanCompleted, payload.JobID, payload, func(ctx context.Context) error {
		return o.newService().HandleScanCompleted(ctx, payload)
	})
}

// HandleScanFailed handles scan.failed events.
func (o *Orchestrator) HandleScanFailed(ctx context.Context, payload *events.ScanFailedPayload) error {
	return o.withInboundEvent(ctx, events.EventScanFailed, payload.JobID, payload, func(ctx context.Context) error {
		return o.newService().HandleScanFailed(ctx, payload)
	})
}
