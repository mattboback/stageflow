package render

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
	"github.com/mattboback/stageflow/clients/cli/internal/exitcode"
	"github.com/mattboback/stageflow/clients/cli/internal/testsupport"
	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
)

func TestRenderUnifiedReport_JSONEnvelope(t *testing.T) {
	createdAt := time.Date(2026, 3, 4, 12, 0, 0, 0, time.FixedZone("X", -5*60*60))
	updatedAt := createdAt.Add(90 * time.Second)

	status := apiclient.JobStatus{
		ID:        "job-123",
		State:     apiclient.JobStateDone,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	doc := testsupport.SampleReport(status.ID)

	var buf bytes.Buffer

	err := UnifiedReport(&buf, "http://localhost:8080", status, doc, RenderOptions{
		Format:    FormatJSON,
		MaxIssues: 1,
	})
	testsupport.RequireNoErr(t, err)

	var payload ReportEnvelope
	testsupport.RequireNoErr(t, json.NewDecoder(bytes.NewReader(buf.Bytes())).Decode(&payload))

	testsupport.RequireEqual(t, payload.Schema, "stageflow-cli/report@v1", "payload.Schema")
	testsupport.RequireEqual(t, payload.API.BaseURL, "http://localhost:8080", "payload.API.BaseURL")

	testsupport.RequireEqual(t, payload.Job.ID, status.ID, "payload.Job.ID")
	testsupport.RequireEqual(t, payload.Job.State, status.State, "payload.Job.State")
	testsupport.RequireEqual(t, payload.Job.CreatedAt, createdAt.UTC().Format(timeFormatRFC3339), "payload.Job.CreatedAt")
	testsupport.RequireEqual(t, payload.Job.UpdatedAt, updatedAt.UTC().Format(timeFormatRFC3339), "payload.Job.UpdatedAt")

	testsupport.RequireEqual(t, payload.Links.Job, "http://localhost:8080/api/v1/jobs/job-123", "payload.Links.Job")
	testsupport.RequireEqual(t, payload.Links.Results, "http://localhost:8080/api/v1/jobs/job-123/results", "payload.Links.Results")

	testsupport.RequireEqual(t, payload.Filters.Sort, issueSortOrder, "payload.Filters.Sort")
	testsupport.RequireEqual(t, payload.Filters.MaxIssues, 1, "payload.Filters.MaxIssues")
	testsupport.RequireEqual(t, payload.Filters.IssuesTotal, 2, "payload.Filters.IssuesTotal")
	testsupport.RequireEqual(t, payload.Filters.IssuesReturned, 1, "payload.Filters.IssuesReturned")
	testsupport.RequireEqual(t, payload.Filters.Truncated, true, "payload.Filters.Truncated")

	testsupport.RequireEqual(t, len(payload.Report.Issues), 1, "len(payload.Report.Issues)")
	testsupport.RequireEqual(t, payload.Report.Issues[0].Id, "issue-critical", "payload.Report.Issues[0].Id")
}

func TestRenderUnifiedReport_Markdown(t *testing.T) {
	status := apiclient.JobStatus{
		ID:        "job-123",
		State:     apiclient.JobStateDone,
		CreatedAt: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 3, 4, 12, 1, 0, 0, time.UTC),
	}

	doc := testsupport.SampleReport(status.ID)
	artifactPath := "screenshots/page-overview-page-1.webp"
	artifactMime := "image/webp"
	doc.Artifacts = []report.ReportArtifact{
		{
			Id:   "page-overview-axe-page-1",
			Type: "page-overview",
			Path: &artifactPath,
			Mime: &artifactMime,
		},
	}

	var buf bytes.Buffer

	err := UnifiedReport(&buf, "http://localhost:8080", status, doc, RenderOptions{
		Format:    FormatMarkdown,
		MaxIssues: 10,
	})
	testsupport.RequireNoErr(t, err)

	output := buf.String()
	for _, want := range []string{
		"## Scan Summary",
		"## Scanners",
		"## Pages",
		"## Findings",
		"## Artifacts",
		"## Report Errors",
		"| axe | Axe | success | 1 |",
		"| page-1 | https://example.com | - | 2 | 1000ms |",
		"| page-overview-axe-page-1 | page-overview | screenshots/page-overview-page-1.webp | image/webp |",
		"- [critical] Critical issue | scanner=axe | rule=color-contrast | page=https://example.com",
		"No report errors.",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("markdown output missing %q:\n%s", want, output)
		}
	}
}

func TestRenderUnifiedReport_MarkdownNoFindingsNoArtifacts(t *testing.T) {
	status := apiclient.JobStatus{ID: "job-456", State: apiclient.JobStateDone}
	doc := testsupport.SampleReport(status.ID)
	doc.Issues = []report.IssueDetail{}
	doc.Artifacts = nil
	doc.Errors = nil
	doc.Summary.TotalIssues = 0
	doc.Summary.PagesWithIssues = 0
	doc.Summary.BySeverity = report.SeverityCounts{}

	var buf bytes.Buffer

	err := UnifiedReport(&buf, "http://localhost:8080", status, doc, RenderOptions{
		Format:    FormatMarkdown,
		MaxIssues: 10,
	})
	testsupport.RequireNoErr(t, err)

	output := buf.String()
	if !strings.Contains(output, "No findings.") {
		t.Fatalf("expected no findings marker:\n%s", output)
	}

	if !strings.Contains(output, "No artifacts.") {
		t.Fatalf("expected no artifacts marker:\n%s", output)
	}
}

func TestRenderUnifiedReport_MarkdownOccurrences(t *testing.T) {
	status := apiclient.JobStatus{ID: "job-occ", State: apiclient.JobStateDone}
	doc := testsupport.SampleReport(status.ID)

	selector1 := "#main > p.intro"
	html1 := `<p class="intro" style="color:#999">...</p>`
	failSummary := "Fix color contrast"
	selector2 := ".footer > a"
	html2 := `<a href="/about" style="color:#aaa">About</a>`
	howToFix := "Increase text contrast ratio to at least 4.5:1"
	category := "accessibility"

	doc.Issues = []report.IssueDetail{
		{
			Id:          "issue-1",
			Scanner:     "axe",
			RuleId:      "color-contrast",
			Severity:    report.IssueSeverityCritical,
			Title:       "Color contrast insufficient",
			Description: "Text contrast is too low.",
			HowToFix:    &howToFix,
			Category:    &category,
			WcagTags:    []string{"1.4.3"},
			HelpUrl:     testsupport.StringPtr("https://example.com/help"),
			PageId:      "page-1",
			PageUrl:     "https://example.com",
			Occurrences: []report.IssueOccurrence{
				{Selector: &selector1, Html: &html1, FailureSummary: &failSummary},
				{Selector: &selector2, Html: &html2},
			},
			ElementCount: 2,
		},
	}

	var buf bytes.Buffer

	err := UnifiedReport(&buf, "http://localhost:8080", status, doc, RenderOptions{
		Format:         FormatMarkdown,
		MaxIssues:      10,
		MaxOccurrences: 3,
	})
	testsupport.RequireNoErr(t, err)

	output := buf.String()
	for _, want := range []string{
		"How to fix: Increase text contrast ratio",
		"category=accessibility",
		"WCAG: 1.4.3",
		"Help: https://example.com/help",
		"Occurrences (2 of 2):",
		"`#main > p.intro`",
		"Fix color contrast",
		"`.footer > a`",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("markdown output missing %q:\n%s", want, output)
		}
	}
}

func TestRenderUnifiedReport_TextOccurrences(t *testing.T) {
	status := apiclient.JobStatus{ID: "job-text-occ", State: apiclient.JobStateDone}
	doc := testsupport.SampleReport(status.ID)

	selector := "#main > p"
	html := `<p>hello</p>`
	howToFix := "Fix the thing"
	category := "accessibility"

	doc.Issues = []report.IssueDetail{
		{
			Id:          "issue-1",
			Scanner:     "axe",
			RuleId:      "r1",
			Severity:    report.IssueSeverityCritical,
			Title:       "Test issue",
			Description: "Desc",
			HowToFix:    &howToFix,
			Category:    &category,
			WcagTags:    []string{"2.1.1"},
			PageId:      "page-1",
			PageUrl:     "https://example.com",
			Occurrences: []report.IssueOccurrence{
				{Selector: &selector, Html: &html},
			},
			ElementCount: 1,
		},
	}

	var buf bytes.Buffer

	err := UnifiedReport(&buf, "http://localhost:8080", status, doc, RenderOptions{
		Format:         FormatText,
		MaxIssues:      10,
		MaxOccurrences: 3,
	})
	testsupport.RequireNoErr(t, err)

	output := buf.String()
	for _, want := range []string{
		"Category: accessibility",
		"How to fix: Fix the thing",
		"WCAG: 2.1.1",
		"Occurrences (1 of 1):",
		"`#main > p`",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("text output missing %q:\n%s", want, output)
		}
	}
}

func TestRenderUnifiedReport_TextOccurrences_FailureSummaryOnly(t *testing.T) {
	status := apiclient.JobStatus{ID: "job-text-summary-only", State: apiclient.JobStateDone}
	doc := testsupport.SampleReport(status.ID)

	failureSummary := "Element has no accessible name"
	doc.Issues = []report.IssueDetail{
		{
			Id:       "issue-1",
			Scanner:  "axe",
			RuleId:   "button-name",
			Severity: report.IssueSeveritySerious,
			Title:    "Button name missing",
			PageUrl:  "https://example.com",
			Occurrences: []report.IssueOccurrence{
				{FailureSummary: &failureSummary},
			},
		},
	}

	var buf bytes.Buffer

	err := UnifiedReport(&buf, "http://localhost:8080", status, doc, RenderOptions{
		Format:         FormatText,
		MaxIssues:      10,
		MaxOccurrences: 3,
	})
	testsupport.RequireNoErr(t, err)

	output := buf.String()
	if !strings.Contains(output, "Occurrences (1 of 1):") {
		t.Fatalf("expected occurrences header:\n%s", output)
	}

	if !strings.Contains(output, "1. Element has no accessible name") {
		t.Fatalf("expected failure summary occurrence detail:\n%s", output)
	}
}

func TestRenderUnifiedReport_MarkdownOccurrences_FailureSummaryOnly(t *testing.T) {
	status := apiclient.JobStatus{ID: "job-md-summary-only", State: apiclient.JobStateDone}
	doc := testsupport.SampleReport(status.ID)

	failureSummary := "Landmark region is missing a label"
	doc.Issues = []report.IssueDetail{
		{
			Id:       "issue-1",
			Scanner:  "axe",
			RuleId:   "landmark-one-main",
			Severity: report.IssueSeverityModerate,
			Title:    "Missing landmark label",
			PageUrl:  "https://example.com",
			Occurrences: []report.IssueOccurrence{
				{FailureSummary: &failureSummary},
			},
		},
	}

	var buf bytes.Buffer

	err := UnifiedReport(&buf, "http://localhost:8080", status, doc, RenderOptions{
		Format:         FormatMarkdown,
		MaxIssues:      10,
		MaxOccurrences: 3,
	})
	testsupport.RequireNoErr(t, err)

	output := buf.String()
	if !strings.Contains(output, "Occurrences (1 of 1):") {
		t.Fatalf("expected occurrences header:\n%s", output)
	}

	if !strings.Contains(output, "1. Landmark region is missing a label") {
		t.Fatalf("expected failure summary occurrence detail:\n%s", output)
	}
}

func TestRenderUnifiedReport_LighthouseScoresMarkdown(t *testing.T) {
	status := apiclient.JobStatus{ID: "job-lh", State: apiclient.JobStateDone}
	doc := testsupport.SampleReport(status.ID)
	doc.Summary.LighthouseCategories = []report.LighthouseCategorySummary{
		{Id: "performance", Title: "Performance", AvgScore: 0.85},
		{Id: "accessibility", Title: "Accessibility", AvgScore: 0.62},
	}

	var buf bytes.Buffer

	err := UnifiedReport(&buf, "http://localhost:8080", status, doc, RenderOptions{
		Format:    FormatMarkdown,
		MaxIssues: 10,
	})
	testsupport.RequireNoErr(t, err)

	output := buf.String()
	for _, want := range []string{
		"## Lighthouse Scores",
		"| Performance | 0.85 |",
		"| Accessibility | 0.62 |",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("markdown output missing %q:\n%s", want, output)
		}
	}
}

func TestRenderUnifiedReport_LighthouseScoresText(t *testing.T) {
	status := apiclient.JobStatus{ID: "job-lh-text", State: apiclient.JobStateDone}
	doc := testsupport.SampleReport(status.ID)
	doc.Summary.LighthouseCategories = []report.LighthouseCategorySummary{
		{Id: "performance", Title: "Performance", AvgScore: 0.85},
		{Id: "seo", Title: "SEO", AvgScore: 0.91},
	}

	var buf bytes.Buffer

	err := UnifiedReport(&buf, "http://localhost:8080", status, doc, RenderOptions{
		Format:    FormatText,
		MaxIssues: 10,
	})
	testsupport.RequireNoErr(t, err)

	output := buf.String()
	if !strings.Contains(output, "Lighthouse: performance=0.85 seo=0.91") {
		t.Fatalf("text output missing Lighthouse line:\n%s", output)
	}
}

func TestRenderUnifiedReport_SummaryOnly(t *testing.T) {
	status := apiclient.JobStatus{ID: "job-summ", State: apiclient.JobStateDone}
	doc := testsupport.SampleReport(status.ID)

	var buf bytes.Buffer

	err := UnifiedReport(&buf, "http://localhost:8080", status, doc, RenderOptions{
		Format:      FormatMarkdown,
		MaxIssues:   10,
		SummaryOnly: true,
	})
	testsupport.RequireNoErr(t, err)

	output := buf.String()
	if strings.Contains(output, "## Findings") {
		t.Fatalf("summary-only should skip findings:\n%s", output)
	}

	if !strings.Contains(output, "## Scan Summary") {
		t.Fatalf("summary-only should include summary:\n%s", output)
	}

	if !strings.Contains(output, "## Scanners") {
		t.Fatalf("summary-only should include scanners:\n%s", output)
	}
}

func TestRenderUnifiedReport_SummaryOnlyText(t *testing.T) {
	status := apiclient.JobStatus{ID: "job-summ-text", State: apiclient.JobStateDone}
	doc := testsupport.SampleReport(status.ID)

	var buf bytes.Buffer

	err := UnifiedReport(&buf, "http://localhost:8080", status, doc, RenderOptions{
		Format:      FormatText,
		MaxIssues:   10,
		SummaryOnly: true,
	})
	testsupport.RequireNoErr(t, err)

	output := buf.String()
	if strings.Contains(output, "\nIssues:\n") {
		t.Fatalf("summary-only text should skip issues section:\n%s", output)
	}

	if strings.Contains(output, "Critical issue") {
		t.Fatalf("summary-only text should not contain issue details:\n%s", output)
	}
}

func TestRenderUnifiedReport_GroupByCategory(t *testing.T) {
	status := apiclient.JobStatus{ID: "job-group", State: apiclient.JobStateDone}
	doc := testsupport.SampleReport(status.ID)

	a11y := "accessibility"
	seo := "seo"
	doc.Issues = []report.IssueDetail{
		{
			Id:       "a",
			Scanner:  "axe",
			RuleId:   "r1",
			Severity: report.IssueSeverityCritical,
			Title:    "A11y issue",
			Category: &a11y,
			PageUrl:  "https://example.com",
		},
		{
			Id:       "b",
			Scanner:  "seo",
			RuleId:   "r2",
			Severity: report.IssueSeveritySerious,
			Title:    "SEO issue",
			Category: &seo,
			PageUrl:  "https://example.com",
		},
		{
			Id:       "c",
			Scanner:  "axe",
			RuleId:   "r3",
			Severity: report.IssueSeverityMinor,
			Title:    "Another a11y",
			Category: &a11y,
			PageUrl:  "https://example.com",
		},
	}

	var buf bytes.Buffer

	err := UnifiedReport(&buf, "http://localhost:8080", status, doc, RenderOptions{
		Format:    FormatMarkdown,
		MaxIssues: 10,
		GroupBy:   "category",
	})
	testsupport.RequireNoErr(t, err)

	output := buf.String()
	for _, want := range []string{
		"### accessibility",
		"### seo",
		"A11y issue",
		"SEO issue",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("grouped output missing %q:\n%s", want, output)
		}
	}

	// accessibility should come before seo alphabetically
	a11yIdx := strings.Index(output, "### accessibility")

	seoIdx := strings.Index(output, "### seo")
	if a11yIdx > seoIdx {
		t.Fatalf("accessibility should come before seo in grouped output")
	}
}

func TestRenderUnifiedReport_GroupByScanner(t *testing.T) {
	status := apiclient.JobStatus{ID: "job-group-scan", State: apiclient.JobStateDone}
	doc := testsupport.SampleReport(status.ID)

	doc.Issues = []report.IssueDetail{
		{
			Id:       "a",
			Scanner:  "axe",
			RuleId:   "r1",
			Severity: report.IssueSeverityCritical,
			Title:    "Axe issue",
			PageUrl:  "https://example.com",
		},
		{
			Id:       "b",
			Scanner:  "lighthouse",
			RuleId:   "r2",
			Severity: report.IssueSeveritySerious,
			Title:    "LH issue",
			PageUrl:  "https://example.com",
		},
	}

	var buf bytes.Buffer

	err := UnifiedReport(&buf, "http://localhost:8080", status, doc, RenderOptions{
		Format:    FormatMarkdown,
		MaxIssues: 10,
		GroupBy:   "scanner",
	})
	testsupport.RequireNoErr(t, err)

	output := buf.String()
	if !strings.Contains(output, "### axe") {
		t.Fatalf("grouped output missing ### axe:\n%s", output)
	}

	if !strings.Contains(output, "### lighthouse") {
		t.Fatalf("grouped output missing ### lighthouse:\n%s", output)
	}
}

func TestRenderUnifiedReport_GroupByNone(t *testing.T) {
	status := apiclient.JobStatus{ID: "job-group-none", State: apiclient.JobStateDone}
	doc := testsupport.SampleReport(status.ID)

	var buf bytes.Buffer

	err := UnifiedReport(&buf, "http://localhost:8080", status, doc, RenderOptions{
		Format:    FormatMarkdown,
		MaxIssues: 10,
		GroupBy:   "none",
	})
	testsupport.RequireNoErr(t, err)

	output := buf.String()
	if strings.Contains(output, "###") {
		t.Fatalf("flat output should not contain ### headings:\n%s", output)
	}
}

func TestRenderUnifiedReport_FailSeverity(t *testing.T) {
	status := apiclient.JobStatus{ID: "job-fail", State: apiclient.JobStateDone}
	doc := testsupport.SampleReport(status.ID)

	var buf bytes.Buffer

	err := UnifiedReport(&buf, "http://localhost:8080", status, doc, RenderOptions{
		Format:       FormatText,
		MaxIssues:    10,
		FailSeverity: "critical",
	})

	var ece exitcode.Error
	if !errors.As(err, &ece) {
		t.Fatalf("expected exitcode.Error, got %v", err)
	}

	testsupport.RequireEqual(t, ece.Code, 1, "exit code")
}

func TestRenderUnifiedReport_FailSeverity_NoMatch(t *testing.T) {
	status := apiclient.JobStatus{ID: "job-fail-no", State: apiclient.JobStateDone}
	doc := testsupport.SampleReport(status.ID)
	doc.Issues = []report.IssueDetail{
		{Id: "a", Severity: report.IssueSeverityMinor},
	}

	var buf bytes.Buffer

	err := UnifiedReport(&buf, "http://localhost:8080", status, doc, RenderOptions{
		Format:       FormatText,
		MaxIssues:    10,
		FailSeverity: "critical",
	})
	testsupport.RequireNoErr(t, err)
}

func TestRenderUnifiedReport_FailSeverity_UsesFilteredIssues(t *testing.T) {
	status := apiclient.JobStatus{ID: "job-fail-filtered", State: apiclient.JobStateDone}
	doc := testsupport.SampleReport(status.ID)

	a11y := "accessibility"
	seo := "seo"
	doc.Issues = []report.IssueDetail{
		{
			Id:       "a",
			Scanner:  "axe",
			RuleId:   "r1",
			Severity: report.IssueSeveritySerious,
			Title:    "A11y serious",
			Category: &a11y,
			PageUrl:  "https://example.com",
		},
		{
			Id:       "b",
			Scanner:  "seo",
			RuleId:   "r2",
			Severity: report.IssueSeverityMinor,
			Title:    "SEO minor",
			Category: &seo,
			PageUrl:  "https://example.com",
		},
	}

	var buf bytes.Buffer

	err := UnifiedReport(&buf, "http://localhost:8080", status, doc, RenderOptions{
		Format:       FormatText,
		MaxIssues:    10,
		Categories:   []string{"seo"},
		FailSeverity: "serious",
	})
	testsupport.RequireNoErr(t, err)
}

func TestRenderUnifiedReport_FailSeverity_Invalid(t *testing.T) {
	status := apiclient.JobStatus{ID: "job-fail-invalid", State: apiclient.JobStateDone}
	doc := testsupport.SampleReport(status.ID)

	var buf bytes.Buffer

	err := UnifiedReport(&buf, "http://localhost:8080", status, doc, RenderOptions{
		Format:       FormatText,
		MaxIssues:    10,
		FailSeverity: "warning",
	})
	if err == nil {
		t.Fatal("expected invalid fail severity error")
	}
}

func TestRenderUnifiedReport_SeverityFilter(t *testing.T) {
	status := apiclient.JobStatus{ID: "job-sev-filter", State: apiclient.JobStateDone}
	doc := testsupport.SampleReport(status.ID)

	var buf bytes.Buffer

	err := UnifiedReport(&buf, "http://localhost:8080", status, doc, RenderOptions{
		Format:     FormatMarkdown,
		MaxIssues:  10,
		Severities: []string{"critical"},
	})
	testsupport.RequireNoErr(t, err)

	output := buf.String()
	if !strings.Contains(output, "Critical issue") {
		t.Fatalf("filtered output should contain critical issue:\n%s", output)
	}

	if strings.Contains(output, "Minor issue") {
		t.Fatalf("filtered output should not contain minor issue:\n%s", output)
	}

	if !strings.Contains(output, "Severity filter: critical") {
		t.Fatalf("filtered output should show severity filter:\n%s", output)
	}
}

func TestRenderUnifiedReport_CategoryFilter(t *testing.T) {
	status := apiclient.JobStatus{ID: "job-cat-filter", State: apiclient.JobStateDone}
	doc := testsupport.SampleReport(status.ID)

	a11y := "accessibility"
	seo := "seo"
	doc.Issues = []report.IssueDetail{
		{
			Id:       "a",
			Scanner:  "axe",
			RuleId:   "r1",
			Severity: report.IssueSeverityCritical,
			Title:    "A11y issue",
			Category: &a11y,
			PageUrl:  "https://example.com",
		},
		{
			Id:       "b",
			Scanner:  "seo",
			RuleId:   "r2",
			Severity: report.IssueSeveritySerious,
			Title:    "SEO issue",
			Category: &seo,
			PageUrl:  "https://example.com",
		},
	}

	var buf bytes.Buffer

	err := UnifiedReport(&buf, "http://localhost:8080", status, doc, RenderOptions{
		Format:     FormatMarkdown,
		MaxIssues:  10,
		Categories: []string{"accessibility"},
	})
	testsupport.RequireNoErr(t, err)

	output := buf.String()
	if !strings.Contains(output, "A11y issue") {
		t.Fatalf("filtered output should contain a11y issue:\n%s", output)
	}

	if strings.Contains(output, "SEO issue") {
		t.Fatalf("filtered output should not contain SEO issue:\n%s", output)
	}
}

func TestRenderUnifiedReport_MaxOccurrencesTruncates(t *testing.T) {
	status := apiclient.JobStatus{ID: "job-max-occ", State: apiclient.JobStateDone}
	doc := testsupport.SampleReport(status.ID)

	sel1 := ".a"
	sel2 := ".b"
	sel3 := ".c"
	html := "<p>x</p>"

	doc.Issues = []report.IssueDetail{
		{
			Id: "issue-1", Scanner: "axe", RuleId: "r1", Severity: report.IssueSeverityCritical,
			Title: "Test", PageUrl: "https://example.com",
			Occurrences: []report.IssueOccurrence{
				{Selector: &sel1, Html: &html},
				{Selector: &sel2, Html: &html},
				{Selector: &sel3, Html: &html},
			},
			ElementCount: 3,
		},
	}

	var buf bytes.Buffer

	err := UnifiedReport(&buf, "http://localhost:8080", status, doc, RenderOptions{
		Format:         FormatMarkdown,
		MaxIssues:      10,
		MaxOccurrences: 2,
	})
	testsupport.RequireNoErr(t, err)

	output := buf.String()
	if !strings.Contains(output, "Occurrences (2 of 3):") {
		t.Fatalf("should show truncated occurrences:\n%s", output)
	}

	if strings.Contains(output, "`.c`") {
		t.Fatalf("third occurrence should be truncated:\n%s", output)
	}
}
