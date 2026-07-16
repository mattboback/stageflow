package jobs

import (
	"context"
	"log/slog"

	"github.com/mattboback/stageflow/libs/go/events"
)

func (s *Service) HandleScanFailed(
	ctx context.Context,
	payload *events.ScanFailedPayload,
) error {
	if err := validateInboundPayload(events.EventScanFailed, payload); err != nil {
		return err
	}

	slog.Error("Handling scan.failed", "job_id", payload.JobID, "scanner", payload.ScannerType)

	return s.RecordScannerFailure(ctx, payload)
}
