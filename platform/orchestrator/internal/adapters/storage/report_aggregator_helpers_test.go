package storage

import (
	"testing"
	"time"

	report "github.com/mattboback/stageflow/packages/contracts/report/generated/go"
)

func TestPageKey(t *testing.T) {
	t.Parallel()

	if got, want := pageKey("axe", 0, report.PageSummary{Id: "page-1"}), "id:page-1"; got != want {
		t.Fatalf("pageKey(PageID): want %q, got %q", want, got)
	}

	if got, want := pageKey(
		"axe",
		0,
		report.PageSummary{Url: "https://example.com/#frag"},
	), "url:https://example.com/"; got != want {
		t.Fatalf("pageKey(URL): want %q, got %q", want, got)
	}

	if got, want := pageKey("axe", 0, report.PageSummary{Path: stringPtr("/about")}), "path:/about"; got != want {
		t.Fatalf("pageKey(Path): want %q, got %q", want, got)
	}

	if got, want := pageKey("axe", 5, report.PageSummary{}), "scan:axe:5"; got != want {
		t.Fatalf("pageKey(fallback): want %q, got %q", want, got)
	}
}

func TestNormalizeURL(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "lowercases_host_and_strips_fragment",
			in:   "https://EXAMPLE.com/About/#Section",
			want: "https://example.com/About",
		},
		{
			name: "keeps_root_slash",
			in:   "https://example.com/#x",
			want: "https://example.com/",
		},
		{
			name: "trims_trailing_slash_and_preserves_query",
			in:   "  https://example.com/foo/?a=1#x  ",
			want: "https://example.com/foo?a=1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := normalizeURL(tc.in)
			if got != tc.want {
				t.Fatalf("normalizeURL(%q): want %q, got %q", tc.in, tc.want, got)
			}
		})
	}
}

func TestDerivePageID(t *testing.T) {
	t.Parallel()

	if got, want := derivePageID(
		report.PageSummary{Url: "https://example.com/about/"},
	), "url-example.com-about"; got != want {
		t.Fatalf("derivePageID(url): want %q, got %q", want, got)
	}

	if got := derivePageID(report.PageSummary{}); got != "" {
		t.Fatalf("derivePageID(empty): want empty, got %q", got)
	}
}

func TestPickEarlierPickLater(t *testing.T) {
	t.Parallel()

	older := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)

	if got, want := pickEarlierTime(&newer, &older), &older; !got.Equal(*want) {
		t.Fatalf("pickEarlierTime: want %v, got %v", want, got)
	}

	if got, want := pickLaterTime(&older, &newer), &newer; !got.Equal(*want) {
		t.Fatalf("pickLaterTime: want %v, got %v", want, got)
	}
}

func TestMergeAggregatedPagePrefersAxeOverview(t *testing.T) {
	t.Parallel()

	pagesByKey := make(map[string]*aggregatedPage)

	mergeAggregatedPage(pagesByKey, "lighthouse", 0, report.PageSummary{
		Id:  "page-1",
		Url: "https://example.com/",
		PageOverview: &report.PageOverview{
			ScreenshotFilename: "lh.png",
			PageWidth:          800,
			PageHeight:         600,
		},
	})

	mergeAggregatedPage(pagesByKey, "axe", 0, report.PageSummary{
		Id:  "page-1",
		Url: "https://example.com/",
		PageOverview: &report.PageOverview{
			ScreenshotFilename: "axe.png",
			PageWidth:          800,
			PageHeight:         600,
		},
	})

	mergeAggregatedPage(pagesByKey, "lighthouse", 0, report.PageSummary{
		Id:  "page-1",
		Url: "https://example.com/",
		PageOverview: &report.PageOverview{
			ScreenshotFilename: "lh2.png",
			PageWidth:          800,
			PageHeight:         600,
		},
	})

	agg := pagesByKey["id:page-1"]
	if agg == nil || agg.page.PageOverview == nil {
		t.Fatalf("expected aggregated page with overview")
	}

	if got, want := agg.page.PageOverview.ScreenshotFilename, "axe.png"; got != want {
		t.Fatalf("expected axe overview %q, got %q", want, got)
	}
}

func TestMergeAggregatedPageAggregatesCountsAndTiming(t *testing.T) {
	t.Parallel()

	pagesByKey := make(map[string]*aggregatedPage)
	start1 := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	end1 := time.Date(2020, 1, 1, 0, 0, 10, 0, time.UTC)
	start2 := time.Date(2019, 12, 31, 23, 59, 0, 0, time.UTC)
	end2 := time.Date(2020, 1, 1, 0, 0, 20, 0, time.UTC)

	mergeAggregatedPage(pagesByKey, "axe", 0, report.PageSummary{
		Id:         "page-1",
		Url:        "",
		Path:       stringPtr("/"),
		IssueCount: 1,
		BySeverity: &report.SeverityCounts{Critical: 1},
		DurationMs: 100,
		StartedAt:  &start1,
		FinishedAt: &end1,
	})

	mergeAggregatedPage(pagesByKey, "lighthouse", 0, report.PageSummary{
		Id:         "page-1",
		Url:        "https://example.com/",
		Path:       nil,
		IssueCount: 2,
		BySeverity: &report.SeverityCounts{Minor: 2},
		DurationMs: 50,
		StartedAt:  &start2,
		FinishedAt: &end2,
	})

	agg := pagesByKey["id:page-1"]
	if agg == nil {
		t.Fatalf("expected aggregated page")
	}

	if got, want := agg.page.Url, "https://example.com/"; got != want {
		t.Fatalf("URL: want %q, got %q", want, got)
	}

	if got, want := stringValue(agg.page.Path), "/"; got != want {
		t.Fatalf("Path: want %q, got %q", want, got)
	}

	if got, want := agg.page.IssueCount, 3; got != want {
		t.Fatalf("IssueCount: want %d, got %d", want, got)
	}

	if agg.page.BySeverity == nil {
		t.Fatalf("expected BySeverity to be set")
	}

	if got, want := agg.page.BySeverity.Critical, 1; got != want {
		t.Fatalf("Critical: want %d, got %d", want, got)
	}

	if got, want := agg.page.BySeverity.Minor, 2; got != want {
		t.Fatalf("Minor: want %d, got %d", want, got)
	}

	if got, want := agg.page.DurationMs, 150.0; got != want {
		t.Fatalf("DurationMs: want %v, got %v", want, got)
	}

	if agg.page.StartedAt == nil || !agg.page.StartedAt.Equal(start2) {
		t.Fatalf("StartedAt: want %v, got %v", start2, agg.page.StartedAt)
	}

	if agg.page.FinishedAt == nil || !agg.page.FinishedAt.Equal(end2) {
		t.Fatalf("FinishedAt: want %v, got %v", end2, agg.page.FinishedAt)
	}
}
