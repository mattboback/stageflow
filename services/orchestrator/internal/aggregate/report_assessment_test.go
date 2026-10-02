package aggregate

import (
	"testing"

	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
)

func TestStagingAssessmentRetainsRawObservationsAndPendingReviews(t *testing.T) {
	issues := []report.IssueDetail{}
	for range 22 {
		issues = append(
			issues,
			report.IssueDetail{
				Id:       "noindex",
				Scanner:  "lighthouse",
				RuleId:   "is-crawlable",
				Severity: report.IssueSeveritySerious,
				ScannerData: map[string]interface{}{
					"details": map[string]interface{}{"source": "x-robots-tag: noindex, nofollow"},
				},
			},
		)
	}

	for range 14 {
		issues = append(
			issues,
			report.IssueDetail{
				Id:          "contrast",
				Scanner:     "axe",
				Severity:    report.IssueSeverityModerate,
				ScannerData: map[string]interface{}{"axeIncomplete": true},
			},
		)
	}

	summary := report.ReportSummary{Score: intPtr(82)}

	staging := buildAssessment("staging", issues, summary, 22)
	if staging.Expected != 22 || staging.NeedsReview != 14 || staging.Actionable != 0 || staging.Score == nil ||
		*staging.Score != 100 ||
		*summary.Score != 82 ||
		len(issues) != 36 {
		t.Fatalf("unexpected assessment: %+v", staging)
	}

	production := buildAssessment("production", issues, summary, 22)
	if production.Expected != 0 || production.Actionable != 22 || production.NeedsReview != 14 {
		t.Fatalf("unexpected production assessment: %+v", production)
	}

	issues[0].ScannerData = map[string]interface{}{
		"details": map[string]interface{}{"source": "robots.txt: Disallow /"},
	}

	staging = buildAssessment("staging", issues, summary, 22)
	if staging.Actionable != 1 || staging.Expected != 21 {
		t.Fatalf("other crawl failures were suppressed: %+v", staging)
	}
}
