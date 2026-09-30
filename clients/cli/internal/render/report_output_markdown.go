package render

import (
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
	"github.com/mattboback/stageflow/libs/go/diff"
)

type markdownRenderOptions struct{ SummaryOnly bool }

// Markdown is a CI digest. Detailed evidence remains in the full report.
func writeMarkdownReport(
	out io.Writer,
	apiBaseURL string,
	status apiclient.JobStatus,
	doc report.UnifiedReportV2,
	filters IssueFilters,
	opts markdownRenderOptions,
) error {
	link, err := buildAPILink(apiBaseURL, "/scan/"+url.PathEscape(status.ID)+"/report")
	if err != nil {
		return err
	}

	if _, err = fmt.Fprint(out, "## Scan Summary\n\n"); err != nil {
		return err
	}

	if err = writeMarkdownScanTotals(out, doc, filters); err != nil {
		return err
	}

	if !opts.SummaryOnly && len(doc.Issues) > 0 {
		if err = writeMarkdownRuleGroups(out, doc.Issues); err != nil {
			return err
		}
	}

	checks := make([]string, 0, len(doc.ManualChecks))
	for _, check := range doc.ManualChecks {
		checks = append(checks, check.Title)
	}

	if err = writeMarkdownNotes(out, "Manual checks", checks); err != nil {
		return err
	}

	warnings := make([]string, 0, len(doc.Errors))
	for _, warning := range doc.Errors {
		warnings = append(warnings, warning.Message)
	}

	if err = writeMarkdownNotes(out, "Scan warnings", warnings); err != nil {
		return err
	}

	_, err = fmt.Fprintf(out, "[View full report (24h)](%s)\n", link)

	return err
}

func writeMarkdownScanTotals(out io.Writer, doc report.UnifiedReportV2, filters IssueFilters) error {
	if missing := diff.IncompleteScanners(doc.Scanners); len(missing) > 0 {
		if _, err := fmt.Fprintf(
			out,
			"**Partial report** — not fully scanned: %s. No site score is given.\n\n",
			strings.Join(missing, ", "),
		); err != nil {
			return err
		}
	} else if score := formatScore(doc); score != "" {
		if _, err := fmt.Fprintf(
			out,
			"**Score:** %s · %d pages scanned\n\n",
			score,
			doc.Summary.PagesScanned,
		); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintln(
		out,
		"| Critical | Serious | Moderate | Minor | Info |\n| --- | --- | --- | --- | --- |",
	); err != nil {
		return err
	}

	counts := doc.Summary.BySeverity
	if _, err := fmt.Fprintf(
		out,
		"| %d | %d | %d | %d | %d |\n\n",
		counts.Critical,
		counts.Serious,
		counts.Moderate,
		counts.Minor,
		infoCount(counts),
	); err != nil {
		return err
	}

	if filters.Truncated || len(filters.Severities) > 0 || len(filters.Categories) > 0 {
		if _, err := fmt.Fprintf(
			out,
			"Showing %d of %d findings after output limits and filters; severity totals describe the full scan.\n\n",
			filters.IssuesReturned,
			doc.Summary.TotalIssues,
		); err != nil {
			return err
		}
	}

	return nil
}

func writeMarkdownNotes(out io.Writer, heading string, notes []string) error {
	if len(notes) == 0 {
		return nil
	}

	if _, err := fmt.Fprintf(out, "### %s\n\n", heading); err != nil {
		return err
	}

	for _, note := range notes {
		if _, err := fmt.Fprintf(out, "- %s\n", markdownCell(note)); err != nil {
			return err
		}
	}

	_, err := fmt.Fprintln(out)

	return err
}

type markdownRuleGroup struct {
	issue report.IssueDetail
	pages map[string]struct{}
	count int
}

func writeMarkdownRuleGroups(out io.Writer, issues []report.IssueDetail) error {
	ordered := groupMarkdownRules(issues)

	if _, err := fmt.Fprint(out, "### Findings\n\n"); err != nil {
		return err
	}

	for _, group := range ordered {
		issue := group.issue
		if _, err := fmt.Fprintf(
			out,
			"- **[%s] %s** — %d finding(s) across %d page(s) (%s).\n",
			issue.Severity,
			markdownCell(issue.Title),
			group.count,
			len(group.pages),
			issue.Scanner,
		); err != nil {
			return err
		}

		fix := stringValue(issue.HowToFix)
		if fix == "" && len(issue.Occurrences) > 0 {
			fix = stringValue(issue.Occurrences[0].FailureSummary)
		}

		if fix != "" {
			if _, err := fmt.Fprintf(out, "  Fix: %s\n", markdownCell(fix)); err != nil {
				return err
			}
		}
	}

	_, err := fmt.Fprintln(out)

	return err
}

func groupMarkdownRules(issues []report.IssueDetail) []*markdownRuleGroup {
	groups := make(map[string]*markdownRuleGroup)

	for _, issue := range issues {
		key := issue.Scanner + "/" + issue.RuleId

		group := groups[key]
		if group == nil {
			group = &markdownRuleGroup{issue: issue, pages: make(map[string]struct{})}
			groups[key] = group
		}

		group.pages[issue.PageUrl] = struct{}{}

		group.count++
		if severityRank(issue.Severity) > severityRank(group.issue.Severity) {
			group.issue = issue
		}
	}

	ordered := make([]*markdownRuleGroup, 0, len(groups))
	for _, group := range groups {
		ordered = append(ordered, group)
	}

	sort.Slice(ordered, func(i, j int) bool {
		a, b := ordered[i], ordered[j]
		if severityRank(a.issue.Severity) != severityRank(b.issue.Severity) {
			return severityRank(a.issue.Severity) > severityRank(b.issue.Severity)
		}

		if len(a.pages) != len(b.pages) {
			return len(a.pages) > len(b.pages)
		}

		return a.issue.Scanner+"/"+a.issue.RuleId < b.issue.Scanner+"/"+b.issue.RuleId
	})

	return ordered
}

func markdownCell(value string) string {
	return strings.NewReplacer("|", "\\|", "\n", " ", "\r", " ", "<", "&lt;", ">", "&gt;").
		Replace(strings.TrimSpace(value))
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}
