package aggregate

import (
	"context"
	"testing"

	"github.com/mattboback/stageflow/libs/go/models"
)

func TestBuildAggregatedReportRejectsNilJob(t *testing.T) {
	aggregator := New(newMemoryFiles())

	if _, err := aggregator.BuildAggregatedReport(context.Background(), nil); err == nil {
		t.Fatal("expected error for nil job")
	}
}

func TestBuildAggregatedReportRejectsNilFiles(t *testing.T) {
	aggregator := New(nil)

	job := &models.Job{
		ID:             "job-nil-files",
		ScannerResults: map[string]*models.ScannerResult{"axe": {ScannerType: "axe"}},
	}

	if _, err := aggregator.BuildAggregatedReport(context.Background(), job); err == nil {
		t.Fatal("expected error for nil job files")
	}
}

func TestBuildAggregatedReportRejectsEmptyScannerResults(t *testing.T) {
	aggregator := New(newMemoryFiles())

	job := &models.Job{ID: "job-no-results"}

	if _, err := aggregator.BuildAggregatedReport(context.Background(), job); err == nil {
		t.Fatal("expected error when no scanner results are present")
	}
}
