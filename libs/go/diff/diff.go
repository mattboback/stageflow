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

// ComputeDiff compares two reports by stable issue ID and returns the diff.
// Issues present in current but not baseline are "new."
// Issues present in baseline but not current are "fixed."
func ComputeDiff(baselineJobID string, baseline report.UnifiedReportV2, currentJobID string, current report.UnifiedReportV2) Result {
	baselineMap := make(map[string]report.IssueDetail, len(baseline.Issues))
	for _, issue := range baseline.Issues {
		baselineMap[issue.Id] = issue
	}

	currentMap := make(map[string]report.IssueDetail, len(current.Issues))
	for _, issue := range current.Issues {
		currentMap[issue.Id] = issue
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
	if baseline.Summary.Score != nil && current.Summary.Score != nil {
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
		New:   newIssues,
		Fixed: fixedIssues,
	}
}
