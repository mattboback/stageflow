package main

import (
	"fmt"
	"io"
	"net/url"
	"sort"
	"strconv"
	"strings"

	report "github.com/mattboback/stageflow/packages/contracts/report/generated/go"
)

type markdownRenderOptions struct {
	MaxOccurrences int
	SummaryOnly    bool
	GroupBy        string
}

func writeMarkdownReport(
	out io.Writer,
	apiBaseURL string,
	status JobStatus,
	doc report.UnifiedReportV2,
	filters issueFilters,
	opts markdownRenderOptions,
) error {
	jobLink, err := buildAPILink(apiBaseURL, fmt.Sprintf("/api/v1/jobs/%s", url.PathEscape(status.ID)))
	if err != nil {
		return err
	}

	resultsLink, err := buildAPILink(apiBaseURL, fmt.Sprintf("/api/v1/jobs/%s/results", url.PathEscape(status.ID)))
	if err != nil {
		return err
	}

	sections := []func(io.Writer) error{
		func(w io.Writer) error {
			return writeMarkdownSummary(w, status, doc, filters, jobLink, resultsLink)
		},
		func(w io.Writer) error {
			return writeMarkdownLighthouseScores(w, doc.Summary.LighthouseCategories)
		},
		func(w io.Writer) error {
			return writeMarkdownScanners(w, doc.Scanners)
		},
		func(w io.Writer) error {
			return writeMarkdownPages(w, doc.Pages)
		},
	}

	if !opts.SummaryOnly {
		groupBy := opts.GroupBy
		if groupBy == "" {
			groupBy = "category"
		}

		sections = append(sections, func(w io.Writer) error {
			return writeMarkdownFindings(w, doc.Issues, opts.MaxOccurrences, groupBy)
		})
	}

	sections = append(sections,
		func(w io.Writer) error {
			return writeMarkdownArtifacts(w, doc.Artifacts)
		},
		func(w io.Writer) error {
			return writeMarkdownErrors(w, doc.Errors)
		},
	)

	for _, section := range sections {
		sectionErr := section(out)
		if sectionErr != nil {
			return sectionErr
		}
	}

	return nil
}

func writeMarkdownSummary(
	out io.Writer,
	status JobStatus,
	doc report.UnifiedReportV2,
	filters issueFilters,
	jobLink string,
	resultsLink string,
) error {
	if _, err := fmt.Fprintln(out, "## Scan Summary"); err != nil {
		return err
	}

	lines := []string{
		"- Job ID: " + status.ID,
		"- State: " + status.State,
		"- Job URL: " + jobLink,
		"- Results URL: " + resultsLink,
	}

	if urls := collectReportURLs(doc); len(urls) > 0 {
		lines = append(lines, "- URLs: "+strings.Join(urls, ", "))
	}

	if duration := formatDuration(doc); duration != "" {
		lines = append(lines, "- Duration: "+duration)
	}

	if score := formatScore(doc); score != "" {
		lines = append(lines, "- Score: "+score)
	}

	lines = append(
		lines,
		fmt.Sprintf("- Pages: %d scanned, %d with issues", doc.Summary.PagesScanned, doc.Summary.PagesWithIssues),
		fmt.Sprintf(
			"- Severity totals: critical=%d serious=%d moderate=%d minor=%d info=%d total=%d",
			doc.Summary.BySeverity.Critical,
			doc.Summary.BySeverity.Serious,
			doc.Summary.BySeverity.Moderate,
			doc.Summary.BySeverity.Minor,
			infoCount(doc.Summary.BySeverity),
			doc.Summary.TotalIssues,
		),
	)

	if filters.Truncated {
		lines = append(
			lines,
			fmt.Sprintf(
				"- Findings returned: %d (truncated from %d; max=%d)",
				filters.IssuesReturned,
				filters.IssuesTotal,
				filters.MaxIssues,
			),
		)
	} else {
		lines = append(
			lines,
			fmt.Sprintf(
				"- Findings returned: %d (max=%d)",
				filters.IssuesReturned,
				filters.MaxIssues,
			),
		)
	}

	if len(filters.Severities) > 0 {
		lines = append(lines, "- Severity filter: "+strings.Join(filters.Severities, ", "))
	}

	if len(filters.Categories) > 0 {
		lines = append(lines, "- Category filter: "+strings.Join(filters.Categories, ", "))
	}

	lines = append(lines, fmt.Sprintf("- Artifact count: %d", len(doc.Artifacts)))
	lines = append(lines, fmt.Sprintf("- Report error count: %d", len(doc.Errors)))

	for _, line := range lines {
		if _, err := fmt.Fprintln(out, line); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintln(out); err != nil {
		return err
	}

	return nil
}

func writeMarkdownLighthouseScores(out io.Writer, categories []report.LighthouseCategorySummary) error {
	if len(categories) == 0 {
		return nil
	}

	if _, err := fmt.Fprintln(out, "## Lighthouse Scores"); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(out, "| Category | Score |"); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(out, "| --- | --- |"); err != nil {
		return err
	}

	for _, cat := range categories {
		if _, err := fmt.Fprintf(out, "| %s | %.2f |\n", cat.Title, cat.AvgScore); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintln(out); err != nil {
		return err
	}

	return nil
}

func writeMarkdownScanners(out io.Writer, scanners []report.ScannerSummary) error {
	if _, err := fmt.Fprintln(out, "## Scanners"); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(
		out,
		"| ID | Name | Status | Issues | Duration | Results Path | Report Path |",
	); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(out, "| --- | --- | --- | --- | --- | --- | --- |"); err != nil {
		return err
	}

	if len(scanners) == 0 {
		if _, err := fmt.Fprintln(out, "| - | - | skipped | 0 | - | - | - |"); err != nil {
			return err
		}

		if _, err := fmt.Fprintln(out); err != nil {
			return err
		}

		return nil
	}

	for _, scanner := range scanners {
		issues := "-"
		if scanner.IssueCount != nil {
			issues = strconv.Itoa(*scanner.IssueCount)
		}

		duration := "-"
		if scanner.DurationMs != nil {
			duration = fmt.Sprintf("%.0fms", *scanner.DurationMs)
		}

		if _, err := fmt.Fprintf(
			out,
			"| %s | %s | %s | %s | %s | %s | %s |\n",
			scanner.Id,
			firstNonEmpty(stringValue(scanner.Name), scanner.Id),
			string(scanner.Status),
			issues,
			duration,
			markdownCell(stringValue(scanner.ResultsPath)),
			markdownCell(stringValue(scanner.ReportPath)),
		); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintln(out); err != nil {
		return err
	}

	return nil
}

func writeMarkdownPages(out io.Writer, pages []report.PageSummary) error {
	if _, err := fmt.Fprintln(out, "## Pages"); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(out, "| ID | URL | Path | Issues | Duration |"); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(out, "| --- | --- | --- | --- | --- |"); err != nil {
		return err
	}

	if len(pages) == 0 {
		if _, err := fmt.Fprintln(out, "| - | - | - | 0 | - |"); err != nil {
			return err
		}

		if _, err := fmt.Fprintln(out); err != nil {
			return err
		}

		return nil
	}

	for _, page := range pages {
		if _, err := fmt.Fprintf(
			out,
			"| %s | %s | %s | %d | %.0fms |\n",
			page.Id,
			markdownCell(page.Url),
			markdownCell(stringValue(page.Path)),
			page.IssueCount,
			page.DurationMs,
		); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintln(out); err != nil {
		return err
	}

	return nil
}

func writeMarkdownFindings(out io.Writer, issues []report.IssueDetail, maxOccurrences int, groupBy string) error {
	if _, err := fmt.Fprintln(out, "## Findings"); err != nil {
		return err
	}

	if len(issues) == 0 {
		if _, err := fmt.Fprintln(out, "No findings."); err != nil {
			return err
		}

		if _, err := fmt.Fprintln(out); err != nil {
			return err
		}

		return nil
	}

	switch groupBy {
	case "category":
		return writeMarkdownFindingsGrouped(out, issues, maxOccurrences, groupByCategory)
	case "scanner":
		return writeMarkdownFindingsGrouped(out, issues, maxOccurrences, groupByScanner)
	default:
		return writeMarkdownFindingsFlat(out, issues, maxOccurrences)
	}
}

func writeMarkdownFindingsFlat(out io.Writer, issues []report.IssueDetail, maxOccurrences int) error {
	for _, issue := range issues {
		if err := writeMarkdownFinding(out, issue, maxOccurrences); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintln(out); err != nil {
		return err
	}

	return nil
}

func groupByCategory(issue report.IssueDetail) string {
	if issue.Category != nil && *issue.Category != "" {
		return *issue.Category
	}

	return "uncategorized"
}

func groupByScanner(issue report.IssueDetail) string {
	return issue.Scanner
}

func writeMarkdownFindingsGrouped(
	out io.Writer,
	issues []report.IssueDetail,
	maxOccurrences int,
	keyFn func(report.IssueDetail) string,
) error {
	groups := make(map[string][]report.IssueDetail)

	var order []string

	for _, issue := range issues {
		key := keyFn(issue)
		if _, exists := groups[key]; !exists {
			order = append(order, key)
		}

		groups[key] = append(groups[key], issue)
	}

	sort.Strings(order)

	for _, key := range order {
		if _, err := fmt.Fprintf(out, "\n### %s\n\n", key); err != nil {
			return err
		}

		for _, issue := range groups[key] {
			if err := writeMarkdownFinding(out, issue, maxOccurrences); err != nil {
				return err
			}
		}
	}

	if _, err := fmt.Fprintln(out); err != nil {
		return err
	}

	return nil
}

func writeMarkdownFinding(out io.Writer, issue report.IssueDetail, maxOccurrences int) error {
	category := ""
	if issue.Category != nil && *issue.Category != "" {
		category = " | category=" + *issue.Category
	}

	if _, err := fmt.Fprintf(
		out,
		"- [%s] %s | scanner=%s | rule=%s | page=%s%s\n",
		issue.Severity,
		issue.Title,
		issue.Scanner,
		issue.RuleId,
		issue.PageUrl,
		category,
	); err != nil {
		return err
	}

	if issue.Description != "" {
		if _, err := fmt.Fprintf(out, "  Description: %s\n", issue.Description); err != nil {
			return err
		}
	}

	if issue.HowToFix != nil && *issue.HowToFix != "" {
		if _, err := fmt.Fprintf(out, "  How to fix: %s\n", *issue.HowToFix); err != nil {
			return err
		}
	}

	if len(issue.WcagTags) > 0 {
		if _, err := fmt.Fprintf(out, "  WCAG: %s\n", strings.Join(issue.WcagTags, ", ")); err != nil {
			return err
		}
	}

	if issue.HelpUrl != nil && *issue.HelpUrl != "" {
		if _, err := fmt.Fprintf(out, "  Help: %s\n", *issue.HelpUrl); err != nil {
			return err
		}
	}

	if err := writeMarkdownOccurrences(out, issue.Occurrences, maxOccurrences); err != nil {
		return err
	}

	return nil
}

func writeMarkdownOccurrences(out io.Writer, occurrences []report.IssueOccurrence, maxOccurrences int) error {
	if len(occurrences) == 0 {
		return nil
	}

	limit := maxOccurrences
	if limit <= 0 || limit > len(occurrences) {
		limit = len(occurrences)
	}

	if _, err := fmt.Fprintf(out, "  Occurrences (%d of %d):\n", limit, len(occurrences)); err != nil {
		return err
	}

	for i := range limit {
		occ := occurrences[i]
		line := formatOccurrenceLine(i+1, occ)

		if _, err := fmt.Fprintln(out, line); err != nil {
			return err
		}

		if fs := stringValue(occ.FailureSummary); fs != "" {
			if _, err := fmt.Fprintf(out, "       %s\n", fs); err != nil {
				return err
			}
		}
	}

	return nil
}

func writeMarkdownArtifacts(out io.Writer, artifacts []report.ReportArtifact) error {
	if _, err := fmt.Fprintln(out, "## Artifacts"); err != nil {
		return err
	}

	if len(artifacts) == 0 {
		if _, err := fmt.Fprintln(out, "No artifacts."); err != nil {
			return err
		}

		if _, err := fmt.Fprintln(out); err != nil {
			return err
		}

		return nil
	}

	if _, err := fmt.Fprintln(out, "| ID | Type | Path | MIME |"); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(out, "| --- | --- | --- | --- |"); err != nil {
		return err
	}

	for _, artifact := range artifacts {
		if _, err := fmt.Fprintf(
			out,
			"| %s | %s | %s | %s |\n",
			artifact.Id,
			artifact.Type,
			markdownCell(stringValue(artifact.Path)),
			markdownCell(stringValue(artifact.Mime)),
		); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintln(out); err != nil {
		return err
	}

	return nil
}

func writeMarkdownErrors(out io.Writer, errors []report.ReportError) error {
	if _, err := fmt.Fprintln(out, "## Report Errors"); err != nil {
		return err
	}

	if len(errors) == 0 {
		if _, err := fmt.Fprintln(out, "No report errors."); err != nil {
			return err
		}

		if _, err := fmt.Fprintln(out); err != nil {
			return err
		}

		return nil
	}

	for _, reportErr := range errors {
		scope := string(reportErr.Scope)
		scannerID := stringValue(reportErr.ScannerId)
		pageID := stringValue(reportErr.PageId)

		if _, err := fmt.Fprintf(
			out,
			"- %s | scope=%s | scanner=%s | page=%s | retryable=%t\n",
			reportErr.Message,
			scope,
			firstNonEmpty(scannerID, "-"),
			firstNonEmpty(pageID, "-"),
			reportErr.Retryable,
		); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintln(out); err != nil {
		return err
	}

	return nil
}

func markdownCell(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "-"
	}

	replacer := strings.NewReplacer("|", "\\|", "\n", "<br>")

	return replacer.Replace(trimmed)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}

	return ""
}
