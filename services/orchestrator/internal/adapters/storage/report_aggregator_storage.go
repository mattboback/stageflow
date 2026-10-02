package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
)

func (a *Aggregator) readScanResults(ctx context.Context, resultsPath string) (*report.UnifiedReportV2, error) {
	reader, err := a.files.Open(ctx, resultsPath)
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

func (a *Aggregator) writeAggregatedReport(
	ctx context.Context,
	key string,
	aggregatedReport report.UnifiedReportV2,
) error {
	blob, err := json.MarshalIndent(aggregatedReport, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal aggregated report: %w", err)
	}

	if writeErr := a.files.Write(ctx, key, bytes.NewReader(blob)); writeErr != nil {
		return fmt.Errorf("write aggregated report: %w", writeErr)
	}

	return nil
}
