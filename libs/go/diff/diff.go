// Package diff computes issue-level diffs between two StageFlow scan reports.
package diff

import (
	"sort"

	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
)

// Result holds the diff between a baseline and current scan.
type Result struct {
	Schema   string               `json:"schema"`
	Baseline ScanMeta             `json:"baseline"`
	Current  ScanMeta             `json:"current"`
	Delta    Delta                `json:"delta"`
	New      []report.IssueDetail `json:"new"`
	Fixed    []report.IssueDetail `json:"fixed"`
	// SkippedScanners did not finish in one of the reports, so their issues
	// are left out of New, Fixed and Unchanged.
	SkippedScanners []string `json:"skippedScanners,omitempty"`
}

// ScanMeta identifies one side of the diff.
type ScanMeta struct {
	JobID       string `json:"jobId,omitempty"`
	Score       *int   `json:"score,omitempty"`
	TotalIssues int    `json:"totalIssues"`
}

// Delta holds the numeric changes between baseline and current.
type Delta struct {
	ScoreDelta      *int `json:"scoreDelta,omitempty"`
	NewIssues       int  `json:"newIssues"`
	FixedIssues     int  `json:"fixedIssues"`
	UnchangedIssues int  `json:"unchangedIssues"`
}

// IncompleteScanners returns, sorted, the scanners that did not finish: they
// failed, or the job expected them and no result came back. A report with any
// of them is partial. `partial` means the scanner ran and some pages failed,
// so it still counts as finished.
func IncompleteScanners(scanners []report.ScannerSummary) []string {
	ids := make([]string, 0)

	for _, scanner := range scanners {
		if scanner.Status == report.ScannerStatusFailed || scanner.Status == report.ScannerStatusSkipped {
			ids = append(ids, scanner.Id)
		}
	}

	sort.Strings(ids)

	return ids
}

// ComputeDiff compares two reports by stable issue ID and returns the diff.
// Issues present in current but not baseline are new.
// Issues present in baseline but not current are fixed.
// A scanner that did not finish in either report is compared in neither:
// otherwise its baseline findings would all look fixed, or its current ones
// new.
func ComputeDiff(
	baselineJobID string,
	baseline report.UnifiedReportV2,
	currentJobID string,
	current report.UnifiedReportV2,
) Result {
	skipped := make(map[string]struct{})
	for _, id := range IncompleteScanners(baseline.Scanners) {
		skipped[id] = struct{}{}
	}

	for _, id := range IncompleteScanners(current.Scanners) {
		skipped[id] = struct{}{}
	}

	baselineMap := make(map[string]report.IssueDetail, len(baseline.Issues))
	for _, issue := range baseline.Issues {
		if _, skip := skipped[issue.Scanner]; !skip {
			baselineMap[issue.Id] = issue
		}
	}

	currentMap := make(map[string]report.IssueDetail, len(current.Issues))
	for _, issue := range current.Issues {
		if _, skip := skipped[issue.Scanner]; !skip {
			currentMap[issue.Id] = issue
		}
	}

	newIssues := make([]report.IssueDetail, 0)
	fixedIssues := make([]report.IssueDetail, 0)

	var unchangedCount int

	for id, issue := range currentMap {
		if _, exists := baselineMap[id]; !exists {
			newIssues = append(newIssues, issue)
		} else {
			unchangedCount++
		}
	}

	for id, issue := range baselineMap {
		if _, exists := currentMap[id]; !exists {
			fixedIssues = append(fixedIssues, issue)
		}
	}

	// Sort for deterministic output.
	sort.Slice(newIssues, func(i, j int) bool { return newIssues[i].Id < newIssues[j].Id })
	sort.Slice(fixedIssues, func(i, j int) bool { return fixedIssues[i].Id < fixedIssues[j].Id })

	var scoreDelta *int

	// A score covers every scanner, so it cannot be compared across a skip.
	if len(skipped) == 0 && baseline.Summary.Score != nil && current.Summary.Score != nil {
		delta := *current.Summary.Score - *baseline.Summary.Score
		scoreDelta = &delta
	}

	return Result{
		Schema: "stageflow/diff@v1",
		Baseline: ScanMeta{
			JobID:       baselineJobID,
			Score:       baseline.Summary.Score,
			TotalIssues: baseline.Summary.TotalIssues,
		},
		Current: ScanMeta{
			JobID:       currentJobID,
			Score:       current.Summary.Score,
			TotalIssues: current.Summary.TotalIssues,
		},
		Delta: Delta{
			ScoreDelta:      scoreDelta,
			NewIssues:       len(newIssues),
			FixedIssues:     len(fixedIssues),
			UnchangedIssues: unchangedCount,
		},
		New:             newIssues,
		Fixed:           fixedIssues,
		SkippedScanners: sortedKeys(skipped),
	}
}

func sortedKeys(set map[string]struct{}) []string {
	if len(set) == 0 {
		return nil
	}

	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	return keys
}
