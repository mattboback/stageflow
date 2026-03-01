package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

	report "github.com/mattboback/stageflow/packages/contracts/report/generated/go"
)

type renderOptions struct {
	format      string
	maxIssues   int
	minSeverity severityLevel
	threshold   thresholdResult
}

type jsonReportOutput struct {
	JobID           string            `json:"job_id"`
	State           string            `json:"state"`
	URLs            []string          `json:"urls"`
	DurationMS      *int64            `json:"duration_ms,omitempty"`
	Score           *int              `json:"score,omitempty"`
	ScoreGrade      *string           `json:"score_grade,omitempty"`
	Summary         jsonSummary       `json:"summary"`
	Issues          []jsonIssue       `json:"issues"`
	ThresholdResult *string           `json:"threshold_result,omitempty"`
	ThresholdDetail *string           `json:"threshold_detail,omitempty"`
	Errors          []jsonReportError `json:"errors"`
}

type jsonSummary struct {
	TotalIssues     int               `json:"total_issues"`
	BySeverity      jsonSeverityCount `json:"by_severity"`
	ByScanner       map[string]int    `json:"by_scanner,omitempty"`
	PagesScanned    int               `json:"pages_scanned"`
	PagesWithIssues int               `json:"pages_with_issues"`
}

type jsonSeverityCount struct {
	Critical int `json:"critical"`
	Serious  int `json:"serious"`
	Moderate int `json:"moderate"`
	Minor    int `json:"minor"`
	Info     int `json:"info"`
}

type jsonIssue struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Severity    string  `json:"severity"`
	Scanner     string  `json:"scanner"`
	RuleID      string  `json:"rule_id"`
	PageURL     string  `json:"page_url"`
	Description string  `json:"description"`
	HelpURL     *string `json:"help_url,omitempty"`
}

type jsonReportError struct {
	Code      string  `json:"code"`
	Message   string  `json:"message"`
	Scope     string  `json:"scope"`
	Retryable bool    `json:"retryable"`
	PageID    *string `json:"page_id,omitempty"`
	ScannerID *string `json:"scanner_id,omitempty"`
}

func validateReportFormat(raw string) (string, error) {
	format := strings.ToLower(strings.TrimSpace(raw))

	switch format {
	case outputFormatSummary, outputFormatJSON, outputFormatQuiet:
		return format, nil
	default:
		return "", fmt.Errorf("invalid format %q (expected summary, json, quiet)", raw)
	}
}

func validateScannersFormat(raw string) (string, error) {
	format := strings.ToLower(strings.TrimSpace(raw))

	switch format {
	case outputFormatSummary, outputFormatJSON:
		return format, nil
	default:
		return "", fmt.Errorf("invalid format %q (expected summary, json)", raw)
	}
}

func fetchJobStatus(ctx context.Context, client *Client, jobID string) (JobStatus, error) {
	var status JobStatus

	apiPath := fmt.Sprintf("/api/v1/jobs/%s", url.PathEscape(jobID))
	if err := client.getJSON(ctx, apiPath, &status); err != nil {
		return JobStatus{}, err
	}

	return status, nil
}

func fetchReport(ctx context.Context, client *Client, jobID string) (report.UnifiedReportV2, error) {
	var doc report.UnifiedReportV2

	apiPath := fmt.Sprintf("/api/v1/jobs/%s/results", url.PathEscape(jobID))
	if err := client.getJSON(ctx, apiPath, &doc); err != nil {
		return report.UnifiedReportV2{}, err
	}

	return doc, nil
}

func renderReport(out io.Writer, status JobStatus, doc report.UnifiedReportV2, opts renderOptions) error {
	filtered := filterIssues(doc.Issues, opts.minSeverity, opts.maxIssues)

	switch opts.format {
	case "summary":
		return writeSummaryReport(out, status, doc, filtered, opts.threshold)
	case "json":
		return writeJSONReport(out, status, doc, filtered, opts.threshold)
	case "quiet":
		return writeQuietReport(out, doc, opts.threshold)
	default:
		return fmt.Errorf("unsupported format %q", opts.format)
	}
}

func writeSummaryReport(
	out io.Writer,
	status JobStatus,
	doc report.UnifiedReportV2,
	filtered []report.IssueDetail,
	th thresholdResult,
) error {
	if err := writeSummaryHeader(out, status, doc, th); err != nil {
		return err
	}

	return writeSummaryIssues(out, doc, filtered)
}

func writeJSONReport(
	out io.Writer,
	status JobStatus,
	doc report.UnifiedReportV2,
	filtered []report.IssueDetail,
	th thresholdResult,
) error {
	payload := jsonReportOutput{
		JobID:      status.ID,
		State:      status.State,
		URLs:       collectReportURLs(doc),
		DurationMS: durationMS(doc.Meta.DurationMs),
		Score:      doc.Summary.Score,
		ScoreGrade: doc.Summary.ScoreGrade,
		Summary: jsonSummary{
			TotalIssues: doc.Summary.TotalIssues,
			BySeverity: jsonSeverityCount{
				Critical: doc.Summary.BySeverity.Critical,
				Serious:  doc.Summary.BySeverity.Serious,
				Moderate: doc.Summary.BySeverity.Moderate,
				Minor:    doc.Summary.BySeverity.Minor,
				Info:     infoCount(doc.Summary.BySeverity),
			},
			ByScanner:       copyByScanner(doc.Summary.ByScanner),
			PagesScanned:    doc.Summary.PagesScanned,
			PagesWithIssues: doc.Summary.PagesWithIssues,
		},
		Issues: make([]jsonIssue, 0, len(filtered)),
		Errors: make([]jsonReportError, 0, len(doc.Errors)),
	}

	if th.Evaluated {
		result := "pass"
		if !th.Passed {
			result = "fail"
		}

		payload.ThresholdResult = &result
		if th.Detail != "" {
			payload.ThresholdDetail = &th.Detail
		}
	}

	for _, issue := range filtered {
		payload.Issues = append(payload.Issues, jsonIssue{
			ID:          issue.Id,
			Title:       issue.Title,
			Severity:    string(issue.Severity),
			Scanner:     issue.Scanner,
			RuleID:      issue.RuleId,
			PageURL:     issue.PageUrl,
			Description: issue.Description,
			HelpURL:     issue.HelpUrl,
		})
	}

	for _, item := range doc.Errors {
		payload.Errors = append(payload.Errors, jsonReportError{
			Code:      item.Code,
			Message:   item.Message,
			Scope:     string(item.Scope),
			Retryable: item.Retryable,
			PageID:    item.PageId,
			ScannerID: item.ScannerId,
		})
	}

	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)

	return encoder.Encode(payload)
}

func writeQuietReport(out io.Writer, doc report.UnifiedReportV2, th thresholdResult) error {
	line := fmt.Sprintf(
		"PASS: critical=%d serious=%d total=%d",
		doc.Summary.BySeverity.Critical,
		doc.Summary.BySeverity.Serious,
		doc.Summary.TotalIssues,
	)

	if th.Evaluated {
		if th.Passed {
			line = fmt.Sprintf(
				"PASS: critical=%d serious=%d total=%d",
				doc.Summary.BySeverity.Critical,
				doc.Summary.BySeverity.Serious,
				doc.Summary.TotalIssues,
			)
		} else {
			line = "FAIL: " + th.Detail
		}
	}

	_, err := fmt.Fprintln(out, line)

	return err
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

	return fmt.Sprintf("%dms", *ms)
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

func copyByScanner(source map[string]int) map[string]int {
	if len(source) == 0 {
		return nil
	}

	dest := make(map[string]int, len(source))
	for key, value := range source {
		dest[key] = value
	}

	return dest
}

func writeSummaryHeader(
	out io.Writer,
	status JobStatus,
	doc report.UnifiedReportV2,
	th thresholdResult,
) error {
	lines := buildSummaryLines(status, doc, th)
	for _, line := range lines {
		if _, err := fmt.Fprintln(out, line); err != nil {
			return err
		}
	}

	return nil
}

func buildSummaryLines(status JobStatus, doc report.UnifiedReportV2, th thresholdResult) []string {
	lines := []string{
		"Job ID: " + status.ID,
		"State: " + status.State,
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

	lines = append(lines, fmt.Sprintf(
		"Severity Totals: critical=%d serious=%d moderate=%d minor=%d info=%d total=%d",
		doc.Summary.BySeverity.Critical,
		doc.Summary.BySeverity.Serious,
		doc.Summary.BySeverity.Moderate,
		doc.Summary.BySeverity.Minor,
		infoCount(doc.Summary.BySeverity),
		doc.Summary.TotalIssues,
	))

	thresholdLine := summaryThresholdLine(th)
	if thresholdLine != "" {
		lines = append(lines, thresholdLine)
	}

	if len(doc.Errors) > 0 {
		lines = append(lines, fmt.Sprintf("Report Errors: %d", len(doc.Errors)))
	}

	return lines
}

func summaryThresholdLine(th thresholdResult) string {
	if !th.Evaluated {
		return ""
	}

	statusText := "PASS"
	if !th.Passed {
		statusText = "FAIL"
	}

	if th.Detail == "" {
		return "Thresholds: " + statusText
	}

	return fmt.Sprintf("Thresholds: %s (%s)", statusText, th.Detail)
}

func writeSummaryIssues(out io.Writer, doc report.UnifiedReportV2, filtered []report.IssueDetail) error {
	if len(filtered) == 0 {
		return writeNoMatchingIssues(out, len(doc.Issues) == 0)
	}

	if _, err := fmt.Fprintln(out, "\nIssues:"); err != nil {
		return err
	}

	for _, issue := range filtered {
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
	}

	return nil
}

func writeNoMatchingIssues(out io.Writer, noIssues bool) error {
	message := "\nNo issues matched the current filters."
	if noIssues {
		message = "\nNo issues found."
	}

	_, err := fmt.Fprintln(out, message)

	return err
}
