package jobstatus

import (
	"context"
	"time"

	"github.com/mattboback/stageflow/libs/go/events"
	sharedmsg "github.com/mattboback/stageflow/libs/go/messaging"
)

type EventHandler struct {
	pipeline JobStatusPipeline
}

func NewEventHandler(pipeline JobStatusPipeline) *EventHandler {
	return &EventHandler{pipeline: pipeline}
}

func (h *EventHandler) HandleJobCreated(ctx context.Context, payload *events.JobCreatedPayload) error {
	if _, err := h.pipeline.Begin(ctx, BeginJob{Payload: payload, ObservedAt: observedAtFromContext(ctx)}); err != nil {
		return err
	}

	_, err := h.pipeline.Apply(ctx, Signal{
		Kind:       SignalJobCreated,
		ObservedAt: observedAtFromContext(ctx),
		JobCreated: payload,
	})

	return err
}

func (h *EventHandler) HandleExtractionReady(ctx context.Context, payload *events.ExtractionReadyPayload) error {
	_, err := h.pipeline.Apply(ctx, Signal{
		Kind:            SignalExtractionReady,
		ObservedAt:      observedAtFromContext(ctx),
		ExtractionReady: payload,
	})

	return err
}

func (h *EventHandler) HandleExtractionFailed(ctx context.Context, payload *events.ExtractionFailedPayload) error {
	_, err := h.pipeline.Apply(ctx, Signal{
		Kind:             SignalExtractionFailed,
		ObservedAt:       observedAtFromContext(ctx),
		ExtractionFailed: payload,
	})

	return err
}

func (h *EventHandler) HandleScanPageCompleted(ctx context.Context, payload *events.ScanPageCompletedPayload) error {
	_, err := h.pipeline.Apply(ctx, Signal{
		Kind:              SignalScanPageCompleted,
		ObservedAt:        observedAtFromContext(ctx),
		ScanPageCompleted: payload,
	})

	return err
}

func (h *EventHandler) HandleScanCompleted(ctx context.Context, payload *events.ScanCompletedPayload) error {
	_, err := h.pipeline.Apply(ctx, Signal{
		Kind:          SignalScanCompleted,
		ObservedAt:    observedAtFromContext(ctx),
		ScanCompleted: payload,
	})

	return err
}

func (h *EventHandler) HandleScanFailed(ctx context.Context, payload *events.ScanFailedPayload) error {
	_, err := h.pipeline.Apply(ctx, Signal{
		Kind:       SignalScanFailed,
		ObservedAt: observedAtFromContext(ctx),
		ScanFailed: payload,
	})

	return err
}

func (h *EventHandler) HandleJobCompleted(ctx context.Context, payload *events.JobCompletedPayload) error {
	_, err := h.pipeline.Apply(ctx, Signal{
		Kind:         SignalJobCompleted,
		ObservedAt:   observedAtFromContext(ctx),
		JobCompleted: payload,
	})

	return err
}

func (h *EventHandler) HandleJobFailed(ctx context.Context, payload *events.JobFailedPayload) error {
	_, err := h.pipeline.Apply(ctx, Signal{
		Kind:       SignalJobFailed,
		ObservedAt: observedAtFromContext(ctx),
		JobFailed:  payload,
	})

	return err
}

func observedAtFromContext(ctx context.Context) time.Time {
	if meta, ok := sharedmsg.ReceivedEventMetaFromContext(ctx); ok && !meta.EnvelopeTimestamp.IsZero() {
		return meta.EnvelopeTimestamp
	}

	return time.Now().UTC()
}
