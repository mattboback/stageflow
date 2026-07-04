package render

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
	"github.com/mattboback/stageflow/clients/cli/internal/buildinfo"
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

const issueSortOrder = "severity_desc, scanner_asc, rule_id_asc, page_url_asc, id_asc"

type CLIMeta struct {
	Version string `json:"version"`
	Commit  string `json:"commit,omitempty"`
	Date    string `json:"date,omitempty"`
}

type APIMeta struct {
	BaseURL string `json:"base_url"`
}

type JobMeta struct {
	ID        string `json:"id"`
	State     string `json:"state"`
	Error     string `json:"error,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type ReportLinks struct {
	Job     string `json:"job"`
	Results string `json:"results"`
}

type IssueFilters struct {
	MaxIssues      int      `json:"max_issues"`
	IssuesReturned int      `json:"issues_returned"`
	IssuesTotal    int      `json:"issues_total"`
	Truncated      bool     `json:"truncated"`
	Sort           string   `json:"sort"`
	Severities     []string `json:"severities,omitempty"`
	Categories     []string `json:"categories,omitempty"`
}

type ReportEnvelope struct {
	Schema  string                 `json:"schema"`
	CLI     CLIMeta                `json:"cli"`
	API     APIMeta                `json:"api"`
	Job     JobMeta                `json:"job"`
	Links   ReportLinks            `json:"links"`
	URLs    []string               `json:"urls,omitempty"`
	Filters IssueFilters           `json:"filters"`
	Report  report.UnifiedReportV2 `json:"report"`
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

func ShouldFailForSeverity(issues []report.IssueDetail, threshold string) (bool, error) {
	if threshold == "" {
		return false, nil
	}

	return HasIssuesAtOrAbove(issues, threshold)
}

func BuildReportEnvelope(
	apiBaseURL string,
	status apiclient.JobStatus,
	doc report.UnifiedReportV2,
	filters IssueFilters,
) (ReportEnvelope, error) {
	jobLink, err := buildAPILink(apiBaseURL, fmt.Sprintf("/api/v1/jobs/%s", url.PathEscape(status.ID)))
	if err != nil {
		return ReportEnvelope{}, err
	}

	resultsLink, err := buildAPILink(apiBaseURL, fmt.Sprintf("/api/v1/jobs/%s/results", url.PathEscape(status.ID)))
	if err != nil {
		return ReportEnvelope{}, err
	}

	job := JobMeta{
		ID:    status.ID,
		State: status.State,
		Error: status.Error,
	}

	if !status.CreatedAt.IsZero() {
		job.CreatedAt = status.CreatedAt.UTC().Format(timeFormatRFC3339)
	}

	if !status.UpdatedAt.IsZero() {
		job.UpdatedAt = status.UpdatedAt.UTC().Format(timeFormatRFC3339)
	}

	payload := ReportEnvelope{
		Schema: "stageflow-cli/report@v1",
		CLI:    currentCLIMeta(),
		API: APIMeta{
			BaseURL: apiBaseURL,
		},
		Job: job,
		Links: ReportLinks{
			Job:     jobLink,
			Results: resultsLink,
		},
		URLs:    collectReportURLs(doc),
		Filters: filters,
		Report:  doc,
	}

	return payload, nil
}

const timeFormatRFC3339 = "2006-01-02T15:04:05Z07:00"

func currentCLIMeta() CLIMeta {
	v := strings.TrimSpace(buildinfo.Version)
	if v == "" {
		v = "dev"
	}

	c := strings.TrimSpace(buildinfo.Commit)
	d := strings.TrimSpace(buildinfo.Date)

	return CLIMeta{
		Version: v,
		Commit:  c,
		Date:    d,
	}
}

func buildAPILink(baseURL, apiPath string) (string, error) {
	base, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return "", fmt.Errorf("invalid API URL %q: %w", baseURL, err)
	}

	ref, err := url.Parse(apiPath)
	if err != nil {
		return "", fmt.Errorf("invalid API path %q: %w", apiPath, err)
	}

	return base.ResolveReference(ref).String(), nil
}

func selectIssues(issues []report.IssueDetail, maxIssues int) ([]report.IssueDetail, IssueFilters) {
	total := len(issues)
	if total == 0 {
		return []report.IssueDetail{}, IssueFilters{
			MaxIssues:      maxIssues,
			IssuesReturned: 0,
			IssuesTotal:    0,
			Truncated:      false,
			Sort:           issueSortOrder,
		}
	}

	sorted := make([]report.IssueDetail, 0, total)
	sorted = append(sorted, issues...)

	slices.SortFunc(sorted, func(a, b report.IssueDetail) int {
		as := severityRank(a.Severity)
		bs := severityRank(b.Severity)

		if as != bs {
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
		if dataBytes, err := json.Marshal(issue.ScannerData); err == nil && string(dataBytes) != "{}" {
			if _, printErr := fmt.Fprintf(out, "  Details: %s\n", string(dataBytes)); printErr != nil {
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

func collectReportURLs(doc report.UnifiedReportV2) []string {
	seen := make(map[string]struct{}, len(doc.Pages)+1)
	urls := make([]string, 0, len(doc.Pages)+1)

	if doc.Meta.BaseUrl != nil && *doc.Meta.BaseUrl != "" {
		seen[*doc.Meta.BaseUrl] = struct{}{}
		urls = append(urls, *doc.Meta.BaseUrl)
	}

	for _, page := range doc.Pages {
		if page.Url == "" {
			continue
		}

		if _, ok := seen[page.Url]; ok {
			continue
		}

		seen[page.Url] = struct{}{}
		urls = append(urls, page.Url)
	}

	return urls
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
