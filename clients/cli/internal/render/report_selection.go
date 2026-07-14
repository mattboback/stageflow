package render

import (
	"slices"
	"strings"

	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
)

const issueSortOrder = "severity_desc, scanner_asc, rule_id_asc, page_url_asc, id_asc"

func ValidatedIssueSelection(
	issues []report.IssueDetail,
	opts Options,
) ([]report.IssueDetail, IssueFilters, error) {
	if _, err := normalizeSeverities(opts.Severities); err != nil {
		return nil, IssueFilters{}, err
	}
	if _, err := normalizeCategories(opts.Categories); err != nil {
		return nil, IssueFilters{}, err
	}
	if opts.FailSeverity != "" {
		if _, err := normalizeSeverities([]string{opts.FailSeverity}); err != nil {
			return nil, IssueFilters{}, err
		}
	}

	filtered := issues
	if len(opts.Severities) > 0 {
		filtered = filterBySeverity(filtered, opts.Severities)
	}
	if len(opts.Categories) > 0 {
		filtered = filterByCategory(filtered, opts.Categories)
	}

	selectedIssues, filters := selectIssues(filtered, opts.MaxIssues)
	filters.Severities = opts.Severities
	filters.Categories = opts.Categories

	return selectedIssues, filters, nil
}

func ShouldFailForSeverity(issues []report.IssueDetail, threshold string) (bool, error) {
	if threshold == "" {
		return false, nil
	}

	return HasIssuesAtOrAbove(issues, threshold)
}

func selectIssues(issues []report.IssueDetail, maxIssues int) ([]report.IssueDetail, IssueFilters) {
	total := len(issues)
	if total == 0 {
		return []report.IssueDetail{}, IssueFilters{
			MaxIssues: maxIssues,
			Sort:      issueSortOrder,
		}
	}

	sorted := append([]report.IssueDetail(nil), issues...)
	slices.SortFunc(sorted, func(a, b report.IssueDetail) int {
		if as, bs := severityRank(a.Severity), severityRank(b.Severity); as != bs {
			return bs - as
		}
		if c := strings.Compare(a.Scanner, b.Scanner); c != 0 {
			return c
		}
		if c := strings.Compare(a.RuleId, b.RuleId); c != 0 {
			return c
		}
		if c := strings.Compare(a.PageUrl, b.PageUrl); c != 0 {
			return c
		}
		return strings.Compare(a.Id, b.Id)
	})

	limit := maxIssues
	if limit <= 0 || limit > total {
		limit = total
	}

	selected := sorted[:limit]
	return selected, IssueFilters{
		MaxIssues:      maxIssues,
		IssuesReturned: len(selected),
		IssuesTotal:    total,
		Truncated:      len(selected) != total,
		Sort:           issueSortOrder,
	}
}
