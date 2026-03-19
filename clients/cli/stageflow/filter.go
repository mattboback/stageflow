package main

import (
	"fmt"
	"strings"

	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
)

var validSeverities = map[report.IssueSeverity]int{
	report.IssueSeverityInfo:     0,
	report.IssueSeverityMinor:    1,
	report.IssueSeverityModerate: 2,
	report.IssueSeveritySerious:  3,
	report.IssueSeverityCritical: 4,
}

var validCategories = map[string]struct{}{
	"accessibility":  {},
	"performance":    {},
	"seo":            {},
	"security":       {},
	"best-practices": {},
}

func severityRank(value report.IssueSeverity) int {
	return validSeverities[value]
}

func severityFromString(s string) report.IssueSeverity {
	return report.IssueSeverity(strings.ToLower(strings.TrimSpace(s)))
}

func isValidSeverity(severity report.IssueSeverity) bool {
	_, ok := validSeverities[severity]

	return ok
}

func normalizeSeverities(values []string) ([]report.IssueSeverity, error) {
	if len(values) == 0 {
		return nil, nil
	}

	normalized := make([]report.IssueSeverity, 0, len(values))
	for _, raw := range values {
		severity := severityFromString(raw)
		if !isValidSeverity(severity) {
			return nil, fmt.Errorf(
				"invalid severity %q (expected critical, serious, moderate, minor, or info)",
				strings.TrimSpace(raw),
			)
		}

		normalized = append(normalized, severity)
	}

	return normalized, nil
}

func normalizeCategories(values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}

	normalized := make([]string, 0, len(values))
	for _, raw := range values {
		category := strings.ToLower(strings.TrimSpace(raw))
		if _, ok := validCategories[category]; !ok {
			return nil, fmt.Errorf(
				"invalid category %q (expected accessibility, performance, seo, security, or best-practices)",
				strings.TrimSpace(raw),
			)
		}

		normalized = append(normalized, category)
	}

	return normalized, nil
}

func filterBySeverity(issues []report.IssueDetail, severities []string) []report.IssueDetail {
	if len(severities) == 0 {
		return issues
	}

	normalized, err := normalizeSeverities(severities)
	if err != nil {
		return nil
	}

	allowed := make(map[report.IssueSeverity]bool, len(normalized))
	for _, severity := range normalized {
		allowed[severity] = true
	}

	filtered := make([]report.IssueDetail, 0, len(issues))
	for _, issue := range issues {
		if allowed[issue.Severity] {
			filtered = append(filtered, issue)
		}
	}

	return filtered
}

func filterByCategory(issues []report.IssueDetail, categories []string) []report.IssueDetail {
	if len(categories) == 0 {
		return issues
	}

	normalized, err := normalizeCategories(categories)
	if err != nil {
		return nil
	}

	allowed := make(map[string]bool, len(normalized))
	for _, category := range normalized {
		allowed[category] = true
	}

	filtered := make([]report.IssueDetail, 0, len(issues))
	for _, issue := range issues {
		cat := ""
		if issue.Category != nil {
			cat = strings.ToLower(*issue.Category)
		}

		if allowed[cat] {
			filtered = append(filtered, issue)
		}
	}

	return filtered
}

func hasIssuesAtOrAbove(issues []report.IssueDetail, threshold string) (bool, error) {
	normalized, err := normalizeSeverities([]string{threshold})
	if err != nil {
		return false, err
	}

	thresholdRank := severityRank(normalized[0])

	for _, issue := range issues {
		if severityRank(issue.Severity) >= thresholdRank {
			return true, nil
		}
	}

	return false, nil
}

func splitCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))

	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}
