package aggregate

import (
	"testing"

	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
)

func TestStagingNoindexAssessmentRequiresDirectiveEvidence(t *testing.T) {
	issue := report.IssueDetail{
		Id:       "crawl",
		Scanner:  "lighthouse",
		RuleId:   "is-crawlable",
		Severity: report.IssueSeveritySerious,
		ScannerData: map[string]interface{}{
			"details": map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{
						"source": "robots.txt: Disallow /",
						"url":    "https://example.com/noindex-guide",
					},
				},
			},
		},
	}

	assessment := buildAssessment("staging", []report.IssueDetail{issue}, report.ReportSummary{Score: intPtr(90)}, 1)
	if assessment.Expected != 0 || assessment.Actionable != 1 {
		t.Fatalf("a URL containing noindex is not a confirmed directive: %+v", assessment)
	}
}
