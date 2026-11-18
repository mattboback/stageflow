package render

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
	"github.com/mattboback/stageflow/clients/cli/internal/testsupport"
	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
)

func TestMarkdownDigestGroupsAcrossPagesAndEscapesMarkup(t *testing.T) {
	doc := testsupport.SampleReport("digest")
	fix := "Set <title>My page</title>."
	doc.Issues = []report.IssueDetail{
		{
			Id:       "a",
			Scanner:  "seo",
			RuleId:   "title",
			Title:    "Missing title",
			Severity: report.IssueSeveritySerious,
			PageUrl:  "https://example.com/a",
			HowToFix: &fix,
		},
		{
			Id:       "b",
			Scanner:  "seo",
			RuleId:   "title",
			Title:    "Missing title",
			Severity: report.IssueSeveritySerious,
			PageUrl:  "https://example.com/b",
			HowToFix: &fix,
		},
	}

	var out bytes.Buffer

	err := UnifiedReport(
		&out,
		"https://stageflow.org",
		apiclient.JobStatus{ID: "digest", State: apiclient.JobStateDone},
		doc,
		Options{Format: FormatMarkdown},
	)
	if err != nil {
		t.Fatal(err)
	}

	text := out.String()
	for _, want := range []string{"2 issue record(s) across 2 page(s)", "&lt;title&gt;My page&lt;/title&gt;", "[View full report (24h)](https://stageflow.org/scan/digest/report)"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in %s", want, text)
		}
	}

	if strings.Count(text, "Missing title") != 1 {
		t.Fatalf("duplicate rule groups: %s", text)
	}

	for _, forbidden := range []string{"Results Path", "Report Path", "## Artifacts", "## Pages", "No report errors"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("digest contains %q", forbidden)
		}
	}
}

func TestMarkdownDigestPartialNeverDisplaysStoredScore(t *testing.T) {
	doc := testsupport.SampleReport("partial")
	doc.Scanners[0].Status = report.ScannerStatusPartial

	var out bytes.Buffer
	if err := UnifiedReport(
		&out,
		"https://stageflow.org",
		apiclient.JobStatus{ID: "partial"},
		doc,
		Options{Format: FormatMarkdown},
	); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(out.String(), "Partial report") || strings.Contains(out.String(), "**Score:**") {
		t.Fatal(out.String())
	}
}
