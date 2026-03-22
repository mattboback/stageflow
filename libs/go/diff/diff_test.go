package diff

import (
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
