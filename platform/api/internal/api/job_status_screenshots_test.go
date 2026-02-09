package api

import (
	"testing"

	report "github.com/mattboback/stageflow/packages/contracts/report/generated/go"
)

func TestBuildDerivedIssueID(t *testing.T) {
	t.Parallel()

	if got, want := buildDerivedIssueID("abc123", 0), "abc123"; got != want {
		t.Fatalf("occurrence 0: want %q, got %q", want, got)
	}

	if got, want := buildDerivedIssueID("abc123", 1), "abc123--occ-2"; got != want {
		t.Fatalf("occurrence 1: want %q, got %q", want, got)
	}
}

func TestCollectScreenshotArtifactPaths(t *testing.T) {
	t.Parallel()

	path := "screenshots/shot.webp"
	ignoredPath := "screenshots/not-used.webp"

	paths := collectScreenshotArtifactPaths([]report.ReportArtifact{
		{
			Id:   "a1",
			Type: "screenshot",
			Path: &path,
		},
		{
			Id:   "ignore-type",
			Type: "har",
			Path: &ignoredPath,
		},
		{
			Id:   "",
			Type: "screenshot",
			Path: &ignoredPath,
		},
	})

	if got, want := len(paths), 1; got != want {
		t.Fatalf("expected %d screenshot path, got %d", want, got)
	}

	if got, ok := paths["a1"]; !ok || got != "screenshots/shot.webp" {
		t.Fatalf("expected artifact path for a1, got %q (exists=%v)", got, ok)
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

func TestBuildPageOverviewArtifactID(t *testing.T) {
	t.Parallel()

	if got, want := buildPageOverviewArtifactID("axe", "page-1"), "page-overview:axe:page-1"; got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestBuildViolationScreenshotArtifact(t *testing.T) {
	t.Parallel()

	artifact, ok := buildViolationScreenshotArtifact(
		"issue-a",
		1,
		"ss-issue-a",
		"axe",
		"page-1",
		"https://example.com",
		"https://minio/shot.webp",
	)
	if !ok {
		t.Fatal("expected valid violation screenshot artifact")
	}
	if artifact.Kind != "violation" {
		t.Fatalf("expected kind=violation, got %q", artifact.Kind)
	}
	if artifact.OccurrenceIndex == nil || *artifact.OccurrenceIndex != 1 {
		t.Fatalf("expected occurrence index 1, got %+v", artifact.OccurrenceIndex)
	}
}

func TestBuildViolationScreenshotArtifact_RejectsMissingIdentityFields(t *testing.T) {
	t.Parallel()

	_, ok := buildViolationScreenshotArtifact(
		"",
		0,
		"ss-issue-a",
		"axe",
		"page-1",
		"https://example.com",
		"https://minio/shot.webp",
	)
	if ok {
		t.Fatal("expected missing issue_id to be rejected")
	}

	_, ok = buildViolationScreenshotArtifact(
		"issue-a",
		0,
		"",
		"axe",
		"page-1",
		"https://example.com",
		"https://minio/shot.webp",
	)
	if ok {
		t.Fatal("expected missing artifact_id to be rejected")
	}
}
