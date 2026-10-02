package storage

import (
	"context"
	"net/url"
	"sort"
	"strings"
	"time"

	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
	"github.com/mattboback/stageflow/libs/go/diff"
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
	manualChecks         map[string]*report.ManualCheck
	hadSuccess           bool
	baseURL              string
	scannedAt            *time.Time
	completedAt          *time.Time
	scanContext          *report.ScanContext
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
		manualChecks:  make(map[string]*report.ManualCheck),
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
	a.hadSuccess = true

	a.absorbMeta(doc.Meta)
	a.absorbScanContext(doc.Meta.ScanContext)
	a.absorbScannerMetadata(scannerID, stat, doc)

	if a.job.ScannerStates[scannerID] == "partial" {
		stat.Status = report.ScannerStatusPartial
		stat.Error = stringPtr(result.Error)
	}

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
		stat.Pages = s.Pages
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

		if isManualCheck(issue) {
			a.absorbManualCheck(issue)
			continue
		}

		a.issues = append(a.issues, issue)
		addSeverity(severity, issue.Severity)

		scannerIssues++
	}

	return scannerIssues
}

// manualCheckDescriptionPrefix is how the Lighthouse scanner labels a manual
// audit; the checklist already says so, so the prefix is dropped.
const manualCheckDescriptionPrefix = "Manual verification required: "

// isManualCheck reports whether a scanner issue is a Lighthouse manual audit: a
// fixed checklist item emitted for every page of every site, not a finding.
func isManualCheck(issue report.IssueDetail) bool {
	manual, ok := issue.ScannerData["lighthouseManual"].(bool)

	return ok && manual
}

// absorbManualCheck lists a manual audit once per rule instead of once per
// page, keeping it out of issue totals, severity counts, and baselines.
func (a *reportAggregation) absorbManualCheck(issue report.IssueDetail) {
	key := issue.Scanner + "|" + issue.RuleId
	if existing, ok := a.manualChecks[key]; ok {
		existing.PageCount++
		if issue.PageId != "" {
			existing.PageIds = append(existing.PageIds, issue.PageId)
		}

		return
	}

	check := &report.ManualCheck{
		Scanner:   issue.Scanner,
		RuleId:    issue.RuleId,
		Title:     issue.Title,
		HelpUrl:   issue.HelpUrl,
		PageCount: 1,
		PageIds:   []string{issue.PageId},
	}

	if description := strings.TrimPrefix(issue.Description, manualCheckDescriptionPrefix); description != "" {
		check.Description = stringPtr(description)
	}

	a.manualChecks[key] = check
}

func (a *reportAggregation) sortedManualChecks() []report.ManualCheck {
	checks := make([]report.ManualCheck, 0, len(a.manualChecks))
	for _, check := range a.manualChecks {
		sort.Strings(check.PageIds)
		checks = append(checks, *check)
	}

	sort.Slice(checks, func(i, j int) bool {
		if checks[i].Scanner != checks[j].Scanner {
			return checks[i].Scanner < checks[j].Scanner
		}

		return checks[i].RuleId < checks[j].RuleId
	})

	return checks
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
	deduplicatedIssues := a.coverageFilteredIssues(pages)

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

	a.recomputeScannerCounts(deduplicatedIssues, dedupedByScanner)

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
	scanners := flattenScannerStats(a.scannerStats)

	// A score from a partial run would describe only the scanners that
	// finished, so a crashed accessibility scanner would raise it. Leave it
	// out and let the scanner list say what is missing.
	if a.job.Config.Verification == nil && len(scanners) > 0 && len(diff.IncompleteScanners(scanners)) == 0 {
		scoredSeverity := report.SeverityCounts{}

		for _, issue := range deduplicatedIssues {
			if !needsHumanReview(issue) {
				addSeverity(&scoredSeverity, issue.Severity)
			}
		}

		score, grade := calculateAccessibilityScore(scoredSeverity, len(pages))
		summary.Score = intPtr(score)
		summary.ScoreGrade = stringPtr(grade)
	}

	var duration *float64
	if a.scannedAt != nil && a.completedAt != nil {
		duration = floatPtr(durationMs(a.scannedAt, a.completedAt))
	}

	scanContext := a.scanContext
	if scanContext == nil && a.job.Config.Verification != nil {
		identityScope := report.ScanContextIdentityScopePath
		if a.job.Config.IdentityScope == "origin" {
			identityScope = report.ScanContextIdentityScopeOrigin
		}

		// A scanner may fail before reading provenance. Keep this run explicitly
		// targeted without inventing successful coverage or execution fingerprints.
		scanContext = &report.ScanContext{
			ExpectedPages:             []string{a.job.Config.Verification.PageID},
			Purpose:                   report.ScanContextPurposeVerification,
			IdentityScope:             identityScope,
			InputType:                 report.ScanContextInputType(a.job.InputType),
			Authenticated:             len(a.job.Config.Auth) > 0,
			ScannerConfigFingerprints: map[string]string{},
		}
	}

	return report.UnifiedReportV2{
		Version: reportVersion,
		Meta: report.ReportMeta{
			JobId:         a.job.ID,
			Profile:       profilePointer(a.job.Config.Profile),
			PolicyVersion: stringPtr("indexing-v1"),
			ScanContext:   scanContext,
			BaseUrl:       stringPtr(a.baseURL),
			ScannedAt:     a.scannedAt,
			CompletedAt:   a.completedAt,
			DurationMs:    duration,
		},
		Summary:   summary,
		Scanners:  scanners,
		Pages:     pages,
		Issues:    deduplicatedIssues,
		Artifacts: flattenArtifacts(a.artifactsByID),
		Errors:    a.errors,

		ManualChecks: a.sortedManualChecks(),
		Assessment:   buildAssessment(a.job.Config.Profile, deduplicatedIssues, summary, len(pages)),
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

func (a *reportAggregation) recomputeScannerCounts(issues []report.IssueDetail, counts map[string]int) {
	severities := make(map[string]report.SeverityCounts)
	for _, issue := range issues {
		severity := severities[issue.Scanner]
		addSeverity(&severity, issue.Severity)
		severities[issue.Scanner] = severity
	}

	for scannerID, stat := range a.scannerStats {
		if stat.Status != report.ScannerStatusSuccess && stat.Status != report.ScannerStatusPartial {
			continue
		}

		severity := severities[scannerID]
		stat.IssueCount = intPtr(counts[scannerID])
		stat.Severity = &severity
	}
}

// coverageFilteredIssues assigns proven overlapping concerns to a successful
// specialist. An incomplete or unselected specialist must never hide fallback
// findings. SEO description, H1, canonical and viewport checks remain independent:
// they are not equivalent to axe's checks.
func (a *reportAggregation) coverageFilteredIssues(pages []report.PageSummary) []report.IssueDetail {
	axeSuccess := a.scannerStats["axe"] != nil && a.scannerStats["axe"].Status == report.ScannerStatusSuccess
	origins, robotsPage := robotsRepresentatives(pages, a.issues)
	seenRobots := make(map[string]bool)

	issues := make([]report.IssueDetail, 0, len(a.issues))
	for _, issue := range a.issues {
		if axeSuccess && axeReplacesIssue(issue) {
			continue
		}

		if issue.Scanner == "lighthouse" && issue.RuleId == "robots-txt" {
			if origin := origins[issue.PageId]; origin != "" {
				if issue.PageId != robotsPage[origin] || seenRobots[origin] {
					continue
				}

				seenRobots[origin] = true
			}
		}

		issues = append(issues, issue)
	}

	return issues
}

func axeReplacesIssue(issue report.IssueDetail) bool {
	switch issue.Scanner {
	case "lighthouse":
		return stringValue(issue.Category) == "accessibility" || ruleEquivalences["axe:"+issue.RuleId] != ""
	case "seo":
		switch strings.TrimPrefix(issue.RuleId, "seo-") {
		case "missing-title", "missing-language", "images-missing-alt", "heading-hierarchy":
			return true
		}
	case "link-checker":
		return strings.TrimPrefix(issue.RuleId, "link-checker-") == "no-text-links"
	}

	return false
}

// robotsRepresentatives chooses a stable page per origin so URL ordering cannot
// move an origin-wide finding to another page between runs.
func robotsRepresentatives(
	pages []report.PageSummary,
	issues []report.IssueDetail,
) (map[string]string, map[string]string) {
	origins := make(map[string]string, len(pages))

	pageURLs := make(map[string]string, len(pages))
	for _, page := range pages {
		pageURLs[page.Id] = page.Url
		if parsed, err := url.Parse(page.Url); err == nil && parsed.Host != "" {
			origins[page.Id] = parsed.Scheme + "://" + parsed.Host
		}
	}

	robotsPage := make(map[string]string)

	for _, issue := range issues {
		if issue.Scanner != "lighthouse" || issue.RuleId != "robots-txt" {
			continue
		}

		origin := origins[issue.PageId]
		if origin == "" {
			continue
		}

		if selected, ok := robotsPage[origin]; !ok || pageURLs[issue.PageId] < pageURLs[selected] {
			robotsPage[origin] = issue.PageId
		}
	}

	return origins, robotsPage
}

func (a *reportAggregation) absorbScanContext(context *report.ScanContext) {
	if context == nil {
		return
	}

	if a.scanContext == nil {
		contextCopy := *context
		contextCopy.ScannerConfigFingerprints = make(map[string]string)
		contextCopy.ScannerConfigurations = make(map[string]map[string]interface{})
		contextCopy.PageConfigFingerprints = make(map[string]string)
		contextCopy.VerificationEligiblePages = make(map[string]bool)
		a.scanContext = &contextCopy
	}

	for key, value := range context.ScannerConfigurations {
		a.scanContext.ScannerConfigurations[key] = value
	}

	for key, value := range context.VerificationEligiblePages {
		a.scanContext.VerificationEligiblePages[key] = value
	}

	for key, value := range context.PageConfigFingerprints {
		a.scanContext.PageConfigFingerprints[key] = value
	}

	for key, value := range context.ScannerConfigFingerprints {
		a.scanContext.ScannerConfigFingerprints[key] = value
	}

	for _, id := range context.ExpectedPages {
		found := false

		for _, existing := range a.scanContext.ExpectedPages {
			if existing == id {
				found = true
				break
			}
		}

		if !found {
			a.scanContext.ExpectedPages = append(a.scanContext.ExpectedPages, id)
		}
	}

	sort.Strings(a.scanContext.ExpectedPages)
}
