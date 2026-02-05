package orchestrator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	report "github.com/mattboback/stageflow/packages/contracts/report/generated/go"
	"github.com/mattboback/stageflow/packages/shared-go/storage"
)

func (o *Orchestrator) downloadScanResults(ctx context.Context, resultsPath string) (*report.UnifiedReportV2, error) {
	reader, err := o.storage.DownloadFile(ctx, storage.BucketArtifacts, resultsPath)
	if err != nil {
		return nil, fmt.Errorf("download results.json: %w", err)
	}

	defer func() {
		_ = reader.Close()
	}()

	var results report.UnifiedReportV2
	if decodeErr := json.NewDecoder(reader).Decode(&results); decodeErr != nil {
		return nil, fmt.Errorf("decode results.json: %w", decodeErr)
	}

	return &results, nil
}

func (o *Orchestrator) uploadAggregatedReport(
	ctx context.Context,
	key string,
	aggregatedReport report.UnifiedReportV2,
) error {
	blob, err := json.MarshalIndent(aggregatedReport, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal aggregated report: %w", err)
	}

	if uploadErr := o.storage.UploadFile(
		ctx,
		storage.BucketArtifacts,
		key,
		bytes.NewReader(blob),
		int64(len(blob)),
	); uploadErr != nil {
		return fmt.Errorf("upload aggregated report: %w", uploadErr)
	}

	return nil
}
