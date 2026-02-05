package models_test

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"
	"time"

	report "github.com/mattboback/stageflow/packages/contracts/report/generated/go"
)

func TestUnifiedReport_JSONShape_AndStrictRoundTrip(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	html := "<button>Click me</button>"
	failureSummary := "Element has insufficient color contrast"

	results := report.UnifiedReportV2{
		Version: "2.0.0",
		Meta: report.ReportMeta{
			JobId:       "job-123",
			BaseUrl:     strPtr("http://localhost:8080"),
			ScannedAt:   &now,
			CompletedAt: &now,
			DurationMs:  floatPtr(1234.5),
		},
		Summary: report.ReportSummary{
			TotalIssues: 3,
			BySeverity: report.SeverityCounts{
				Critical: 0,
				Serious:  2,
				Moderate: 1,
				Minor:    0,
			},
			ByScanner: map[string]int{
				"axe":      2,
				"keyboard": 1,
			},
			PagesScanned:    2,
			PagesWithIssues: 1,
		},
		Scanners: []report.ScannerSummary{
			{
				Id:          "axe",
				Name:        strPtr("Axe"),
				Status:      report.ScannerStatusSuccess,
				IssueCount:  intPtr(2),
				Severity:    &report.SeverityCounts{Critical: 0, Serious: 2, Moderate: 0, Minor: 0},
				StartedAt:   &now,
				CompletedAt: &now,
				DurationMs:  floatPtr(1000),
			},
		},
		Pages: []report.PageSummary{
			{
				Id:         "page-001",
				Url:        "http://localhost:8080/index.html",
				Path:       strPtr("/index.html"),
				IssueCount: 3,
				BySeverity: &report.SeverityCounts{Critical: 0, Serious: 2, Moderate: 1, Minor: 0},
				DurationMs: 1234.5,
				StartedAt:  &now,
				FinishedAt: &now,
			},
		},
		Issues: []report.IssueDetail{
			{
				Id:           "issue-001",
				Scanner:      "axe",
				RuleId:       "color-contrast",
				Severity:     report.IssueSeveritySerious,
				Title:        "Elements must have sufficient color contrast",
				Description:  "Some elements do not have sufficient color contrast",
				HelpUrl:      strPtr("https://dequeuniversity.com/rules/axe/4.0/color-contrast"),
				PageId:       "page-001",
				PageUrl:      "http://localhost:8080/index.html",
				ElementCount: 1,
				Occurrences: []report.IssueOccurrence{
					{
						Target:         []string{"button.primary"},
						Html:           &html,
						FailureSummary: &failureSummary,
					},
				},
			},
		},
	}

	b, err := json.Marshal(results)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Verify key JSON field names (camelCase in report contract).
	var top map[string]any
	if unmarshalErr := json.Unmarshal(b, &top); unmarshalErr != nil {
		t.Fatalf("unmarshal: %v", unmarshalErr)
	}

	metaAny, metaExists := top["meta"]
	if !metaExists {
		t.Fatalf("expected meta in JSON: %s", string(b))
	}

	meta, metaOK := metaAny.(map[string]any)
	if !metaOK {
		t.Fatalf("expected meta to be object")
	}

	if _, jobIDExists := meta["jobId"]; !jobIDExists {
		t.Fatalf("expected meta.jobId key (not snake_case): %s", string(b))
	}

	// Strict decode the roundtrip JSON (ensures we didn't introduce unknown fields).
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()

	var decoded report.UnifiedReportV2
	if decodeErr := dec.Decode(&decoded); decodeErr != nil {
		t.Fatalf("strict unmarshal: %v", decodeErr)
	}

	if trailingErr := dec.Decode(&struct{}{}); trailingErr != io.EOF {
		t.Fatalf("expected EOF after single JSON value, got %v", trailingErr)
	}

	if decoded.Meta.JobId != "job-123" || decoded.Summary.TotalIssues != 3 {
		t.Fatalf("Expected roundtrip to preserve key fields")
	}

	if len(decoded.Issues) != 1 || len(decoded.Issues[0].Occurrences) != 1 {
		t.Fatalf("Expected one issue with one occurrence")
	}

	if decoded.Issues[0].Occurrences[0].Target[0] != "button.primary" {
		t.Fatalf("Expected occurrence targets to be preserved")
	}
}

func strPtr(value string) *string {
	if value == "" {
		return nil
	}

	return &value
}

func intPtr(value int) *int {
	v := value

	return &v
}

func floatPtr(value float64) *float64 {
	v := value

	return &v
}
