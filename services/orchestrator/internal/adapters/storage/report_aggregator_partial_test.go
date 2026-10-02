package storage

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/mattboback/stageflow/internal/artifacts"
	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
	"github.com/mattboback/stageflow/libs/go/models"
	scanners "github.com/mattboback/stageflow/libs/go/scannerregistry"
)

// buildPartialTestReport aggregates one successful SEO result with a serious
// issue, plus whatever axe result the caller passes (nil: axe never reported).
func buildPartialTestReport(t *testing.T, axe *models.ScannerResult) report.UnifiedReportV2 {
	t.Helper()

	ctx := context.Background()
	mem := newMemoryStorage()
	registry := scanners.NewRegistry("scanner-runner")
	_ = registry.Register(&scanners.Definition{ID: "axe", Name: "Accessibility", Enabled: true})
	_ = registry.Register(&scanners.Definition{ID: "seo", Name: "SEO", Enabled: true})

	now := time.Now().UTC()
	seoResults := report.UnifiedReportV2{
		Version: reportVersion,
		Meta:    report.ReportMeta{JobId: "job-p", ScannedAt: &now, CompletedAt: &now},
		Summary: report.ReportSummary{
			TotalIssues:  1,
			BySeverity:   report.SeverityCounts{Serious: 1},
			PagesScanned: 1,
		},
		Pages: []report.PageSummary{{Id: "page-1", Url: "https://example.com/", IssueCount: 1}},
		Issues: []report.IssueDetail{{
			Id:           "seo-1",
			Scanner:      "seo",
			RuleId:       "missing-description",
			Severity:     report.IssueSeveritySerious,
			Title:        "Missing description",
			Description:  "The page has no meta description.",
			PageId:       "page-1",
			PageUrl:      "https://example.com/",
			ElementCount: 1,
		}},
		Scanners: []report.ScannerSummary{{Id: "seo", Status: report.ScannerStatusSuccess}},
	}

	data, err := json.Marshal(seoResults)
	if err != nil {
		t.Fatalf("marshal seo results: %v", err)
	}

	mem.objects[mem.key(artifacts.BucketArtifacts, "job-p/seo/results.json")] = data

	job := &models.Job{
		ID:               "job-p",
		ExpectedScanners: []string{"axe", "seo"},
		ScannerResults: map[string]*models.ScannerResult{
			"seo": {ScannerType: "seo", ResultsPath: "job-p/seo/results.json", Success: true},
		},
	}
	if axe != nil {
		job.ScannerResults["axe"] = axe
	}

	key, err := NewAggregator(mem, registry).BuildAggregatedReport(ctx, job)
	if err != nil {
		t.Fatalf("BuildAggregatedReport: %v", err)
	}

	blob := mem.objects[mem.key(artifacts.BucketArtifacts, key)]

	var aggregated report.UnifiedReportV2
	if decodeErr := json.Unmarshal(blob, &aggregated); decodeErr != nil {
		t.Fatalf("unmarshal report: %v", decodeErr)
	}

	return aggregated
}

func TestBuildAggregatedReportWithholdsScoreWhenAScannerFailed(t *testing.T) {
	aggregated := buildPartialTestReport(t, &models.ScannerResult{
		ScannerType: "axe",
		Success:     false,
		Error:       "axe ran out of memory (limit 3072 MB)",
	})

	if aggregated.Summary.Score != nil || aggregated.Summary.ScoreGrade != nil {
		t.Fatalf("score = %v grade = %v, want none on a partial report",
			aggregated.Summary.Score, aggregated.Summary.ScoreGrade)
	}

	if aggregated.Summary.TotalIssues != 1 {
		t.Fatalf("total issues = %d, want the successful scanner's 1", aggregated.Summary.TotalIssues)
	}
}

func TestBuildAggregatedReportWithholdsScoreWhenAScannerNeverReported(t *testing.T) {
	aggregated := buildPartialTestReport(t, nil)

	if aggregated.Summary.Score != nil {
		t.Fatalf("score = %d, want none when an expected scanner is skipped", *aggregated.Summary.Score)
	}
}
