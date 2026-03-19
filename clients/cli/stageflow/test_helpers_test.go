package main

import (
	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
)

func stubEnv(string) string {
	return ""
}

func sampleReport(jobID string) report.UnifiedReportV2 {
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
				Name:       stringPtr("Axe"),
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
				HelpUrl:      stringPtr("https://example.com/help/color-contrast"),
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

func stringPtr(value string) *string {
	return &value
}
