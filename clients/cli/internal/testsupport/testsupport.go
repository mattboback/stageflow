// Package testsupport provides shared fixtures and assertion helpers for CLI tests.
package testsupport

import (
	"reflect"
	"testing"

	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
)

// StubEnv is a getenv stand-in that resolves every variable to empty.
func StubEnv(string) string {
	return ""
}

// SampleReport returns a small two-issue unified report for the given job ID.
func SampleReport(jobID string) report.UnifiedReportV2 {
	baseURL := "https://example.com"
	duration := 47000.0
	score := 72
	scoreGrade := "C"
	infoCount := 0
	criticalIssues := 1

	return report.UnifiedReportV2{
		Version: "2.0.0",
		Meta: report.ReportMeta{
			JobId:      jobID,
			BaseUrl:    &baseURL,
			DurationMs: &duration,
		},
		Summary: report.ReportSummary{
			ByScanner: map[string]int{
				"axe": 2,
			},
			BySeverity: report.SeverityCounts{
				Critical: 1,
				Serious:  0,
				Moderate: 0,
				Minor:    1,
				Info:     &infoCount,
			},
			PagesScanned:    1,
			PagesWithIssues: 1,
			Score:           &score,
			ScoreGrade:      &scoreGrade,
			TotalIssues:     2,
		},
		Scanners: []report.ScannerSummary{
			{
				Id:         "axe",
				Name:       StringPtr("Axe"),
				Status:     report.ScannerStatusSuccess,
				IssueCount: &criticalIssues,
			},
		},
		Pages: []report.PageSummary{
			{
				Id:         "page-1",
				Url:        "https://example.com",
				IssueCount: 2,
				DurationMs: 1000,
			},
		},
		Issues: []report.IssueDetail{
			{
				Id:           "issue-critical",
				Scanner:      "axe",
				RuleId:       "color-contrast",
				Severity:     report.IssueSeverityCritical,
				Title:        "Critical issue",
				Description:  "Text contrast is too low.",
				HelpUrl:      StringPtr("https://example.com/help/color-contrast"),
				PageId:       "page-1",
				PageUrl:      "https://example.com",
				ElementCount: 1,
			},
			{
				Id:           "issue-minor",
				Scanner:      "axe",
				RuleId:       "landmark-one-main",
				Severity:     report.IssueSeverityMinor,
				Title:        "Minor issue",
				Description:  "Document should contain a main landmark.",
				PageId:       "page-1",
				PageUrl:      "https://example.com",
				ElementCount: 1,
			},
		},
	}
}

func StringPtr(value string) *string {
	return &value
}

func RequireNoErr(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func RequireEqual[T comparable](t *testing.T, got, want T, label string) {
	t.Helper()

	if got != want {
		t.Fatalf("%s = %v, want %v", label, got, want)
	}
}

func RequireDeepEqual(t *testing.T, got, want any, label string) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", label, got, want)
	}
}

func RequireBoolPtr(t *testing.T, got *bool, want bool, label string) {
	t.Helper()

	if got == nil {
		t.Fatalf("%s = nil, want %v", label, want)
	}

	if *got != want {
		t.Fatalf("%s = %v, want %v", label, *got, want)
	}
}
