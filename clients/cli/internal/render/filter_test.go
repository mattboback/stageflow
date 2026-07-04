package render

import (
	"testing"

	"github.com/mattboback/stageflow/clients/cli/internal/testsupport"
	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
)

func TestParseMinimumSeverity(t *testing.T) {
	ranks := []struct {
		sev  report.IssueSeverity
		rank int
	}{
		{report.IssueSeverityInfo, 0},
		{report.IssueSeverityMinor, 1},
		{report.IssueSeverityModerate, 2},
		{report.IssueSeveritySerious, 3},
		{report.IssueSeverityCritical, 4},
	}

	for _, item := range ranks {
		if got := severityRank(item.sev); got != item.rank {
			t.Fatalf("severityRank(%q) = %d, want %d", item.sev, got, item.rank)
		}
	}
}

func TestSelectIssues_SortsAndTruncates(t *testing.T) {
	issues := []report.IssueDetail{
		{Id: "b", Severity: report.IssueSeveritySerious, Scanner: "seo", RuleId: "r2", PageUrl: "https://b"},
		{Id: "a", Severity: report.IssueSeverityCritical, Scanner: "axe", RuleId: "r1", PageUrl: "https://a"},
		{Id: "c", Severity: report.IssueSeveritySerious, Scanner: "axe", RuleId: "r1", PageUrl: "https://c"},
		{Id: "d", Severity: report.IssueSeveritySerious, Scanner: "axe", RuleId: "r1", PageUrl: "https://b"},
		{Id: "e", Severity: report.IssueSeverityMinor, Scanner: "axe", RuleId: "r9", PageUrl: "https://e"},
	}

	selected, filters := selectIssues(issues, 3)

	if !filters.Truncated {
		t.Fatalf("filters.Truncated = %v, want true", filters.Truncated)
	}

	if filters.IssuesTotal != len(issues) {
		t.Fatalf("filters.IssuesTotal = %d, want %d", filters.IssuesTotal, len(issues))
	}

	if len(selected) != 3 {
		t.Fatalf("len(selected) = %d, want 3", len(selected))
	}

	// Critical comes first, then serious sorted by scanner/rule/page/id.
	if selected[0].Id != "a" {
		t.Fatalf("selected[0].Id = %q, want %q", selected[0].Id, "a")
	}

	if selected[1].Id != "d" {
		t.Fatalf("selected[1].Id = %q, want %q", selected[1].Id, "d")
	}

	if selected[2].Id != "c" {
		t.Fatalf("selected[2].Id = %q, want %q", selected[2].Id, "c")
	}
}

func TestFilterBySeverity(t *testing.T) {
	issues := []report.IssueDetail{
		{Id: "a", Severity: report.IssueSeverityCritical},
		{Id: "b", Severity: report.IssueSeveritySerious},
		{Id: "c", Severity: report.IssueSeverityMinor},
		{Id: "d", Severity: report.IssueSeverityCritical},
	}

	filtered := filterBySeverity(issues, []string{"critical"})
	testsupport.RequireEqual(t, len(filtered), 2, "len(filtered)")
	testsupport.RequireEqual(t, filtered[0].Id, "a", "filtered[0].Id")
	testsupport.RequireEqual(t, filtered[1].Id, "d", "filtered[1].Id")
}

func TestFilterBySeverity_MultipleLevels(t *testing.T) {
	issues := []report.IssueDetail{
		{Id: "a", Severity: report.IssueSeverityCritical},
		{Id: "b", Severity: report.IssueSeveritySerious},
		{Id: "c", Severity: report.IssueSeverityMinor},
	}

	filtered := filterBySeverity(issues, []string{"critical", "serious"})
	testsupport.RequireEqual(t, len(filtered), 2, "len(filtered)")
	testsupport.RequireEqual(t, filtered[0].Id, "a", "filtered[0].Id")
	testsupport.RequireEqual(t, filtered[1].Id, "b", "filtered[1].Id")
}

func TestFilterBySeverity_Empty(t *testing.T) {
	issues := []report.IssueDetail{
		{Id: "a", Severity: report.IssueSeverityCritical},
	}

	filtered := filterBySeverity(issues, nil)
	testsupport.RequireEqual(t, len(filtered), 1, "len(filtered)")
}

func TestFilterByCategory(t *testing.T) {
	a11y := "accessibility"
	seo := "seo"

	issues := []report.IssueDetail{
		{Id: "a", Category: &a11y},
		{Id: "b", Category: &seo},
		{Id: "c", Category: &a11y},
		{Id: "d"},
	}

	filtered := filterByCategory(issues, []string{"accessibility"})
	testsupport.RequireEqual(t, len(filtered), 2, "len(filtered)")
	testsupport.RequireEqual(t, filtered[0].Id, "a", "filtered[0].Id")
	testsupport.RequireEqual(t, filtered[1].Id, "c", "filtered[1].Id")
}

func TestFilterByCategory_Empty(t *testing.T) {
	issues := []report.IssueDetail{
		{Id: "a"},
	}

	filtered := filterByCategory(issues, nil)
	testsupport.RequireEqual(t, len(filtered), 1, "len(filtered)")
}

func TestHasIssuesAtOrAbove(t *testing.T) {
	issues := []report.IssueDetail{
		{Severity: report.IssueSeverityMinor},
		{Severity: report.IssueSeveritySerious},
	}

	matched, err := HasIssuesAtOrAbove(issues, "critical")
	testsupport.RequireNoErr(t, err)
	testsupport.RequireEqual(t, matched, false, "critical threshold")

	matched, err = HasIssuesAtOrAbove(issues, "serious")
	testsupport.RequireNoErr(t, err)
	testsupport.RequireEqual(t, matched, true, "serious threshold")

	matched, err = HasIssuesAtOrAbove(issues, "minor")
	testsupport.RequireNoErr(t, err)
	testsupport.RequireEqual(t, matched, true, "minor threshold")

	matched, err = HasIssuesAtOrAbove(issues, "info")
	testsupport.RequireNoErr(t, err)
	testsupport.RequireEqual(t, matched, true, "info threshold")
}

func TestHasIssuesAtOrAbove_NoIssues(t *testing.T) {
	matched, err := HasIssuesAtOrAbove(nil, "critical")
	testsupport.RequireNoErr(t, err)
	testsupport.RequireEqual(t, matched, false, "nil issues")
}

func TestHasIssuesAtOrAbove_InvalidSeverity(t *testing.T) {
	_, err := HasIssuesAtOrAbove(nil, "warning")
	if err == nil {
		t.Fatal("expected invalid severity error")
	}
}

func TestNormalizeSeverities_Invalid(t *testing.T) {
	_, err := normalizeSeverities([]string{"critical", "warning"})
	if err == nil {
		t.Fatal("expected invalid severity error")
	}
}

func TestNormalizeCategories_Invalid(t *testing.T) {
	_, err := normalizeCategories([]string{"accessibility", "ops"})
	if err == nil {
		t.Fatal("expected invalid category error")
	}
}

func TestSplitCSV(t *testing.T) {
	testsupport.RequireDeepEqual(t, splitCSV("critical,serious"), []string{"critical", "serious"}, "basic")
	testsupport.RequireDeepEqual(t, splitCSV(" critical , serious "), []string{"critical", "serious"}, "with spaces")
	testsupport.RequireDeepEqual(t, splitCSV(""), []string(nil), "empty")
	testsupport.RequireDeepEqual(t, splitCSV("  "), []string(nil), "whitespace")
	testsupport.RequireDeepEqual(t, splitCSV("single"), []string{"single"}, "single value")
}
