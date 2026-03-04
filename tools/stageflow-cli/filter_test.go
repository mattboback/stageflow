package main

import (
	"testing"

	report "github.com/mattboback/stageflow/packages/contracts/report/generated/go"
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
	requireEqual(t, len(filtered), 2, "len(filtered)")
	requireEqual(t, filtered[0].Id, "a", "filtered[0].Id")
	requireEqual(t, filtered[1].Id, "d", "filtered[1].Id")
}

func TestFilterBySeverity_MultipleLevels(t *testing.T) {
	issues := []report.IssueDetail{
		{Id: "a", Severity: report.IssueSeverityCritical},
		{Id: "b", Severity: report.IssueSeveritySerious},
		{Id: "c", Severity: report.IssueSeverityMinor},
	}

	filtered := filterBySeverity(issues, []string{"critical", "serious"})
	requireEqual(t, len(filtered), 2, "len(filtered)")
	requireEqual(t, filtered[0].Id, "a", "filtered[0].Id")
	requireEqual(t, filtered[1].Id, "b", "filtered[1].Id")
}

func TestFilterBySeverity_Empty(t *testing.T) {
	issues := []report.IssueDetail{
		{Id: "a", Severity: report.IssueSeverityCritical},
	}

	filtered := filterBySeverity(issues, nil)
	requireEqual(t, len(filtered), 1, "len(filtered)")
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
	requireEqual(t, len(filtered), 2, "len(filtered)")
	requireEqual(t, filtered[0].Id, "a", "filtered[0].Id")
	requireEqual(t, filtered[1].Id, "c", "filtered[1].Id")
}

func TestFilterByCategory_Empty(t *testing.T) {
	issues := []report.IssueDetail{
		{Id: "a"},
	}

	filtered := filterByCategory(issues, nil)
	requireEqual(t, len(filtered), 1, "len(filtered)")
}

func TestHasIssuesAtOrAbove(t *testing.T) {
	issues := []report.IssueDetail{
		{Severity: report.IssueSeverityMinor},
		{Severity: report.IssueSeveritySerious},
	}

	matched, err := hasIssuesAtOrAbove(issues, "critical")
	requireNoErr(t, err)
	requireEqual(t, matched, false, "critical threshold")

	matched, err = hasIssuesAtOrAbove(issues, "serious")
	requireNoErr(t, err)
	requireEqual(t, matched, true, "serious threshold")

	matched, err = hasIssuesAtOrAbove(issues, "minor")
	requireNoErr(t, err)
	requireEqual(t, matched, true, "minor threshold")

	matched, err = hasIssuesAtOrAbove(issues, "info")
	requireNoErr(t, err)
	requireEqual(t, matched, true, "info threshold")
}

func TestHasIssuesAtOrAbove_NoIssues(t *testing.T) {
	matched, err := hasIssuesAtOrAbove(nil, "critical")
	requireNoErr(t, err)
	requireEqual(t, matched, false, "nil issues")
}

func TestHasIssuesAtOrAbove_InvalidSeverity(t *testing.T) {
	_, err := hasIssuesAtOrAbove(nil, "warning")
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
	requireDeepEqual(t, splitCSV("critical,serious"), []string{"critical", "serious"}, "basic")
	requireDeepEqual(t, splitCSV(" critical , serious "), []string{"critical", "serious"}, "with spaces")
	requireDeepEqual(t, splitCSV(""), []string(nil), "empty")
	requireDeepEqual(t, splitCSV("  "), []string(nil), "whitespace")
	requireDeepEqual(t, splitCSV("single"), []string{"single"}, "single value")
}
