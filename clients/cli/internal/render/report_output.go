package render

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
	"github.com/mattboback/stageflow/clients/cli/internal/exitcode"
	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
)

type Options struct {
	Format         Format
	MaxIssues      int
	MaxOccurrences int
	Severities     []string
	Categories     []string
	FailSeverity   string
	SummaryOnly    bool
	GroupBy        string
}

func FetchJobStatus(ctx context.Context, client *apiclient.Client, jobID string) (apiclient.JobStatus, error) {
	var status apiclient.JobStatus

	apiPath := fmt.Sprintf("/api/v1/jobs/%s", url.PathEscape(jobID))
	if err := client.GetJSON(ctx, apiPath, &status); err != nil {
		return apiclient.JobStatus{}, err
	}

	return status, nil
}

func FetchReport(ctx context.Context, client *apiclient.Client, jobID string) (report.UnifiedReportV2, error) {
	apiPath := fmt.Sprintf("/api/v1/jobs/%s/results", url.PathEscape(jobID))

	var raw json.RawMessage
	if err := client.GetJSON(ctx, apiPath, &raw); err != nil {
		return report.UnifiedReportV2{}, err
	}

	raw = SanitizeScoreGrade(raw)

	var doc report.UnifiedReportV2
	if err := json.Unmarshal(raw, &doc); err != nil {
		return report.UnifiedReportV2{}, fmt.Errorf("failed to decode report: %w", err)
	}

	return doc, nil
}

// SanitizeScoreGrade fixes scoreGrade values that don't match the schema
// pattern ^[A-F][+-]?$. Older API versions returned "Excellent" for
// perfect scores; the CLI must tolerate these to avoid crashing on
// existing reports.
func SanitizeScoreGrade(raw json.RawMessage) json.RawMessage {
	return scoreGradeReplacer.ReplaceAll(raw, []byte(`"scoreGrade":"A+"`))
}

var scoreGradeReplacer = regexp.MustCompile(`"scoreGrade"\s*:\s*"(?:Excellent)"`)

func UnifiedReport(
	out io.Writer,
	apiBaseURL string,
	status apiclient.JobStatus,
	doc report.UnifiedReportV2,
	opts Options,
) error {
	selectedIssues, filters, err := ValidatedIssueSelection(doc.Issues, opts)
	if err != nil {
		return err
	}

	docFiltered := doc
	docFiltered.Issues = selectedIssues

	if renderErr := writeRenderedReport(out, apiBaseURL, status, docFiltered, filters, opts); renderErr != nil {
		return renderErr
	}

	fail, err := ShouldFailForSeverity(selectedIssues, opts.FailSeverity)
	if err != nil {
		return err
	}

	if fail {
		return exitcode.Error{Code: 1}
	}

	return nil
}

func writeRenderedReport(
	out io.Writer,
	apiBaseURL string,
	status apiclient.JobStatus,
	doc report.UnifiedReportV2,
	filters IssueFilters,
	opts Options,
) error {
	switch opts.Format {
	case FormatText:
		return writeSummaryReport(out, apiBaseURL, status, doc, filters, opts.MaxOccurrences, opts.SummaryOnly)
	case FormatJSON:
		return writeJSONReport(out, apiBaseURL, status, doc, filters, opts.SummaryOnly)
	case FormatMarkdown:
		return writeMarkdownReport(out, apiBaseURL, status, doc, filters, markdownRenderOptions{
			MaxOccurrences: opts.MaxOccurrences,
			SummaryOnly:    opts.SummaryOnly,
			GroupBy:        opts.GroupBy,
		})
	default:
		return fmt.Errorf("unsupported output format %q", opts.Format)
	}
}

func writeJSONReport(
	out io.Writer,
	apiBaseURL string,
	status apiclient.JobStatus,
	doc report.UnifiedReportV2,
	filters IssueFilters,
	summaryOnly bool,
) error {
	if summaryOnly {
		doc.Issues = nil
	}

	payload, err := BuildReportEnvelope(apiBaseURL, status, doc, filters)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)

	return encoder.Encode(payload)
}

func writeSummaryReport(
	out io.Writer,
	apiBaseURL string,
	status apiclient.JobStatus,
	doc report.UnifiedReportV2,
	filters IssueFilters,
	maxOccurrences int,
	summaryOnly bool,
) error {
	lines := buildSummaryLines(apiBaseURL, status, doc, filters)

	for _, line := range lines {
		if _, writeErr := fmt.Fprintln(out, line); writeErr != nil {
			return writeErr
		}
	}

	if summaryOnly {
		return nil
	}

	return writeSummaryIssues(out, doc, maxOccurrences)
}

//nolint:gocyclo
func buildSummaryLines(
	_ string,
	status apiclient.JobStatus,
	doc report.UnifiedReportV2,
	filters IssueFilters,
) []string {
	lines := []string{
		"Job: " + status.ID,
	}

	if status.State != apiclient.JobStateDone {
		lines = append(lines, "State: "+status.State)
	}

	urls := collectReportURLs(doc)
	if len(urls) > 0 {
		lines = append(lines, "URLs: "+strings.Join(urls, ", "))
	}

	duration := formatDuration(doc)
	if duration != "" {
		lines = append(lines, "Duration: "+duration)
	}

	score := formatScore(doc)
	if score != "" {
		lines = append(lines, "Score: "+score)
	}

	lines = append(lines, fmt.Sprintf(
		"Pages: %d scanned, %d with issues",
		doc.Summary.PagesScanned,
		doc.Summary.PagesWithIssues,
	))

	bySev := doc.Summary.BySeverity

	var severityParts []string

	if bySev.Critical > 0 {
		severityParts = append(severityParts, fmt.Sprintf("critical=%d", bySev.Critical))
	}

	if bySev.Serious > 0 {
		severityParts = append(severityParts, fmt.Sprintf("serious=%d", bySev.Serious))
	}

	if bySev.Moderate > 0 {
		severityParts = append(severityParts, fmt.Sprintf("moderate=%d", bySev.Moderate))
	}

	if bySev.Minor > 0 {
		severityParts = append(severityParts, fmt.Sprintf("minor=%d", bySev.Minor))
	}

	if infoCount(bySev) > 0 {
		severityParts = append(severityParts, fmt.Sprintf("info=%d", infoCount(bySev)))
	}

	if len(severityParts) > 0 {
		severityParts = append(severityParts, fmt.Sprintf("total=%d", doc.Summary.TotalIssues))
		lines = append(lines, "Severity: "+strings.Join(severityParts, " "))
	}

	if lh := doc.Summary.LighthouseCategories; len(lh) > 0 {
		var parts []string
		for _, cat := range lh {
			parts = append(parts, fmt.Sprintf("%s=%.2f", string(cat.Id), cat.AvgScore))
		}

		lines = append(lines, "Lighthouse: "+strings.Join(parts, " "))
	}

	if filters.Truncated {
		lines = append(lines, fmt.Sprintf(
			"Issues: %d returned (truncated from %d; max=%d)",
			filters.IssuesReturned,
			filters.IssuesTotal,
			filters.MaxIssues,
		))
	} else {
		lines = append(lines, fmt.Sprintf("Issues: %d", filters.IssuesReturned))
	}

	if len(filters.Severities) > 0 {
		lines = append(lines, "Severity filter: "+strings.Join(filters.Severities, ", "))
	}

	if len(filters.Categories) > 0 {
		lines = append(lines, "Category filter: "+strings.Join(filters.Categories, ", "))
	}

	if len(doc.Errors) > 0 {
		lines = append(lines, fmt.Sprintf("Report Errors: %d", len(doc.Errors)))
	}

	return lines
}

func writeSummaryIssues(out io.Writer, doc report.UnifiedReportV2, maxOccurrences int) error {
	if len(doc.Issues) == 0 {
		_, err := fmt.Fprintln(out, "\nNo issues found.")
		return err
	}

	if _, err := fmt.Fprintln(out, "\nIssues:"); err != nil {
		return err
	}

	for i, issue := range doc.Issues {
		if i > 0 {
			fmt.Fprintln(out)
		}

		if err := writeSummaryIssue(out, issue, maxOccurrences); err != nil {
			return err
		}
	}

	return nil
}

func writeSummaryIssue(out io.Writer, issue report.IssueDetail, maxOccurrences int) error {
	if _, err := fmt.Fprintf(
		out,
		"- [%s] %s (%s/%s)\n  %s\n  %s\n",
		issue.Severity,
		issue.Title,
		issue.Scanner,
		issue.RuleId,
		issue.PageUrl,
		issue.Description,
	); err != nil {
		return err
	}

	if issue.Category != nil && *issue.Category != "" {
		if _, err := fmt.Fprintf(out, "  Category: %s\n", *issue.Category); err != nil {
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

	if len(issue.ScannerData) > 0 {
		if dataBytes, err := json.MarshalIndent(issue.ScannerData, "    ", "  "); err == nil &&
			string(dataBytes) != "{}" {
			if _, printErr := fmt.Fprintf(out, "  Details:\n    %s\n", string(dataBytes)); printErr != nil {
				return printErr
			}
		}
	}

	return writeTextOccurrences(out, issue.Occurrences, maxOccurrences, stringValue(issue.HowToFix))
}

func writeTextOccurrences(
	out io.Writer,
	occurrences []report.IssueOccurrence,
	maxOccurrences int,
	issueHowToFix string,
) error {
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

		if fs := stringValue(occ.FailureSummary); fs != "" && fs != issueHowToFix {
			if _, err := fmt.Fprintf(out, "       %s\n", fs); err != nil {
				return err
			}
		}
	}

	return nil
}

func formatOccurrenceLine(index int, occ report.IssueOccurrence) string {
	selector := stringValue(occ.Selector)
	html := stringValue(occ.Html)
	failureSummary := stringValue(occ.FailureSummary)

	line := fmt.Sprintf("    %d.", index)

	switch {
	case selector != "":
		line += fmt.Sprintf(" `%s`", selector)
		if html != "" {
			line += " — " + html
		}
	case html != "":
		line += " " + html
	case failureSummary != "":
		line += " " + failureSummary
	}

	return line
}

func formatDuration(doc report.UnifiedReportV2) string {
	ms := durationMS(doc.Meta.DurationMs)
	if ms == nil {
		return ""
	}

	return formatDurationMs(*ms)
}

func formatDurationMs(durationMs int64) string {
	return fmt.Sprintf("%.1fs", float64(durationMs)/1000)
}

func durationMS(value *float64) *int64 {
	if value == nil {
		return nil
	}

	rounded := int64(*value)

	return &rounded
}

func formatScore(doc report.UnifiedReportV2) string {
	if doc.Summary.Score == nil {
		return ""
	}

	if doc.Summary.ScoreGrade != nil && *doc.Summary.ScoreGrade != "" {
		return fmt.Sprintf("%d (%s)", *doc.Summary.Score, *doc.Summary.ScoreGrade)
	}

	return strconv.Itoa(*doc.Summary.Score)
}

func infoCount(counts report.SeverityCounts) int {
	if counts.Info == nil {
		return 0
	}

	return *counts.Info
}
