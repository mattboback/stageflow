package orchestrator

import (
	"context"

	"github.com/mattboback/stageflow/libs/go/events"
)

// HandleJobCreated handles job.created events.
func (o *Orchestrator) HandleJobCreated(ctx context.Context, payload *events.JobCreatedPayload) error {
	return o.newService().HandleJobCreated(ctx, payload)
}

// HandleExtractionReady handles extraction.ready events.
func (o *Orchestrator) HandleExtractionReady(ctx context.Context, payload *events.ExtractionReadyPayload) error {
	return o.newService().HandleExtractionReady(ctx, payload)
}

// HandleExtractionFailed handles extraction.failed events.
func (o *Orchestrator) HandleExtractionFailed(ctx context.Context, payload *events.ExtractionFailedPayload) error {
	return o.newService().HandleExtractionFailed(ctx, payload)
}

// HandleScanPageCompleted handles scanner progress updates for job status reads.
func (o *Orchestrator) HandleScanPageCompleted(ctx context.Context, payload *events.ScanPageCompletedPayload) error {
	return o.newService().HandleScanPageCompleted(ctx, payload)
}

// HandleScanCompleted handles scan.completed events.
func (o *Orchestrator) HandleScanCompleted(ctx context.Context, payload *events.ScanCompletedPayload) error {
	return o.newService().HandleScanCompleted(ctx, payload)
}

// HandleScanFailed handles scan.failed events.
func (o *Orchestrator) HandleScanFailed(ctx context.Context, payload *events.ScanFailedPayload) error {
	return o.newService().HandleScanFailed(ctx, payload)
}
