package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
	sharedstorage "github.com/mattboback/stageflow/libs/go/storage"
)

func (a *Aggregator) downloadScanResults(ctx context.Context, resultsPath string) (*report.UnifiedReportV2, error) {
	reader, err := a.storage.DownloadFile(ctx, sharedstorage.BucketArtifacts, resultsPath)
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

func (a *Aggregator) uploadAggregatedReport(
	ctx context.Context,
	key string,
	aggregatedReport report.UnifiedReportV2,
) error {
	blob, err := json.MarshalIndent(aggregatedReport, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal aggregated report: %w", err)
	}

	if uploadErr := a.storage.UploadFile(
		ctx,
		sharedstorage.BucketArtifacts,
		key,
		bytes.NewReader(blob),
		int64(len(blob)),
	); uploadErr != nil {
		return fmt.Errorf("upload aggregated report: %w", uploadErr)
	}

	return nil
}
