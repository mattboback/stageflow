// Package diff computes issue-level diffs between two StageFlow scan reports.
package diff

import (
	"sort"
	"strings"

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
	SkippedScanners      []string `json:"skippedScanners,omitempty"`
	IncompatibleIdentity bool     `json:"incompatibleIdentity,omitempty"`
	NotComparedPages     []string `json:"notComparedPages,omitempty"`
	ContextLimited       bool     `json:"contextLimited,omitempty"`
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
// failed, skipped, or completed only part of their page coverage.
func IncompleteScanners(scanners []report.ScannerSummary) []string {
	ids := make([]string, 0)

	for _, scanner := range scanners {
		if scanner.Status == report.ScannerStatusFailed || scanner.Status == report.ScannerStatusSkipped ||
			scanner.Status == report.ScannerStatusPartial {
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
	skipped := comparisonExclusions(baseline, current)
	contextualScannerExclusions(baseline, current, skipped)
	common, notComparedPages := commonPages(baseline, current)
	baselineMap := comparedFindings(baseline.Issues, skipped)

	currentMap := comparedFindings(current.Issues, skipped)
	for id, issue := range baselineMap {
		if !common[issue.PageId] || !evaluatedOnBoth(baseline, current, issue) {
			delete(baselineMap, id)
		}
	}

	for id, issue := range currentMap {
		if !common[issue.PageId] || !evaluatedOnBoth(baseline, current, issue) {
			delete(currentMap, id)
		}
	}

	incompatible := hasStableIdentity(baseline) != hasStableIdentity(current)
	if incompatible {
		baselineMap = map[string]report.IssueDetail{}
		currentMap = map[string]report.IssueDetail{}
	}

	newIssues, fixedIssues, unchangedCount := compareFindings(baselineMap, currentMap)

	// Sort for deterministic output.
	sort.Slice(newIssues, func(i, j int) bool { return findingSortKey(newIssues[i]) < findingSortKey(newIssues[j]) })
	sort.Slice(
		fixedIssues,
		func(i, j int) bool { return findingSortKey(fixedIssues[i]) < findingSortKey(fixedIssues[j]) },
	)

	baselineScore := baseline.Summary.Score
	currentScore := current.Summary.Score

	if incompatible || !scoresComparable(baseline, current, skipped, notComparedPages) {
		baselineScore = nil
		currentScore = nil
	}

	var scoreDelta *int

	// A score covers every scanner, so it cannot be compared across a skip.
	if baselineScore != nil && currentScore != nil {
		delta := *current.Summary.Score - *baseline.Summary.Score
		scoreDelta = &delta
	}

	return Result{
		Schema: "stageflow/diff@v1",
		Baseline: ScanMeta{
			JobID:       baselineJobID,
			Score:       baselineScore,
			TotalIssues: baseline.Summary.TotalIssues,
		},
		Current: ScanMeta{
			JobID:       currentJobID,
			Score:       currentScore,
			TotalIssues: current.Summary.TotalIssues,
		},
		Delta: Delta{
			ScoreDelta:      scoreDelta,
			NewIssues:       len(newIssues),
			FixedIssues:     len(fixedIssues),
			UnchangedIssues: unchangedCount,
		},
		New:                  newIssues,
		Fixed:                fixedIssues,
		SkippedScanners:      sortedKeys(skipped),
		IncompatibleIdentity: incompatible,
		NotComparedPages:     notComparedPages,
		ContextLimited:       baseline.Meta.ScanContext == nil || current.Meta.ScanContext == nil,
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

// Preserve the issue ID for report navigation while comparing every occurrence.
// Legacy reports retain their issue-level comparison until a fresh baseline.
func occurrenceFindings(issue report.IssueDetail) map[string]report.IssueDetail {
	findings := make(map[string]report.IssueDetail)
	if !strings.HasPrefix(issue.PageId, "page-v2-") || len(issue.Occurrences) == 0 {
		findings[issue.Id] = issue

		return findings
	}

	for _, occurrence := range issue.Occurrences {
		finding := issue
		finding.Occurrences = []report.IssueOccurrence{occurrence}
		finding.ElementCount = 1
		finding.ElementsTruncated = nil

		if occurrence.ElementId == nil {
			findings[issue.Id] = issue
			continue
		}

		findings[*occurrence.ElementId] = finding
	}

	return findings
}

func hasStableIdentity(r report.UnifiedReportV2) bool {
	for _, page := range r.Pages {
		if strings.HasPrefix(page.Id, "page-v2-") {
			return true
		}
	}

	for _, issue := range r.Issues {
		if strings.HasPrefix(issue.PageId, "page-v2-") {
			return true
		}
	}

	return false
}

func findingSortKey(issue report.IssueDetail) string {
	key := issue.Id
	if len(issue.Occurrences) == 1 && issue.Occurrences[0].ElementId != nil {
		key += ":" + *issue.Occurrences[0].ElementId
	}

	return key
}

func comparisonExclusions(baseline, current report.UnifiedReportV2) map[string]struct{} {
	skipped := make(map[string]struct{})
	presence := make(map[string]int)

	for side, doc := range []report.UnifiedReportV2{baseline, current} {
		scanners := doc.Scanners
		for _, scanner := range scanners {
			if scanner.Status == report.ScannerStatusFailed || scanner.Status == report.ScannerStatusSkipped ||
				(scanner.Status == report.ScannerStatusPartial && (doc.Meta.ScanContext == nil || len(scanner.Pages) == 0)) {
				skipped[scanner.Id] = struct{}{}
			}
		}

		for _, scanner := range scanners {
			presence[scanner.Id] |= 1 << side
		}
	}
	// A scanner omitted from either side has no comparable coverage.
	for id, sides := range presence {
		if sides != 3 {
			skipped[id] = struct{}{}
		}
	}

	return skipped
}

func comparedFindings(issues []report.IssueDetail, skipped map[string]struct{}) map[string]report.IssueDetail {
	findings := make(map[string]report.IssueDetail, len(issues))
	for _, issue := range issues {
		if _, skip := skipped[issue.Scanner]; skip {
			continue
		}

		for key, finding := range occurrenceFindings(issue) {
			findings[key] = finding
		}
	}

	return findings
}

func compareFindings(
	baseline, current map[string]report.IssueDetail,
) ([]report.IssueDetail, []report.IssueDetail, int) {
	newIssues := make([]report.IssueDetail, 0)
	fixedIssues := make([]report.IssueDetail, 0)
	unchanged := 0

	for id, issue := range current {
		if _, exists := baseline[id]; exists {
			unchanged++
		} else {
			newIssues = append(newIssues, issue)
		}
	}

	for id, issue := range baseline {
		if _, exists := current[id]; !exists {
			fixedIssues = append(fixedIssues, issue)
		}
	}

	return newIssues, fixedIssues, unchanged
}
