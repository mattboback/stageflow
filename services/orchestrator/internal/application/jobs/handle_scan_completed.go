package jobs

import (
	"context"

	"github.com/mattboback/stageflow/libs/go/events"
)

func (s *Service) HandleScanCompleted(
	ctx context.Context,
	payload *events.ScanCompletedPayload,
) error {
	if err := validateInboundPayload(events.EventScanCompleted, payload); err != nil {
		return err
	}

	return s.RecordScannerCompletion(ctx, payload)
}
