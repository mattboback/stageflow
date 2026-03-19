package jobs

import (
	"context"

	"github.com/mattboback/stageflow/libs/go/events"
)

func (s *Service) HandleScanCompleted(
	ctx context.Context,
	payload *events.ScanCompletedPayload,
) error {
	return s.RecordScannerCompletion(ctx, payload)
}
