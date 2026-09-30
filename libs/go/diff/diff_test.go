package diff

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
)

func intPtr(v int) *int { return &v }

func TestComputeDiff_NewAndFixed(t *testing.T) {
	baseline := report.UnifiedReportV2{
		Summary: report.ReportSummary{Score: intPtr(90), TotalIssues: 3},
		Issues: []report.IssueDetail{
			{Id: "a", Title: "Issue A", Severity: report.IssueSeverityCritical},
			{Id: "b", Title: "Issue B", Severity: report.IssueSeveritySerious},
			{Id: "c", Title: "Issue C", Severity: report.IssueSeverityMinor},
		},
	}

	current := report.UnifiedReportV2{
		Summary: report.ReportSummary{Score: intPtr(85), TotalIssues: 3},
		Issues: []report.IssueDetail{
			{Id: "a", Title: "Issue A", Severity: report.IssueSeverityCritical},
			{Id: "d", Title: "Issue D", Severity: report.IssueSeverityModerate},
			{Id: "e", Title: "Issue E", Severity: report.IssueSeverityMinor},
		},
	}

	result := ComputeDiff("baseline-job", baseline, "current-job", current)

	if result.Schema != "stageflow/diff@v1" {
		t.Fatalf("schema: got %q", result.Schema)
	}

	if result.Delta.NewIssues != 2 {
		t.Fatalf("new issues: got %d, want 2", result.Delta.NewIssues)
	}

	if result.Delta.FixedIssues != 2 {
		t.Fatalf("fixed issues: got %d, want 2", result.Delta.FixedIssues)
	}

	if result.Delta.UnchangedIssues != 1 {
		t.Fatalf("unchanged: got %d, want 1", result.Delta.UnchangedIssues)
	}

	if result.Delta.ScoreDelta == nil || *result.Delta.ScoreDelta != -5 {
		t.Fatalf("score delta: got %v, want -5", result.Delta.ScoreDelta)
	}

	if result.Baseline.JobID != "baseline-job" {
		t.Fatalf("baseline job: got %q", result.Baseline.JobID)
	}

	if result.Current.JobID != "current-job" {
		t.Fatalf("current job: got %q", result.Current.JobID)
	}

	// New issues should be sorted by ID.
	if len(result.New) != 2 || result.New[0].Id != "d" || result.New[1].Id != "e" {
		t.Fatalf("new issue IDs: got %v", issueIDs(result.New))
	}

	// Fixed issues should be sorted by ID.
	if len(result.Fixed) != 2 || result.Fixed[0].Id != "b" || result.Fixed[1].Id != "c" {
		t.Fatalf("fixed issue IDs: got %v", issueIDs(result.Fixed))
	}
}

func TestComputeDiff_NoChanges(t *testing.T) {
	issues := []report.IssueDetail{
		{Id: "a"},
		{Id: "b"},
	}

	r := report.UnifiedReportV2{
		Summary: report.ReportSummary{Score: intPtr(100), TotalIssues: 2},
		Issues:  issues,
	}

	result := ComputeDiff("b", r, "c", r)

	if result.Delta.NewIssues != 0 {
		t.Fatalf("new: got %d", result.Delta.NewIssues)
	}

	if result.Delta.FixedIssues != 0 {
		t.Fatalf("fixed: got %d", result.Delta.FixedIssues)
	}

	if result.Delta.UnchangedIssues != 2 {
		t.Fatalf("unchanged: got %d", result.Delta.UnchangedIssues)
	}

	if result.Delta.ScoreDelta == nil || *result.Delta.ScoreDelta != 0 {
		t.Fatalf("score delta: got %v, want 0", result.Delta.ScoreDelta)
	}
}

func TestComputeDiff_EmptyBaseline(t *testing.T) {
	baseline := report.UnifiedReportV2{
		Summary: report.ReportSummary{TotalIssues: 0},
	}

	current := report.UnifiedReportV2{
		Summary: report.ReportSummary{TotalIssues: 2},
		Issues: []report.IssueDetail{
			{Id: "a"},
			{Id: "b"},
		},
	}

	result := ComputeDiff("", baseline, "j1", current)

	if result.Delta.NewIssues != 2 {
		t.Fatalf("new: got %d", result.Delta.NewIssues)
	}

	if result.Delta.FixedIssues != 0 {
		t.Fatalf("fixed: got %d", result.Delta.FixedIssues)
	}

	if result.Delta.ScoreDelta != nil {
		t.Fatalf("score delta should be nil when baseline has no score")
	}
}

func TestComputeDiff_EmptyCurrent(t *testing.T) {
	baseline := report.UnifiedReportV2{
		Summary: report.ReportSummary{Score: intPtr(50), TotalIssues: 2},
		Issues: []report.IssueDetail{
			{Id: "a"},
			{Id: "b"},
		},
	}

	current := report.UnifiedReportV2{
		Summary: report.ReportSummary{Score: intPtr(100), TotalIssues: 0},
	}

	result := ComputeDiff("b", baseline, "c", current)

	if result.Delta.NewIssues != 0 {
		t.Fatalf("new: got %d", result.Delta.NewIssues)
	}

	if result.Delta.FixedIssues != 2 {
		t.Fatalf("fixed: got %d", result.Delta.FixedIssues)
	}

	if result.Delta.ScoreDelta == nil || *result.Delta.ScoreDelta != 50 {
		t.Fatalf("score delta: got %v, want 50", result.Delta.ScoreDelta)
	}
}

func issueIDs(issues []report.IssueDetail) []string {
	ids := make([]string, len(issues))
	for i, issue := range issues {
		ids[i] = issue.Id
	}

	return ids
}

func TestComputeDiff_SkipsScannersThatDidNotFinish(t *testing.T) {
	baseline := report.UnifiedReportV2{
		Summary: report.ReportSummary{Score: intPtr(80), TotalIssues: 3},
		Scanners: []report.ScannerSummary{
			{Id: "axe", Status: report.ScannerStatusSuccess},
			{Id: "seo", Status: report.ScannerStatusSuccess},
			{Id: "lighthouse", Status: report.ScannerStatusSkipped},
		},
		Issues: []report.IssueDetail{
			{Id: "axe-1", Scanner: "axe"},
			{Id: "axe-2", Scanner: "axe"},
			{Id: "seo-1", Scanner: "seo"},
		},
	}

	current := report.UnifiedReportV2{
		Summary: report.ReportSummary{Score: intPtr(95), TotalIssues: 2},
		Scanners: []report.ScannerSummary{
			{Id: "axe", Status: report.ScannerStatusFailed},
			{Id: "seo", Status: report.ScannerStatusSuccess},
			{Id: "lighthouse", Status: report.ScannerStatusSuccess},
		},
		Issues: []report.IssueDetail{
			{Id: "seo-2", Scanner: "seo"},
			{Id: "lh-1", Scanner: "lighthouse"},
		},
	}

	result := ComputeDiff("baseline-job", baseline, "current-job", current)

	if got := result.SkippedScanners; len(got) != 2 || got[0] != "axe" || got[1] != "lighthouse" {
		t.Fatalf("skipped scanners: got %v, want [axe lighthouse]", got)
	}

	if len(result.Fixed) != 1 || result.Fixed[0].Id != "seo-1" {
		t.Fatalf("fixed: got %v, want only seo-1 (axe did not run in current)", issueIDs(result.Fixed))
	}

	if len(result.New) != 1 || result.New[0].Id != "seo-2" {
		t.Fatalf("new: got %v, want only seo-2 (lighthouse did not run in baseline)", issueIDs(result.New))
	}

	if result.Baseline.Score != nil || result.Current.Score != nil {
		t.Fatal("incomplete comparison exposed scores that cannot be compared")
	}

	if result.Delta.ScoreDelta != nil {
		t.Fatalf("score delta: got %d, want none across a skipped scanner", *result.Delta.ScoreDelta)
	}
}

func TestComputeDiff_PartialScannerIsExcluded(t *testing.T) {
	scanners := []report.ScannerSummary{{Id: "axe", Status: report.ScannerStatusPartial}}
	baseline := report.UnifiedReportV2{Scanners: scanners, Issues: []report.IssueDetail{{Id: "a", Scanner: "axe"}}}
	current := report.UnifiedReportV2{Scanners: scanners}

	result := ComputeDiff("", baseline, "", current)

	if len(result.SkippedScanners) != 1 || result.Delta.FixedIssues != 0 {
		t.Fatalf("skipped=%v fixed=%d, want incomplete page coverage excluded",
			result.SkippedScanners, result.Delta.FixedIssues)
	}
}

// Mirrors the golden-pair case in clients/web/app/lib/projects.test.ts.
func TestComputeDiff_GoldenPairWithAxeFailed(t *testing.T) {
	load := func(name string) report.UnifiedReportV2 {
		t.Helper()

		data, err := os.ReadFile(filepath.Join("..", "..", "..", "qa", "fixtures", "scan-diff-golden", name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}

		var envelope struct {
			Report report.UnifiedReportV2 `json:"report"`
		}
		if decodeErr := json.Unmarshal(data, &envelope); decodeErr != nil {
			t.Fatalf("decode %s: %v", name, decodeErr)
		}

		return envelope.Report
	}

	baseline := load("golden-baseline-report.json")
	current := load("golden-regression-report.json")

	for i := range current.Scanners {
		if current.Scanners[i].Id == "axe" {
			current.Scanners[i].Status = report.ScannerStatusFailed
		}
	}

	result := ComputeDiff("baseline-job", baseline, "current-job", current)

	if len(result.SkippedScanners) != 1 || result.SkippedScanners[0] != "axe" {
		t.Fatalf("skipped scanners: got %v, want [axe]", result.SkippedScanners)
	}

	if result.Delta.NewIssues != 0 || result.Delta.FixedIssues != 0 || result.Delta.UnchangedIssues != 0 {
		t.Fatalf("delta: got %+v, want nothing compared", result.Delta)
	}

	if result.Delta.ScoreDelta != nil {
		t.Fatalf("score delta: got %d, want none", *result.Delta.ScoreDelta)
	}
}

func TestComputeDiff_SixthOccurrenceIsOneNew(t *testing.T) {
	issue := report.IssueDetail{Id: "images", PageId: "page-v2-root", Scanner: "axe", RuleId: "image-alt"}
	for i := range 5 {
		issue.Occurrences = append(
			issue.Occurrences,
			report.IssueOccurrence{ElementId: strPtr(fmt.Sprintf("occ-v2-%d", i))},
		)
	}

	baseline := report.UnifiedReportV2{Issues: []report.IssueDetail{issue}}
	issue.Occurrences = append(
		append([]report.IssueOccurrence{}, issue.Occurrences...),
		report.IssueOccurrence{ElementId: strPtr("occ-v2-5")},
	)
	current := report.UnifiedReportV2{Issues: []report.IssueDetail{issue}}

	result := ComputeDiff("b", baseline, "c", current)
	if result.Delta.NewIssues != 1 || result.Delta.UnchangedIssues != 5 || result.Delta.FixedIssues != 0 {
		t.Fatalf("unexpected delta: %+v", result.Delta)
	}

	if len(result.New) != 1 || *result.New[0].Occurrences[0].ElementId != "occ-v2-5" {
		t.Fatalf("new occurrence: %+v", result.New)
	}
}

func TestComputeDiff_IncompatibleIdentityNeedsFreshBaseline(t *testing.T) {
	baseline := report.UnifiedReportV2{Pages: []report.PageSummary{{Id: "url-1"}}}
	current := report.UnifiedReportV2{Pages: []report.PageSummary{{Id: "page-v2-root"}}}

	result := ComputeDiff("", baseline, "", current)
	if !result.IncompatibleIdentity || result.Delta.ScoreDelta != nil {
		t.Fatalf("expected identity incompatibility: %+v", result)
	}
}

func strPtr(value string) *string { return &value }
