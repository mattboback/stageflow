package diff

import (
	"slices"
	"testing"

	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
)

func productCoverageReport() report.UnifiedReportV2 {
	return report.UnifiedReportV2{
		Meta: report.ReportMeta{ScanContext: &report.ScanContext{
			ExpectedPages: []string{"page-v2-a", "page-v2-b"}, IdentityScope: report.ScanContextIdentityScopePath,
			InputType: report.ScanContextInputTypeUrls, Purpose: report.ScanContextPurposeFull,
			ScannerConfigFingerprints: map[string]string{"axe": "axe-settings"},
			PageConfigFingerprints:    map[string]string{"page-v2-a": "page-settings", "page-v2-b": "page-settings"},
		}},
		Pages: []report.PageSummary{{Id: "page-v2-a"}, {Id: "page-v2-b"}},
		Issues: []report.IssueDetail{
			{Id: "a", Scanner: "axe", PageId: "page-v2-a"},
			{Id: "b", Scanner: "axe", PageId: "page-v2-b"},
		},
		Scanners: []report.ScannerSummary{
			{
				Id:          "axe",
				Status:      report.ScannerStatusSuccess,
				ToolVersion: strPtr("1"),
				Pages: []report.ScannerPageCoverage{
					{
						PageId:   "page-v2-a",
						Status:   report.ScannerPageCoverageStatusSuccess,
						FinalUrl: strPtr("https://production.example/a"),
					},
					{
						PageId:   "page-v2-b",
						Status:   report.ScannerPageCoverageStatusSuccess,
						FinalUrl: strPtr("https://production.example/b"),
					},
				},
			},
		},
		Summary: report.ReportSummary{Score: intPtr(70)},
	}
}

func TestProductDiffRelocationMatchesIdentityScope(t *testing.T) {
	for _, tc := range []struct {
		name          string
		input         report.ScanContextInputType
		scope         report.ScanContextIdentityScope
		before, after string
		fixed         int
	}{
		{"zip ephemeral port", report.ScanContextInputTypeZip, report.ScanContextIdentityScopePath, "http://127.0.0.1:30001/a", "http://127.0.0.1:30099/a", 2},
		{"production to staging", report.ScanContextInputTypeUrls, report.ScanContextIdentityScopePath, "https://production.example/a", "https://staging.example/a", 2},
		{"origin-scoped relocation", report.ScanContextInputTypeUrls, report.ScanContextIdentityScopeOrigin, "https://production.example/a", "https://staging.example/a", 1},
		{"changed destination path", report.ScanContextInputTypeUrls, report.ScanContextIdentityScopePath, "https://production.example/a", "https://production.example/other", 1},
		{"changed destination query", report.ScanContextInputTypeUrls, report.ScanContextIdentityScopePath, "https://production.example/a?q=1", "https://production.example/a?q=2", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before, after := productCoverageReport(), productCoverageReport()
			before.Meta.ScanContext.InputType, after.Meta.ScanContext.InputType = tc.input, tc.input
			before.Meta.ScanContext.IdentityScope, after.Meta.ScanContext.IdentityScope = tc.scope, tc.scope
			before.Scanners[0].Pages[0].FinalUrl, after.Scanners[0].Pages[0].FinalUrl = strPtr(
				tc.before,
			), strPtr(
				tc.after,
			)
			after.Issues = nil

			result := ComputeDiff("before", before, "after", after)
			if result.Delta.FixedIssues != tc.fixed {
				t.Fatalf("fixed=%d, want %d: %+v", result.Delta.FixedIssues, tc.fixed, result)
			}

			if tc.fixed == 1 &&
				(result.Delta.ScoreDelta != nil || !slices.Contains(result.NotComparedPages, "page-v2-a")) {
				t.Fatalf("unverified destination compared: %+v", result)
			}
		})
	}
}

func TestProductDiffKeepsCommonSuccessfulPartialCoverage(t *testing.T) {
	before, after := productCoverageReport(), productCoverageReport()
	after.Issues = nil
	after.Scanners[0].Status = report.ScannerStatusPartial
	after.Scanners[0].Pages[1].Status = report.ScannerPageCoverageStatusFailed

	result := ComputeDiff("before", before, "after", after)
	if result.Delta.FixedIssues != 1 || result.Fixed[0].Id != "a" ||
		!slices.Contains(result.NotComparedPages, "page-v2-b") ||
		result.Delta.ScoreDelta != nil {
		t.Fatalf("successful common coverage must compare, failed page must not: %+v", result)
	}
}

func TestProductDiffOmittedExpectedPageIsNotFixed(t *testing.T) {
	before, after := productCoverageReport(), productCoverageReport()
	after.Pages = after.Pages[:1]
	after.Scanners[0].Pages = after.Scanners[0].Pages[:1]
	after.Issues = nil

	result := ComputeDiff("before", before, "after", after)
	if result.Delta.FixedIssues != 1 || result.Fixed[0].Id != "a" ||
		!slices.Contains(result.NotComparedPages, "page-v2-b") ||
		result.Delta.ScoreDelta != nil {
		t.Fatalf("omitted expected page compared: %+v", result)
	}
}

func TestProductDiffProfileTogglePreservesRawIssuesWithCoverage(t *testing.T) {
	before, after := productCoverageReport(), productCoverageReport()
	staging := report.ScanProfileStaging
	after.Meta.Profile = &staging

	result := ComputeDiff("before", before, "after", after)
	if result.Delta.FixedIssues != 0 || result.Delta.NewIssues != 0 || result.Delta.UnchangedIssues != 2 ||
		result.Delta.ScoreDelta != nil {
		t.Fatalf("profile toggle changed raw findings: %+v", result)
	}
}

func TestProductDiffRuleErrorsNeverBecomeFixes(t *testing.T) {
	before, after := productCoverageReport(), productCoverageReport()
	before.Issues[0].RuleId = "image-alt"
	after.Issues = nil
	before.Scanners[0].Pages[0].Rules = []report.RuleEvaluation{
		{RuleId: "image-alt", Outcome: report.RuleEvaluationOutcomeFail},
	}
	after.Scanners[0].Pages[0].Rules = []report.RuleEvaluation{
		{RuleId: "image-alt", Outcome: report.RuleEvaluationOutcomeError},
	}

	result := ComputeDiff("before", before, "after", after)
	if result.Delta.FixedIssues != 1 || result.Fixed[0].Id != "b" || result.Delta.ScoreDelta != nil ||
		!slices.Contains(result.NotComparedPages, "page-v2-a") {
		t.Fatalf("failed rule became a fix: %+v", result)
	}
}
