package storage

import (
	"context"
	"errors"
	"io"
	"sort"

	"github.com/mattboback/stageflow/libs/go/models"
)

const (
	reportVersion = "2.2.0"
)

// JobFiles reads and writes job files. Keys are "<job ID>/<path>".
type JobFiles interface {
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	Write(ctx context.Context, key string, reader io.Reader) error
}

type Aggregator struct {
	files JobFiles
}

func NewAggregator(files JobFiles) *Aggregator {
	return &Aggregator{files: files}
}

func (a *Aggregator) BuildAggregatedReport(ctx context.Context, job *models.Job) (string, error) {
	if job == nil {
		return "", errors.New("job is required")
	}

	if a.files == nil {
		return "", errors.New("job files are required for report aggregation")
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

	if err := a.writeAggregatedReport(ctx, reportKey, report); err != nil {
		return "", err
	}

	return reportKey, nil
}
