package orchestrator

import report "github.com/mattboback/stageflow/packages/contracts/report/generated/go"

const scannerIDAxe = "axe"

type aggregatedPage struct {
	page            report.PageSummary
	overviewScanner string
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
