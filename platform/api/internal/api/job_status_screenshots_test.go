package api

import (
	"testing"

	report "github.com/mattboback/stageflow/packages/contracts/report/generated/go"
)

func TestCollectScreenshotRefs(t *testing.T) {
	t.Parallel()

	refs := collectScreenshotRefs([]report.IssueDetail{
		{
			Scanner: "",
			RuleId:  "color-contrast",
			PageId:  "page-1",
			Occurrences: []report.IssueOccurrence{
				{ArtifactIds: []string{"a1", "a2", ""}},
			},
		},
		{
			Scanner: "lighthouse",
			RuleId:  "lh-performance",
			PageId:  "page-1",
			Occurrences: []report.IssueOccurrence{
				{ArtifactIds: []string{"a1"}},
			},
		},
		{
			Scanner: scannerTypeAxe,
			RuleId:  "",
			PageId:  "page-1",
			Occurrences: []report.IssueOccurrence{
				{ArtifactIds: []string{"ignored"}},
			},
		},
	}, scannerTypeAxe)

	if got := len(refs["a1"]); got != 2 {
		t.Fatalf("expected 2 refs for a1, got %d", got)
	}

	if got := refs["a1"][0].ScannerType; got != scannerTypeAxe {
		t.Fatalf("expected default scanner axe for first ref, got %q", got)
	}

	if got := refs["a2"][0].RuleID; got != "color-contrast" {
		t.Fatalf("expected rule id propagated, got %q", got)
	}

	if _, ok := refs["ignored"]; ok {
		t.Fatalf("expected issue with empty RuleID to be ignored")
	}
}

func TestResolveOverviewScannerV2(t *testing.T) {
	t.Parallel()

	issues := []report.IssueDetail{
		{Scanner: "lighthouse", PageId: "page-1"},
		{Scanner: scannerTypeAxe, PageId: "page-1"},
		{Scanner: "custom", PageId: "page-2"},
	}

	if got, want := resolveOverviewScannerV2(issues, "page-1"), scannerTypeAxe; got != want {
		t.Fatalf("page-1: want %q, got %q", want, got)
	}

	if got, want := resolveOverviewScannerV2(issues, "page-2"), "custom"; got != want {
		t.Fatalf("page-2: want %q, got %q", want, got)
	}

	if got, want := resolveOverviewScannerV2(issues, "page-3"), scannerTypeAxe; got != want {
		t.Fatalf("page-3: want default %q, got %q", want, got)
	}
}
