package storage

import (
	"context"
	"sort"
	"time"

	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
	"github.com/mattboback/stageflow/libs/go/models"
)

type reportAggregation struct {
	aggregator *Aggregator
	job        *models.Job

	scannerStats         map[string]*report.ScannerSummary
	pagesByKey           map[string]*aggregatedPage
	byScanner            map[string]int
	totalSeverity        report.SeverityCounts
	lighthouseCategories []report.LighthouseCategorySummary
	errors               []report.ReportError
	artifactsByID        map[string]report.ReportArtifact
	issues               []report.IssueDetail
	hadSuccess           bool
	baseURL              string
	scannedAt            *time.Time
	completedAt          *time.Time
}

func newReportAggregation(aggregator *Aggregator, job *models.Job) *reportAggregation {
	return &reportAggregation{
		aggregator:    aggregator,
		job:           job,
		scannerStats:  make(map[string]*report.ScannerSummary),
		pagesByKey:    make(map[string]*aggregatedPage),
		byScanner:     make(map[string]int),
		artifactsByID: make(map[string]report.ReportArtifact),
		issues:        make([]report.IssueDetail, 0),
	}
}

func (a *reportAggregation) initExpectedScanners() {
	for _, expected := range a.job.ExpectedScanners {
		a.scannerStat(expected)
	}
}

func (a *reportAggregation) finalizeExpectedScanners() {
	for _, expected := range a.job.ExpectedScanners {
		stat := a.scannerStat(expected)
		if stat.Status == "" {
			stat.Status = report.ScannerStatusSkipped
		}
	}
}

func (a *reportAggregation) scannerStat(id string) *report.ScannerSummary {
	if id == "" {
		id = "unknown"
	}

	if stat, ok := a.scannerStats[id]; ok {
		return stat
	}

	stat := &report.ScannerSummary{Id: id}
	if a.aggregator.scannerRegistry != nil {
		if def, ok := a.aggregator.scannerRegistry.Resolve(id); ok {
			stat.Name = stringPtr(def.Name)
		}
	}

	a.scannerStats[id] = stat

	return stat
}

func (a *reportAggregation) aggregateScanner(ctx context.Context, scannerID string, result *models.ScannerResult) {
	stat := a.scannerStat(scannerID)

	if result == nil {
		a.recordScannerFailure(scannerID, stat, "missing_result", "missing scanner result", false)

		return
	}

	if !result.Success {
		a.recordScannerFailure(scannerID, stat, string(report.ScannerStatusFailed), result.Error, false)

		return
	}

	if result.ResultsPath == "" {
		a.recordScannerFailure(scannerID, stat, "missing_results_path", "missing results path", false)

		return
	}

	doc, err := a.aggregator.downloadScanResults(ctx, result.ResultsPath)
	if err != nil {
		a.recordScannerFailure(scannerID, stat, "download_failed", err.Error(), true)

		return
	}

	stat.Status = report.ScannerStatusSuccess
	stat.ResultsPath = stringPtr(result.ResultsPath)
	stat.ReportPath = stringPtr(result.ReportPath)
	a.hadSuccess = true

	a.absorbMeta(doc.Meta)
	a.absorbScannerMetadata(scannerID, stat, doc)
	a.absorbLighthouseCategories(doc.Summary.LighthouseCategories)

	scannerSeverity := report.SeverityCounts{}
	scannerIssues := a.absorbIssues(scannerID, doc.Issues, &scannerSeverity)
	a.absorbPages(scannerID, doc.Pages)

	stat.IssueCount = intPtr(scannerIssues)
	stat.Severity = &scannerSeverity
	a.byScanner[scannerID] = scannerIssues
	addSeverityTotals(&a.totalSeverity, scannerSeverity)

	a.absorbArtifacts(doc.Artifacts)
	a.absorbErrors(scannerID, doc.Errors)
}

func (a *reportAggregation) recordScannerFailure(
	scannerID string,
	stat *report.ScannerSummary,
	code, message string,
	retryable bool,
) {
	stat.Status = report.ScannerStatusFailed
	stat.Error = stringPtr(message)
	a.errors = append(a.errors, report.ReportError{
		Scope:     report.ErrorScopeScanner,
		ScannerId: stringPtr(scannerID),
		Code:      code,
		Message:   message,
		Retryable: retryable,
	})
}

func (a *reportAggregation) absorbMeta(meta report.ReportMeta) {
	if a.baseURL == "" && meta.BaseUrl != nil {
		a.baseURL = *meta.BaseUrl
	}

	a.scannedAt = pickEarlierTime(a.scannedAt, meta.ScannedAt)
	a.completedAt = pickLaterTime(a.completedAt, meta.CompletedAt)
}

func (a *reportAggregation) absorbScannerMetadata(
	scannerID string,
	stat *report.ScannerSummary,
	doc *report.UnifiedReportV2,
) {
	if len(doc.Scanners) == 0 {
		return
	}

	for _, s := range doc.Scanners {
		if s.Id != scannerID {
			continue
		}

		stat.ToolVersion = s.ToolVersion
		stat.StartedAt = s.StartedAt
		stat.CompletedAt = s.CompletedAt
		stat.DurationMs = s.DurationMs
		if s.Status != "" {
			stat.Status = s.Status
		}

		return
	}
}

func (a *reportAggregation) absorbLighthouseCategories(categories []report.LighthouseCategorySummary) {
	if len(a.lighthouseCategories) > 0 {
		return
	}

	if len(categories) == 0 {
		return
	}

	a.lighthouseCategories = append([]report.LighthouseCategorySummary{}, categories...)
}

func (a *reportAggregation) absorbIssues(
	scannerID string,
	issues []report.IssueDetail,
	severity *report.SeverityCounts,
) int {
	scannerIssues := 0

	for _, issue := range issues {
		if issue.Scanner == "" {
			issue.Scanner = scannerID
		}

		a.issues = append(a.issues, issue)
		addSeverity(severity, issue.Severity)

		scannerIssues++
	}

	return scannerIssues
}

func (a *reportAggregation) absorbPages(scannerID string, pages []report.PageSummary) {
	for idx, page := range pages {
		mergeAggregatedPage(a.pagesByKey, scannerID, idx, page)
	}
}

func (a *reportAggregation) absorbArtifacts(artifacts []report.ReportArtifact) {
	for _, artifact := range artifacts {
		if artifact.Id == "" {
			continue
		}

		if _, exists := a.artifactsByID[artifact.Id]; exists {
			continue
		}

		a.artifactsByID[artifact.Id] = artifact
	}
}

func (a *reportAggregation) absorbErrors(scannerID string, errors []report.ReportError) {
	for _, reportError := range errors {
		if reportError.ScannerId == nil || *reportError.ScannerId == "" {
			reportError.ScannerId = stringPtr(scannerID)
		}

		a.errors = append(a.errors, reportError)
	}
}

func (a *reportAggregation) buildReport() report.UnifiedReportV2 {
	pages := a.sortedPages()

	// Deduplicate issues that are flagged by multiple scanners.
	deduplicatedIssues := deduplicateIssues(a.issues)

	// Recalculate severity counts after deduplication.
	dedupedSeverity := report.SeverityCounts{}
	for _, issue := range deduplicatedIssues {
		addSeverity(&dedupedSeverity, issue.Severity)
	}

	totalIssues := len(deduplicatedIssues)

	// Recompute byScanner from deduplicated issues so it sums to totalIssues.
	dedupedByScanner := make(map[string]int, len(a.byScanner))
	for _, issue := range deduplicatedIssues {
		dedupedByScanner[issue.Scanner]++
	}

	// Recompute page-level counts from deduplicated issues so they stay
	// consistent with the summary totals.
	recomputePageCounts(pages, deduplicatedIssues)

	pagesWithIssues := 0

	for _, page := range pages {
		if page.IssueCount > 0 {
			pagesWithIssues++
		}
	}

	summary := report.ReportSummary{
		TotalIssues:          totalIssues,
		BySeverity:           dedupedSeverity,
		ByScanner:            dedupedByScanner,
		PagesScanned:         len(pages),
		PagesWithIssues:      pagesWithIssues,
		LighthouseCategories: a.lighthouseCategories,
	}
	if totalIssues > 0 {
		score, grade := calculateAccessibilityScore(dedupedSeverity)
		summary.Score = intPtr(score)
		summary.ScoreGrade = stringPtr(grade)
	}

	var duration *float64
	if a.scannedAt != nil && a.completedAt != nil {
		duration = floatPtr(durationMs(a.scannedAt, a.completedAt))
	}

	return report.UnifiedReportV2{
		Version: reportVersion,
		Meta: report.ReportMeta{
			JobId:       a.job.ID,
			BaseUrl:     stringPtr(a.baseURL),
			ScannedAt:   a.scannedAt,
			CompletedAt: a.completedAt,
			DurationMs:  duration,
		},
		Summary:   summary,
		Scanners:  flattenScannerStats(a.scannerStats),
		Pages:     pages,
		Issues:    deduplicatedIssues,
		Artifacts: flattenArtifacts(a.artifactsByID),
		Errors:    a.errors,
	}
}

func (a *reportAggregation) sortedPages() []report.PageSummary {
	pages := make([]report.PageSummary, 0, len(a.pagesByKey))
	for _, agg := range a.pagesByKey {
		page := agg.page
		if stringValue(page.Path) == "" {
			page.Path = stringPtr(derivePath(page.Url))
		}

		pages = append(pages, page)
	}

	sort.Slice(pages, func(i, j int) bool {
		if pages[i].Url != pages[j].Url {
			return pages[i].Url < pages[j].Url
		}

		return pages[i].Id < pages[j].Id
	})

	return pages
}
