package main

import (
	"fmt"
	"strings"

	report "github.com/mattboback/stageflow/packages/contracts/report/generated/go"
)

type severityLevel int

const (
	severityInfo severityLevel = iota
	severityMinor
	severityModerate
	severitySerious
	severityCritical
)

func parseMinimumSeverity(raw string) (severityLevel, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "info":
		return severityInfo, nil
	case "minor":
		return severityMinor, nil
	case "moderate":
		return severityModerate, nil
	case "serious":
		return severitySerious, nil
	case "critical":
		return severityCritical, nil
	default:
		return severityInfo, fmt.Errorf("invalid severity %q (expected critical, serious, moderate, minor, info)", raw)
	}
}

func severityLevelFor(value report.IssueSeverity) severityLevel {
	switch value {
	case report.IssueSeverityCritical:
		return severityCritical
	case report.IssueSeveritySerious:
		return severitySerious
	case report.IssueSeverityModerate:
		return severityModerate
	case report.IssueSeverityMinor:
		return severityMinor
	case report.IssueSeverityInfo:
		return severityInfo
	default:
		return severityInfo
	}
}

func filterIssues(issues []report.IssueDetail, minimum severityLevel, maxItems int) []report.IssueDetail {
	if len(issues) == 0 {
		return nil
	}

	limit := maxItems
	if limit <= 0 {
		limit = len(issues)
	}

	filtered := make([]report.IssueDetail, 0, minInt(limit, len(issues)))
	for _, issue := range issues {
		if severityLevelFor(issue.Severity) < minimum {
			continue
		}

		filtered = append(filtered, issue)
		if len(filtered) == limit {
			break
		}
	}

	return filtered
}

func minInt(a, b int) int {
	if a < b {
		return a
	}

	return b
}
