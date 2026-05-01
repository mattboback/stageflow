package storage

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
	"github.com/mattboback/stageflow/libs/go/models"
	scanners "github.com/mattboback/stageflow/libs/go/scannerregistry"
	sharedstorage "github.com/mattboback/stageflow/libs/go/storage"
)

//nolint:gocyclo // Test function covers multiple aggregation scenarios
func TestBuildAggregatedReportSuccess(t *testing.T) {
	ctx := context.Background()
	mem := newMemoryStorage()
	registry := scanners.NewRegistry("scanner-runner")
	_ = registry.Register(&scanners.Definition{ID: "axe", Name: "Accessibility", Enabled: true})
	_ = registry.Register(&scanners.Definition{ID: "lighthouse", Name: "Lighthouse", Enabled: true})

	aggregator := NewAggregator(mem, registry)

	now := time.Now().UTC()

	axeResults := report.UnifiedReportV2{
		Version: reportVersion,
		Meta: report.ReportMeta{
			JobId:       "job-1",
			BaseUrl:     stringPtr("https://example.com"),
			ScannedAt:   &now,
			CompletedAt: &now,
		},
		Summary: report.ReportSummary{
			TotalIssues: 2,
			BySeverity: report.SeverityCounts{
				Critical: 1,
				Serious:  1,
				Moderate: 0,
				Minor:    0,
			},
			PagesScanned:    1,
			PagesWithIssues: 1,
		},
		Pages: []report.PageSummary{
			{
				Id:         "page-1",
				Url:        "https://example.com/",
				Path:       stringPtr("/"),
				IssueCount: 2,
				BySeverity: &report.SeverityCounts{Critical: 1, Serious: 1, Moderate: 0, Minor: 0},
				DurationMs: 0,
			},
		},
		Issues: []report.IssueDetail{
			{
				Id:           "axe-1",
				Scanner:      "axe",
				RuleId:       "color-contrast",
				Severity:     report.IssueSeveritySerious,
				Title:        "Color contrast",
				Description:  "Insufficient color contrast",
				HelpUrl:      stringPtr("https://dequeuniversity.com/rules/axe/4.0/color-contrast"),
				PageId:       "page-1",
				PageUrl:      "https://example.com/",
				ElementCount: 1,
			},
			{
				Id:           "axe-2",
				Scanner:      "axe",
				RuleId:       "image-alt",
				Severity:     report.IssueSeverityCritical,
				Title:        "Images must have alt text",
				Description:  "Missing alt text",
				PageId:       "page-1",
				PageUrl:      "https://example.com/",
				ElementCount: 1,
				Occurrences: []report.IssueOccurrence{
					{Selector: stringPtr("img.hero")},
				},
			},
		},
		Scanners: []report.ScannerSummary{
			{
				Id:          "axe",
				Status:      report.ScannerStatusSuccess,
				IssueCount:  intPtr(2),
				Severity:    &report.SeverityCounts{Critical: 1, Serious: 1, Moderate: 0, Minor: 0},
				ToolVersion: stringPtr("test"),
				StartedAt:   &now,
				CompletedAt: &now,
				DurationMs:  floatPtr(100),
				ResultsPath: stringPtr("job-1/axe/results.json"),
				ReportPath:  stringPtr("job-1/axe/report.html"),
			},
		},
		Artifacts: []report.ReportArtifact{
			{Id: "axe-report", Type: "html", Path: stringPtr("job-1/axe/report.html"), Mime: stringPtr("text/html")},
		},
		Errors: nil,
	}

	lighthouseResults := report.UnifiedReportV2{
		Version: reportVersion,
		Meta: report.ReportMeta{
			JobId:       "job-1",
			BaseUrl:     stringPtr("https://example.com"),
			ScannedAt:   &now,
			CompletedAt: &now,
		},
		Summary: report.ReportSummary{
			TotalIssues: 2,
			BySeverity: report.SeverityCounts{
				Critical: 0,
				Serious:  0,
				Moderate: 1,
				Minor:    1,
			},
			PagesScanned:    2,
			PagesWithIssues: 2,
		},
		Pages: []report.PageSummary{
			{
				Id:         "page-1",
				Url:        "https://example.com/",
				Path:       stringPtr("/"),
				IssueCount: 1,
				BySeverity: &report.SeverityCounts{Critical: 0, Serious: 0, Moderate: 1, Minor: 0},
				DurationMs: 0,
			},
			{
				Id:         "page-2",
				Url:        "https://example.com/about",
				Path:       stringPtr("/about"),
				IssueCount: 1,
				BySeverity: &report.SeverityCounts{Critical: 0, Serious: 0, Moderate: 0, Minor: 1},
				DurationMs: 0,
			},
		},
		Issues: []report.IssueDetail{
			{
				Id:           "lh-1",
				Scanner:      "lighthouse",
				RuleId:       "lh-performance",
				Severity:     report.IssueSeverityModerate,
				Title:        "Performance opportunity",
				Description:  "Improve performance",
				PageId:       "page-1",
				PageUrl:      "https://example.com/",
				ElementCount: 1,
			},
			{
				Id:           "lh-2",
				Scanner:      "lighthouse",
				RuleId:       "lh-seo",
				Severity:     report.IssueSeverityMinor,
				Title:        "SEO opportunity",
				Description:  "Improve SEO",
				PageId:       "page-2",
				PageUrl:      "https://example.com/about",
				ElementCount: 1,
			},
		},
		Scanners: []report.ScannerSummary{
			{
				Id:          "lighthouse",
				Status:      report.ScannerStatusSuccess,
				IssueCount:  intPtr(2),
				Severity:    &report.SeverityCounts{Critical: 0, Serious: 0, Moderate: 1, Minor: 1},
				ToolVersion: stringPtr("test"),
				StartedAt:   &now,
				CompletedAt: &now,
				DurationMs:  floatPtr(200),
				ResultsPath: stringPtr("job-1/lighthouse/results.json"),
				ReportPath:  stringPtr("job-1/lighthouse/report.html"),
			},
		},
		Artifacts: []report.ReportArtifact{
			{
				Id:   "lh-report",
				Type: "html",
				Path: stringPtr("job-1/lighthouse/report.html"),
				Mime: stringPtr("text/html"),
			},
		},
		Errors: nil,
	}

	axeBytes, err := json.Marshal(axeResults)
	if err != nil {
		t.Fatalf("marshal axe results: %v", err)
	}

	lhBytes, err := json.Marshal(lighthouseResults)
	if err != nil {
		t.Fatalf("marshal lighthouse results: %v", err)
	}

	mem.objects[mem.key(sharedstorage.BucketArtifacts, "job-1/axe/results.json")] = axeBytes
	mem.objects[mem.key(sharedstorage.BucketArtifacts, "job-1/lighthouse/results.json")] = lhBytes

	job := &models.Job{
		ID:               "job-1",
		ExpectedScanners: []string{"axe", "lighthouse"},
		ScannerResults: map[string]*models.ScannerResult{
			"axe": {
				ScannerType: "axe",
				ResultsPath: "job-1/axe/results.json",
				ReportPath:  "job-1/axe/report.html",
				Success:     true,
			},
			"lighthouse": {
				ScannerType: "lighthouse",
				ResultsPath: "job-1/lighthouse/results.json",
				ReportPath:  "job-1/lighthouse/report.html",
				Success:     true,
			},
		},
	}

	reportKey, err := aggregator.BuildAggregatedReport(ctx, job)
	if err != nil {
		t.Fatalf("buildAggregatedReport: %v", err)
	}

	if reportKey != "job-1/report.json" {
		t.Fatalf("expected report key job-1/report.json, got %s", reportKey)
	}

	reportBytes := mem.objects[mem.key(sharedstorage.BucketArtifacts, reportKey)]

	var aggReport report.UnifiedReportV2
	if unmarshalErr := json.Unmarshal(reportBytes, &aggReport); unmarshalErr != nil {
		t.Fatalf("unmarshal report: %v", unmarshalErr)
	}

	if aggReport.Version != reportVersion {
		t.Fatalf("expected version %s, got %s", reportVersion, aggReport.Version)
	}

	if aggReport.Meta.JobId != "job-1" {
		t.Fatalf("expected job-1, got %s", aggReport.Meta.JobId)
	}

	if aggReport.Summary.TotalIssues != 4 {
		t.Fatalf("expected 4 total issues, got %d", aggReport.Summary.TotalIssues)
	}

	if aggReport.Summary.ByScanner["axe"] != 2 || aggReport.Summary.ByScanner["lighthouse"] != 2 {
		t.Fatalf("unexpected byScanner counts: %#v", aggReport.Summary.ByScanner)
	}

	if aggReport.Summary.PagesScanned != 2 {
		t.Fatalf("expected 2 pages, got %d", aggReport.Summary.PagesScanned)
	}

	page1 := aggReport.Pages[0]
	if page1.Id != "page-1" || page1.IssueCount != 3 {
		t.Fatalf("expected merged page-1 with 3 issues, got %#v", page1)
	}

	if aggReport.Summary.BySeverity.Critical != 1 || aggReport.Summary.BySeverity.Serious != 1 ||
		aggReport.Summary.BySeverity.Moderate != 1 ||
		aggReport.Summary.BySeverity.Minor != 1 {
		t.Fatalf("unexpected severity totals: %#v", aggReport.Summary.BySeverity)
	}
}

func TestBuildAggregatedReportDedupByScanner(t *testing.T) {
	ctx := context.Background()
	mem := newMemoryStorage()
	registry := scanners.NewRegistry("scanner-runner")
	_ = registry.Register(&scanners.Definition{ID: "axe", Name: "Accessibility", Enabled: true})
	_ = registry.Register(&scanners.Definition{ID: "lighthouse", Name: "Lighthouse", Enabled: true})

	aggregator := NewAggregator(mem, registry)

	now := time.Now().UTC()

	// Both scanners report the same image-alt issue on the same page.
	axeResults := report.UnifiedReportV2{
		Version: reportVersion,
		Meta: report.ReportMeta{
			JobId:       "job-dedup",
			BaseUrl:     stringPtr("https://example.com"),
			ScannedAt:   &now,
			CompletedAt: &now,
		},
		Summary: report.ReportSummary{
			TotalIssues: 1,
			BySeverity: report.SeverityCounts{
				Critical: 1,
			},
			PagesScanned:    1,
			PagesWithIssues: 1,
		},
		Pages: []report.PageSummary{
			{
				Id:         "page-1",
				Url:        "https://example.com/",
				IssueCount: 1,
				DurationMs: 0,
			},
		},
		Issues: []report.IssueDetail{
			{
				Id:           "axe-1",
				Scanner:      "axe",
				RuleId:       "image-alt",
				Severity:     report.IssueSeverityCritical,
				Title:        "Image alt",
				Description:  "Missing alt",
				PageId:       "page-1",
				PageUrl:      "https://example.com/",
				ElementCount: 1,
				Occurrences: []report.IssueOccurrence{
					{Selector: stringPtr("img.hero")},
				},
			},
		},
		Scanners: []report.ScannerSummary{
			{
				Id:          "axe",
				Status:      report.ScannerStatusSuccess,
				IssueCount:  intPtr(1),
				StartedAt:   &now,
				CompletedAt: &now,
				DurationMs:  floatPtr(100),
			},
		},
	}

	lighthouseResults := report.UnifiedReportV2{
		Version: reportVersion,
		Meta: report.ReportMeta{
			JobId:       "job-dedup",
			BaseUrl:     stringPtr("https://example.com"),
			ScannedAt:   &now,
			CompletedAt: &now,
		},
		Summary: report.ReportSummary{
			TotalIssues: 1,
			BySeverity: report.SeverityCounts{
				Critical: 1,
			},
			PagesScanned:    1,
			PagesWithIssues: 1,
		},
		Pages: []report.PageSummary{
			{
				Id:         "page-1",
				Url:        "https://example.com/",
				IssueCount: 1,
				DurationMs: 0,
			},
		},
		Issues: []report.IssueDetail{
			{
				Id:           "lh-1",
				Scanner:      "lighthouse",
				RuleId:       "image-alt",
				Severity:     report.IssueSeverityCritical,
				Title:        "Image alt",
				Description:  "Missing alt",
				PageId:       "page-1",
				PageUrl:      "https://example.com/",
				ElementCount: 1,
				Occurrences: []report.IssueOccurrence{
					{Selector: stringPtr("img.hero")},
				},
			},
		},
		Scanners: []report.ScannerSummary{
			{
				Id:          "lighthouse",
				Status:      report.ScannerStatusSuccess,
				IssueCount:  intPtr(1),
				StartedAt:   &now,
				CompletedAt: &now,
				DurationMs:  floatPtr(200),
			},
		},
	}

	axeBytes, _ := json.Marshal(axeResults)
	lhBytes, _ := json.Marshal(lighthouseResults)
	mem.objects[mem.key(sharedstorage.BucketArtifacts, "job-dedup/axe/results.json")] = axeBytes
	mem.objects[mem.key(sharedstorage.BucketArtifacts, "job-dedup/lighthouse/results.json")] = lhBytes

	job := &models.Job{
		ID:               "job-dedup",
		ExpectedScanners: []string{"axe", "lighthouse"},
		ScannerResults: map[string]*models.ScannerResult{
			"axe":        {ScannerType: "axe", ResultsPath: "job-dedup/axe/results.json", Success: true},
			"lighthouse": {ScannerType: "lighthouse", ResultsPath: "job-dedup/lighthouse/results.json", Success: true},
		},
	}

	reportKey, err := aggregator.BuildAggregatedReport(ctx, job)
	if err != nil {
		t.Fatalf("buildAggregatedReport: %v", err)
	}

	reportBytes := mem.objects[mem.key(sharedstorage.BucketArtifacts, reportKey)]

	var aggReport report.UnifiedReportV2
	if unmarshalErr := json.Unmarshal(reportBytes, &aggReport); unmarshalErr != nil {
		t.Fatalf("unmarshal report: %v", unmarshalErr)
	}

	// After dedup: one issue remains (axe wins with higher priority).
	if aggReport.Summary.TotalIssues != 1 {
		t.Fatalf("expected 1 deduplicated issue, got %d", aggReport.Summary.TotalIssues)
	}

	if len(aggReport.Issues) != 1 {
		t.Fatalf("expected 1 issue in array, got %d", len(aggReport.Issues))
	}

	if aggReport.Issues[0].Scanner != "axe" {
		t.Fatalf("expected axe issue to win dedup, got scanner=%s", aggReport.Issues[0].Scanner)
	}

	// byScanner must reflect deduplicated counts: only axe=1, no lighthouse entry.
	byScannerSum := 0
	for _, count := range aggReport.Summary.ByScanner {
		byScannerSum += count
	}

	if byScannerSum != aggReport.Summary.TotalIssues {
		t.Fatalf(
			"byScanner sum=%d != totalIssues=%d (byScanner=%v)",
			byScannerSum,
			aggReport.Summary.TotalIssues,
			aggReport.Summary.ByScanner,
		)
	}

	if aggReport.Summary.ByScanner["axe"] != 1 {
		t.Fatalf("expected byScanner[axe]=1, got %d", aggReport.Summary.ByScanner["axe"])
	}

	// Page-level counts must also reflect deduplication.
	if len(aggReport.Pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(aggReport.Pages))
	}

	if aggReport.Pages[0].IssueCount != 1 {
		t.Fatalf("expected page issueCount=1 after dedup, got %d", aggReport.Pages[0].IssueCount)
	}

	if aggReport.Pages[0].BySeverity == nil {
		t.Fatal("expected page bySeverity to be set")
	}

	if aggReport.Pages[0].BySeverity.Critical != 1 {
		t.Fatalf("expected page critical=1, got %d", aggReport.Pages[0].BySeverity.Critical)
	}

	if aggReport.Summary.PagesWithIssues != 1 {
		t.Fatalf("expected pagesWithIssues=1, got %d", aggReport.Summary.PagesWithIssues)
	}
}

func TestBuildAggregatedReportNoSuccess(t *testing.T) {
	ctx := context.Background()
	mem := newMemoryStorage()
	registry := scanners.NewRegistry("scanner-runner")

	aggregator := NewAggregator(mem, registry)

	job := &models.Job{
		ID:               "job-2",
		ExpectedScanners: []string{"axe"},
		ScannerResults: map[string]*models.ScannerResult{
			"axe": {
				ScannerType: "axe",
				ResultsPath: "job-2/axe/results.json",
				Success:     false,
				Error:       "scanner failed",
			},
		},
	}

	if _, err := aggregator.BuildAggregatedReport(ctx, job); err == nil {
		t.Fatal("expected error when no successful scanners")
	}
}
