package storage

import (
	"context"
	"errors"
	"sort"

	"github.com/mattboback/stageflow/libs/go/models"
	scanners "github.com/mattboback/stageflow/libs/go/scannerregistry"
	sharedstorage "github.com/mattboback/stageflow/libs/go/storage"
)

const (
	reportVersion = "2.0.0"
)

type Aggregator struct {
	storage         sharedstorage.Client
	scannerRegistry *scanners.Registry
}

func NewAggregator(storageClient sharedstorage.Client, scannerRegistry *scanners.Registry) *Aggregator {
	return &Aggregator{
		storage:         storageClient,
		scannerRegistry: scannerRegistry,
	}
}

func (a *Aggregator) BuildAggregatedReport(ctx context.Context, job *models.Job) (string, error) {
	if job == nil {
		return "", errors.New("job is required")
	}

	if a.storage == nil {
		return "", errors.New("storage client is required for report aggregation")
	}

	if len(job.ScannerResults) == 0 {
		return "", errors.New("no scanner results found for aggregation")
	}

	agg := newReportAggregation(a, job)
	agg.initExpectedScanners()

	scannerIDs := make([]string, 0, len(job.ScannerResults))
	for scannerID := range job.ScannerResults {
		scannerIDs = append(scannerIDs, scannerID)
	}

	sort.Strings(scannerIDs)

	for _, scannerID := range scannerIDs {
		agg.aggregateScanner(ctx, scannerID, job.ScannerResults[scannerID])
	}

	agg.finalizeExpectedScanners()

	if !agg.hadSuccess {
		return "", errors.New("no successful scanner results to aggregate")
	}

	report := agg.buildReport()
	reportKey := job.ID + "/report.json"

	if err := a.uploadAggregatedReport(ctx, reportKey, report); err != nil {
		return "", err
	}

	return reportKey, nil
}
