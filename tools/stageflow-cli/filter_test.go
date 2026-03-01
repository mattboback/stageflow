package main

import (
	"testing"

	report "github.com/mattboback/stageflow/packages/contracts/report/generated/go"
)

func TestParseMinimumSeverity(t *testing.T) {
	level, err := parseMinimumSeverity("serious")
	if err != nil {
		t.Fatalf("parseMinimumSeverity returned error: %v", err)
	}

	if level != severitySerious {
		t.Fatalf("level = %v, want %v", level, severitySerious)
	}

	_, invalidErr := parseMinimumSeverity("blocker")
	if invalidErr == nil {
		t.Fatal("expected invalid severity error")
	}
}

func TestFilterIssues(t *testing.T) {
	issues := sampleReport("job-filter").Issues

	filtered := filterIssues(issues, severitySerious, 0)
	if len(filtered) != 1 || filtered[0].Severity != report.IssueSeverityCritical {
		t.Fatalf("filtered serious+ = %#v", filtered)
	}

	filtered = filterIssues(issues, severityInfo, 1)
	if len(filtered) != 1 {
		t.Fatalf("len(filtered) = %d, want 1", len(filtered))
	}
}
