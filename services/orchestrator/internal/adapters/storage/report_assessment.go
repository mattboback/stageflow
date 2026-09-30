package storage

import (
	"encoding/json"
	"regexp"
	"strings"

	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
)

func profilePointer(value string) *report.ScanProfile {
	profile := report.ScanProfileProduction
	if value == "staging" {
		profile = report.ScanProfileStaging
	}

	return &profile
}

func buildAssessment(
	profile string,
	issues []report.IssueDetail,
	summary report.ReportSummary,
	pages int,
) *report.ReportAssessment {
	result := &report.ReportAssessment{
		Profile:            *profilePointer(profile),
		PolicyVersion:      "indexing-v1",
		DispositionByIssue: make(map[string]report.FindingDisposition),
	}
	severity := report.SeverityCounts{}

	for _, issue := range issues {
		disposition := report.FindingDispositionActionable

		switch {
		case needsHumanReview(issue):
			disposition = report.FindingDispositionNeedsReview
			result.NeedsReview++
		case profile == "staging" && expectedIndexingRestriction(issue):
			disposition = report.FindingDispositionExpected
			result.Expected++
		default:
			result.Actionable++

			addSeverity(&severity, issue.Severity)
		}

		result.DispositionByIssue[issue.Id] = disposition
	}

	if summary.Score != nil {
		score, grade := calculateAccessibilityScore(severity, pages)
		result.Score = intPtr(score)
		result.ScoreGrade = stringPtr(grade)
	}

	return result
}

func expectedIndexingRestriction(issue report.IssueDetail) bool {
	if issue.Scanner != "lighthouse" || issue.RuleId != "is-crawlable" {
		return false
	}
	// Read directive sources only, never incidental URLs or labels.
	evidence, err := json.Marshal(issue.ScannerData["details"])
	if err != nil {
		return false
	}

	var details struct {
		Source json.RawMessage `json:"source"`
		Items  []struct {
			Source json.RawMessage `json:"source"`
		} `json:"items"`
	}
	if json.Unmarshal(evidence, &details) != nil {
		return false
	}

	if confirmedNoindexSource(details.Source) {
		return true
	}

	for _, item := range details.Items {
		if confirmedNoindexSource(item.Source) {
			return true
		}
	}

	return false
}

var noindexDirective = regexp.MustCompile(`(?i)(?:^|[\s,:;])noindex(?:$|[\s,;])`)
var robotsMetaName = regexp.MustCompile(`(?i)\bname\s*=\s*["'](?:robots|googlebot|bingbot)["']`)
var robotsMetaContent = regexp.MustCompile(`(?i)\bcontent\s*=\s*["']([^"']*)["']`)

func confirmedNoindexSource(source json.RawMessage) bool {
	var header string
	if json.Unmarshal(source, &header) == nil {
		return strings.HasPrefix(strings.ToLower(strings.TrimSpace(header)), "x-robots-tag:") &&
			noindexDirective.MatchString(header)
	}

	var node struct {
		Snippet string `json:"snippet"`
	}
	if json.Unmarshal(source, &node) != nil ||
		!strings.HasPrefix(strings.ToLower(strings.TrimSpace(node.Snippet)), "<meta") ||
		!robotsMetaName.MatchString(node.Snippet) {
		return false
	}

	content := robotsMetaContent.FindStringSubmatch(node.Snippet)

	return len(content) == 2 && noindexDirective.MatchString(content[1])
}
