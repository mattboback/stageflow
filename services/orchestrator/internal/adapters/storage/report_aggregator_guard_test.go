package storage

import (
	"context"
	"testing"

	"github.com/mattboback/stageflow/libs/go/models"
	scanners "github.com/mattboback/stageflow/libs/go/scannerregistry"
)

func TestBuildAggregatedReportRejectsNilJob(t *testing.T) {
	aggregator := NewAggregator(newMemoryStorage(), scanners.NewRegistry("scanner-runner"))

	if _, err := aggregator.BuildAggregatedReport(context.Background(), nil); err == nil {
		t.Fatal("expected error for nil job")
	}
}

func TestBuildAggregatedReportRejectsNilStorage(t *testing.T) {
	aggregator := NewAggregator(nil, scanners.NewRegistry("scanner-runner"))

	job := &models.Job{
		ID:             "job-nil-storage",
		ScannerResults: map[string]*models.ScannerResult{"axe": {ScannerType: "axe"}},
	}

	if _, err := aggregator.BuildAggregatedReport(context.Background(), job); err == nil {
		t.Fatal("expected error for nil storage client")
	}
}

func TestBuildAggregatedReportRejectsEmptyScannerResults(t *testing.T) {
	aggregator := NewAggregator(newMemoryStorage(), scanners.NewRegistry("scanner-runner"))

	job := &models.Job{ID: "job-no-results"}

	if _, err := aggregator.BuildAggregatedReport(context.Background(), job); err == nil {
		t.Fatal("expected error when no scanner results are present")
	}
}
