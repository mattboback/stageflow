package api

import (
	"context"
	"strings"
	"testing"

	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
)

func TestResolveOverviewArtifactV2PrefersAxe(t *testing.T) {
	t.Parallel()

	axePath := "screenshots/axe.webp"
	lighthousePath := "screenshots/lighthouse.webp"
	artifacts := []report.ReportArtifact{
		{
			Id:   "page-overview-lighthouse-page-1",
			Type: artifactTypePageOverview,
			Path: &lighthousePath,
		},
		{
			Id:   "page-overview-axe-page-1",
			Type: artifactTypePageOverview,
			Path: &axePath,
		},
	}
	scanners := []report.ScannerSummary{
		{Id: "lighthouse"},
		{Id: scannerTypeAxe},
	}

	scannerID, artifactPath, ok := resolveOverviewArtifactV2(artifacts, scanners, "page-1")
	if !ok {
		t.Fatal("expected page-overview artifact")
	}

	if got, want := scannerID, scannerTypeAxe; got != want {
		t.Fatalf("scanner ID: want %q, got %q", want, got)
	}

	if got, want := artifactPath, axePath; got != want {
		t.Fatalf("artifact path: want %q, got %q", want, got)
	}
}

func TestResolveOverviewArtifactV2IgnoresMalformedArtifacts(t *testing.T) {
	t.Parallel()

	wrongTypePath := "screenshots/wrong-type.webp"
	emptyPath := ""
	artifacts := []report.ReportArtifact{
		{
			Id:   "page-overview-lighthouse-page-1",
			Type: "screenshot",
			Path: &wrongTypePath,
		},
		{
			Id:   "page-overview-axe-page-1",
			Type: artifactTypePageOverview,
			Path: &emptyPath,
		},
	}
	scanners := []report.ScannerSummary{
		{Id: scannerTypeAxe},
		{Id: "lighthouse"},
	}

	if scannerID, artifactPath, ok := resolveOverviewArtifactV2(artifacts, scanners, "page-1"); ok {
		t.Fatalf("expected no artifact, got scanner %q and path %q", scannerID, artifactPath)
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

func TestExtractScreenshotsFromReportUsesOverviewArtifactOwner(t *testing.T) {
	t.Parallel()

	server := &Server{config: &ServerConfig{Storage: newFakeStorage()}}
	artifactPath := "screenshots/page-overview-page-1.webp"
	results := report.UnifiedReportV2{
		Scanners: []report.ScannerSummary{
			{Id: scannerTypeAxe},
			{Id: "lighthouse"},
		},
		Pages: []report.PageSummary{
			{
				Id:  "page-1",
				Url: "https://example.com/dashboard",
				PageOverview: &report.PageOverview{
					ScreenshotFilename: "page-overview-page-1.webp",
				},
			},
		},
		Issues: []report.IssueDetail{
			{Scanner: scannerTypeAxe, PageId: "page-1"},
		},
		Artifacts: []report.ReportArtifact{
			{
				Id:   "page-overview-lighthouse-page-1",
				Type: "page-overview",
				Path: &artifactPath,
			},
		},
	}

	screenshots := server.extractScreenshotsFromReport(context.Background(), "job-1", results, "")
	if got, want := len(screenshots), 1; got != want {
		t.Fatalf("want %d screenshot, got %d", want, got)
	}

	overview := screenshots[0]
	if got, want := overview.ScannerID, "lighthouse"; got != want {
		t.Fatalf("scanner ID: want %q, got %q", want, got)
	}

	if got, want := overview.ArtifactID, "page-overview:lighthouse:page-1"; got != want {
		t.Fatalf("artifact ID: want %q, got %q", want, got)
	}

	wantPath := "/jobs/job-1/files/lighthouse/page-1/screenshots/page-overview-page-1.webp"
	if !strings.Contains(overview.URL, wantPath) {
		t.Fatalf("overview URL %q does not contain %q", overview.URL, wantPath)
	}
}
