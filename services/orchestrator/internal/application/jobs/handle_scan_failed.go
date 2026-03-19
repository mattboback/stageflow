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
	slog.Error("Handling scan.failed", "job_id", payload.JobID, "scanner", payload.ScannerType, "error", payload.Error)
	return s.RecordScannerFailure(ctx, payload)
}
