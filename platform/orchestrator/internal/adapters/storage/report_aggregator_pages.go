package storage

import report "github.com/mattboback/stageflow/packages/contracts/report/generated/go"

const scannerIDAxe = "axe"

type aggregatedPage struct {
	page            report.PageSummary
	overviewScanner string
}

// recomputePageCounts rebuilds each page's IssueCount and BySeverity from the
// deduplicated issue list. This ensures page-level counts stay consistent with
// the report-level summary after cross-scanner deduplication.
func recomputePageCounts(pages []report.PageSummary, issues []report.IssueDetail) {
	// Build per-page severity counts from the deduplicated issues.
	type pageCounts struct {
		issueCount int
		severity   report.SeverityCounts
	}

	byPage := make(map[string]*pageCounts, len(pages))
	for i := range pages {
		byPage[pages[i].Id] = &pageCounts{}
	}

	for _, issue := range issues {
		pc, ok := byPage[issue.PageId]
		if !ok {
			continue
		}

		pc.issueCount++
		addSeverity(&pc.severity, issue.Severity)
	}

	for i := range pages {
		pc, ok := byPage[pages[i].Id]
		if !ok {
			continue
		}

		pages[i].IssueCount = pc.issueCount

		if pc.issueCount > 0 {
			pages[i].BySeverity = &pc.severity
		} else {
			pages[i].BySeverity = nil
		}
	}
}

func mergeAggregatedPage(
	pagesByKey map[string]*aggregatedPage,
	scannerID string,
	index int,
	page report.PageSummary,
) {
	key := pageKey(scannerID, index, page)

	agg, ok := pagesByKey[key]
	if !ok {
		agg = &aggregatedPage{page: report.PageSummary{
			Id:         page.Id,
			Url:        page.Url,
			Path:       page.Path,
			IssueCount: 0,
			DurationMs: 0,
		}}
		if agg.page.Id == "" {
			agg.page.Id = derivePageID(page)
		}

		agg.page.StartedAt = page.StartedAt
		agg.page.FinishedAt = page.FinishedAt
		pagesByKey[key] = agg
	}

	agg.page.Url = firstNonEmpty(agg.page.Url, page.Url)
	agg.page.Path = stringPtr(firstNonEmpty(stringValue(agg.page.Path), stringValue(page.Path)))

	agg.page.IssueCount += page.IssueCount
	agg.page.BySeverity = addSeverityCounts(agg.page.BySeverity, page.BySeverity)

	agg.page.DurationMs += page.DurationMs
	agg.page.StartedAt = pickEarlierTime(agg.page.StartedAt, page.StartedAt)
	agg.page.FinishedAt = pickLaterTime(agg.page.FinishedAt, page.FinishedAt)

	if page.PageOverview != nil {
		if agg.page.PageOverview == nil || scannerID == scannerIDAxe && agg.overviewScanner != scannerIDAxe {
			agg.page.PageOverview = page.PageOverview
			agg.overviewScanner = scannerID
		}
	}
}
