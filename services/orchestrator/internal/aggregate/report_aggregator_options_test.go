package aggregate

import (
	"context"
	"encoding/json"
	"testing"

	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
	"github.com/mattboback/stageflow/libs/go/models"
)

func TestAxeCoverageOptionsRetainFallbackUnlessUnrestricted(t *testing.T) {
	cases := []struct {
		name         string
		config       string
		wantFallback bool
	}{
		{name: "absent", config: `{}`},
		{name: "empty scanner configs", config: `{"scanner_configs":{}}`},
		{name: "empty axe config", config: `{"scanner_configs":{"axe":{}}}`},
		{name: "other scanner options", config: `{"scanner_configs":{"seo":{"custom":true}}}`},
		{name: "wait only", config: `{"scanner_configs":{"axe":{"dynamicContentWaitMs":500}}}`},
		{name: "empty disabled rules", config: `{"scanner_configs":{"axe":{"disabledRules":[]}}}`},
		{name: "empty tags", config: `{"scanner_configs":{"axe":{"runOnlyTags":[]}}}`},
		{
			name:   "empty filters and wait",
			config: `{"scanner_configs":{"axe":{"disabledRules":[],"runOnlyTags":[],"dynamicContentWaitMs":0}}}`,
		},
		{
			name:         "disabled overlapping rule",
			config:       `{"scanner_configs":{"axe":{"disabledRules":["color-contrast"]}}}`,
			wantFallback: true,
		},
		{name: "restricted tags", config: `{"scanner_configs":{"axe":{"runOnlyTags":["wcag2a"]}}}`, wantFallback: true},
		{name: "unknown key", config: `{"scanner_configs":{"axe":{"futureFilter":[]}}}`, wantFallback: true},
		{
			name:         "unknown key and wait",
			config:       `{"scanner_configs":{"axe":{"dynamicContentWaitMs":500,"rules":[]}}}`,
			wantFallback: true,
		},
		{name: "null disabled rules", config: `{"scanner_configs":{"axe":{"disabledRules":null}}}`, wantFallback: true},
		{name: "string disabled rules", config: `{"scanner_configs":{"axe":{"disabledRules":""}}}`, wantFallback: true},
		{name: "object disabled rules", config: `{"scanner_configs":{"axe":{"disabledRules":{}}}}`, wantFallback: true},
		{name: "null tags", config: `{"scanner_configs":{"axe":{"runOnlyTags":null}}}`, wantFallback: true},
		{name: "boolean tags", config: `{"scanner_configs":{"axe":{"runOnlyTags":false}}}`, wantFallback: true},
		{name: "number tags", config: `{"scanner_configs":{"axe":{"runOnlyTags":0}}}`, wantFallback: true},
		{
			name:         "malformed filter elements",
			config:       `{"scanner_configs":{"axe":{"runOnlyTags":[null]}}}`,
			wantFallback: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			job := &models.Job{ID: "coverage-options"}
			// Intake and PostgreSQL both decode nested JSON arrays into []any.
			if err := json.Unmarshal([]byte(tc.config), &job.Config); err != nil {
				t.Fatal(err)
			}

			assertAxeFallbackCoverage(t, job, tc.wantFallback)
		})
	}
}

func TestAxeCoverageOptionsWithGoFilterSlices(t *testing.T) {
	cases := []struct {
		name         string
		options      map[string]any
		wantFallback bool
	}{
		{name: "empty string slices", options: map[string]any{"disabledRules": []string{}, "runOnlyTags": []string{}}},
		{name: "nil string slice", options: map[string]any{"disabledRules": []string(nil)}, wantFallback: true},
		{name: "nil decoded slice", options: map[string]any{"runOnlyTags": []any(nil)}, wantFallback: true},
		{
			name:         "disabled rule",
			options:      map[string]any{"disabledRules": []string{"color-contrast"}},
			wantFallback: true,
		},
		{name: "nonstring empty slice", options: map[string]any{"runOnlyTags": []int{}}, wantFallback: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			job := &models.Job{ID: "coverage-go-options", Config: models.JobConfig{
				ScannerConfigs: map[string]map[string]any{"axe": tc.options},
			}}
			assertAxeFallbackCoverage(t, job, tc.wantFallback)
		})
	}
}

func assertAxeFallbackCoverage(t *testing.T, job *models.Job, wantFallback bool) {
	t.Helper()

	a := newReportAggregation(&Aggregator{}, job)
	a.scannerStats["axe"] = &report.ScannerSummary{Id: "axe", Status: report.ScannerStatusSuccess}
	a.issues = []report.IssueDetail{
		{Id: "fallback", Scanner: "lighthouse", RuleId: "color-contrast", Category: stringPtr("accessibility")},
		{Id: "alt-fallback", Scanner: "seo", RuleId: "seo-images-missing-alt"},
		{Id: "link-fallback", Scanner: "link-checker", RuleId: "link-checker-no-text-links"},
		{Id: "independent", Scanner: "seo", RuleId: "seo-missing-description"},
	}
	got := a.coverageFilteredIssues(nil)

	wantCount := 1
	if wantFallback {
		wantCount += 3
	}

	if len(got) != wantCount || got[len(got)-1].Id != "independent" {
		t.Fatalf(
			"fallback=%v: got %#v, want %d issues including the independent SEO finding",
			wantFallback,
			got,
			wantCount,
		)
	}

	if wantFallback && got[0].Id != "fallback" {
		t.Fatal("restricted axe coverage removed the Lighthouse fallback")
	}
}

func TestBuildAggregatedReportFilteredAxePreservesFindingsAndCounts(t *testing.T) {
	for _, config := range []string{
		`{}`,
		`{"disabledRules":["color-contrast"]}`,
		`{"runOnlyTags":["wcag2a"]}`,
	} {
		t.Run(config, func(t *testing.T) {
			job := &models.Job{ID: "filtered-axe", ExpectedScanners: []string{"axe", "lighthouse"}}

			var options map[string]any
			if err := json.Unmarshal([]byte(config), &options); err != nil {
				t.Fatal(err)
			}

			job.Config.ScannerConfigs = map[string]map[string]any{"axe": options}
			mem := newMemoryFiles()
			page := report.PageSummary{Id: "page", Url: "https://example.com/"}
			incomplete := report.IssueDetail{
				Id:          "contrast-review",
				Scanner:     "axe",
				RuleId:      "color-contrast",
				PageId:      page.Id,
				PageUrl:     page.Url,
				Title:       "Contrast needs review",
				Description: "Background color could not be determined.",
				Severity:    report.IssueSeverityModerate,
				ScannerData: map[string]any{"axeIncomplete": true},
			}
			fallback := report.IssueDetail{
				Id: "contrast-fallback", Scanner: "lighthouse", RuleId: "color-contrast", PageId: page.Id,
				PageUrl: page.Url, Title: "Low contrast", Description: "Text contrast is insufficient.",
				Category: stringPtr("accessibility"), Severity: report.IssueSeveritySerious,
			}
			manual := report.IssueDetail{
				Id:          "manual",
				Scanner:     "lighthouse",
				RuleId:      "logical-tab-order",
				PageId:      page.Id,
				PageUrl:     page.Url,
				Title:       "Tab order",
				Description: "Manual verification required: Check keyboard order.",
				Severity:    report.IssueSeverityInfo,
				ScannerData: map[string]any{"lighthouseManual": true},
			}

			job.ScannerResults = make(map[string]*models.ScannerResult)
			for scannerID, issues := range map[string][]report.IssueDetail{
				"axe": {incomplete}, "lighthouse": {fallback, manual},
			} {
				doc := report.UnifiedReportV2{
					Version: reportVersion, Pages: []report.PageSummary{page}, Issues: issues,
					Meta:     report.ReportMeta{JobId: job.ID},
					Scanners: []report.ScannerSummary{{Id: scannerID, Status: report.ScannerStatusSuccess}},
				}

				data, err := json.Marshal(doc)
				if err != nil {
					t.Fatal(err)
				}

				key := job.ID + "/" + scannerID + "/results.json"
				mem.files[key] = data
				job.ScannerResults[scannerID] = &models.ScannerResult{Success: true, ResultsPath: key}
			}

			key, err := New(mem).BuildAggregatedReport(context.Background(), job)
			if err != nil {
				t.Fatal(err)
			}

			var result report.UnifiedReportV2
			if decodeErr := json.Unmarshal(mem.files[key], &result); decodeErr != nil {
				t.Fatal(decodeErr)
			}

			wantFallback := config != `{}`
			assertFilteredAxeReport(t, result, wantFallback)
		})
	}
}

func assertFilteredAxeReport(t *testing.T, result report.UnifiedReportV2, wantFallback bool) {
	t.Helper()

	wantIssues, wantSerious := 1, 0
	if wantFallback {
		wantIssues, wantSerious = 2, 1
	}

	assertReportFindingCounts(t, result, wantIssues, wantSerious)

	if len(result.ManualChecks) != 1 || result.ManualChecks[0].RuleId != "logical-tab-order" ||
		result.Assessment == nil || result.Assessment.NeedsReview != 1 || result.Assessment.Actionable != wantSerious {
		t.Fatalf("manual/incomplete findings changed: %#v / %#v", result.ManualChecks, result.Assessment)
	}

	for _, scanner := range result.Scanners {
		want := 1
		if scanner.Id == "lighthouse" {
			want = wantSerious
		}

		if scanner.IssueCount == nil || *scanner.IssueCount != want {
			t.Fatalf("scanner count does not match retained findings: %#v", scanner)
		}
	}
}

func assertReportFindingCounts(t *testing.T, result report.UnifiedReportV2, wantIssues, wantSerious int) {
	t.Helper()

	if len(result.Issues) != wantIssues || result.Summary.TotalIssues != wantIssues ||
		result.Summary.BySeverity.Serious != wantSerious || result.Summary.BySeverity.Moderate != 1 ||
		result.Summary.ByScanner["axe"] != 1 || result.Summary.ByScanner["lighthouse"] != wantSerious {
		t.Fatalf(
			"want %d findings: inconsistent findings or summary: %#v / %#v",
			wantIssues,
			result.Issues,
			result.Summary,
		)
	}

	if len(result.Pages) != 1 || result.Pages[0].IssueCount != wantIssues ||
		result.Pages[0].BySeverity == nil || result.Pages[0].BySeverity.Serious != wantSerious {
		t.Fatalf("want %d findings: inconsistent page counts: %#v", wantIssues, result.Pages)
	}
}
