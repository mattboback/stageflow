package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
	"github.com/mattboback/stageflow/libs/go/diff"
)

type diffEnvelope struct {
	Schema    string               `json:"schema"`
	Baseline  diffBaselineMeta     `json:"baseline"`
	Current   diffCurrentMeta      `json:"current"`
	Delta     diffDelta            `json:"delta"`
	New       []report.IssueDetail `json:"new"`
	Fixed     []report.IssueDetail `json:"fixed"`
	Regressed bool                 `json:"regressed"`
}

type diffBaselineMeta struct {
	File        string `json:"file"`
	Score       *int   `json:"score,omitempty"`
	TotalIssues int    `json:"totalIssues"`
}

type diffCurrentMeta struct {
	JobID       string `json:"jobId,omitempty"`
	File        string `json:"file,omitempty"`
	Score       *int   `json:"score,omitempty"`
	TotalIssues int    `json:"totalIssues"`
}

type diffDelta struct {
	ScoreDelta      *int `json:"scoreDelta,omitempty"`
	NewIssues       int  `json:"newIssues"`
	FixedIssues     int  `json:"fixedIssues"`
	UnchangedIssues int  `json:"unchangedIssues"`
}

func newDiffCmd(root *rootOptions) *cobra.Command {
	var (
		failOnNew        string
		failOnRegression bool
		timeout          time.Duration
		noStream         bool
	)

	cmd := &cobra.Command{
		Use:                   "diff <baseline.json> <current.json | url>",
		Short:                 "Compare a current scan against a saved baseline",
		DisableFlagsInUseLine: true,
		Args:                  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDiffCommand(
				cmd,
				root,
				args[0],
				args[1],
				timeout,
				noStream,
				failOnNew,
				failOnRegression,
			)
		},
	}

	cmd.Flags().StringVar(
		&failOnNew,
		"fail-on-new",
		"",
		"Exit 1 if any NEW issue meets threshold "+
			"(critical, serious, moderate, minor, info) or 'any'",
	)
	cmd.Flags().Lookup("fail-on-new").NoOptDefVal = "any"
	cmd.Flags().BoolVar(
		&failOnRegression,
		"fail-on-regression",
		false,
		"Exit 1 if score dropped or new issues appeared",
	)
	cmd.Flags().DurationVar(&timeout, "timeout", 5*time.Minute, "Max wait time for live scan")
	cmd.Flags().BoolVar(&noStream, "no-stream", false, "Poll instead of SSE for live scan")
	cobra.CheckErr(cmd.Flags().MarkHidden("no-stream"))

	return cmd
}

func runDiffCommand(
	cmd *cobra.Command,
	root *rootOptions,
	baselinePath, currentTarget string,
	timeout time.Duration,
	noStream bool,
	failOnNew string,
	failOnRegression bool,
) error {
	baselineEnv, err := loadReportFile(baselinePath)
	if err != nil {
		return exitCodeError{Code: 2, Err: fmt.Errorf("baseline: %w", err)}
	}

	currentEnv, currentJobID, currentFile, err := loadCurrentDiffTarget(
		cmd,
		root,
		baselineEnv,
		currentTarget,
		timeout,
		noStream,
	)
	if err != nil {
		return exitCodeError{Code: 2, Err: err}
	}

	result := diff.ComputeDiff("", baselineEnv.Report, currentJobID, currentEnv.Report)
	d := diffFromResult(result, baselinePath, currentFile)

	regressed, err := evaluateDiffRegression(d, failOnRegression, failOnNew)
	if err != nil {
		return exitCodeError{Code: 2, Err: err}
	}

	d.Regressed = regressed

	format, err := root.outputFormat()
	if err != nil {
		return exitCodeError{Code: 2, Err: err}
	}

	err = renderDiff(cmd.OutOrStdout(), d, format)
	if err != nil {
		return exitCodeError{Code: 2, Err: err}
	}

	if regressed {
		return exitCodeError{Code: 1}
	}

	return nil
}

func loadCurrentDiffTarget(
	cmd *cobra.Command,
	root *rootOptions,
	baselineEnv reportEnvelope,
	currentTarget string,
	timeout time.Duration,
	noStream bool,
) (reportEnvelope, string, string, error) {
	if !isRemoteDiffTarget(currentTarget) {
		currentEnv, err := loadReportFile(currentTarget)
		if err != nil {
			return reportEnvelope{}, "", "", fmt.Errorf("current: %w", err)
		}

		return currentEnv, "", currentTarget, nil
	}

	currentEnv, jobID, err := runLiveDiffScan(cmd, root, baselineEnv, currentTarget, timeout, noStream)
	if err != nil {
		return reportEnvelope{}, "", "", err
	}

	return currentEnv, jobID, "", nil
}

func isRemoteDiffTarget(target string) bool {
	return strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://")
}

func runLiveDiffScan(
	cmd *cobra.Command,
	root *rootOptions,
	baselineEnv reportEnvelope,
	currentTarget string,
	timeout time.Duration,
	noStream bool,
) (reportEnvelope, string, error) {
	client := NewClient(root.apiURL, root.apiKey, nil)
	req := SubmitJobRequest{
		URLs:                []string{currentTarget},
		Modules:             diffScanModules(baselineEnv),
		AllowPrivateTargets: containsPrivateTargets([]string{currentTarget}),
	}

	status, doc, err := runScanJob(
		cmd.Context(),
		client,
		req,
		timeout,
		cmd.ErrOrStderr(),
		noStream,
	)
	if err != nil {
		return reportEnvelope{}, "", err
	}

	return reportEnvelope{
		Job:    jobMeta{ID: status.ID},
		Report: doc,
	}, status.ID, nil
}

func diffScanModules(env reportEnvelope) []string {
	modules := make([]string, 0, len(env.Report.Scanners))
	for _, scanner := range env.Report.Scanners {
		modules = append(modules, scanner.Id)
	}

	return modules
}

func evaluateDiffRegression(d diffEnvelope, failOnRegression bool, failOnNew string) (bool, error) {
	regressed := failOnRegression && isDiffRegressed(d)

	if failOnNew == "" || len(d.New) == 0 {
		return regressed, nil
	}

	if failOnNew == "any" {
		return true, nil
	}

	hasSeverity, err := hasIssuesAtOrAbove(d.New, failOnNew)
	if err != nil {
		return false, err
	}

	return regressed || hasSeverity, nil
}

func isDiffRegressed(d diffEnvelope) bool {
	return (d.Delta.ScoreDelta != nil && *d.Delta.ScoreDelta < 0) || d.Delta.NewIssues > 0
}

func diffFromResult(r diff.Result, baselineFile, currentFile string) diffEnvelope {
	return diffEnvelope{
		Schema: r.Schema,
		Baseline: diffBaselineMeta{
			File:        baselineFile,
			Score:       r.Baseline.Score,
			TotalIssues: r.Baseline.TotalIssues,
		},
		Current: diffCurrentMeta{
			JobID:       r.Current.JobID,
			File:        currentFile,
			Score:       r.Current.Score,
			TotalIssues: r.Current.TotalIssues,
		},
		Delta: diffDelta{
			ScoreDelta:      r.Delta.ScoreDelta,
			NewIssues:       r.Delta.NewIssues,
			FixedIssues:     r.Delta.FixedIssues,
			UnchangedIssues: r.Delta.UnchangedIssues,
		},
		New:   r.New,
		Fixed: r.Fixed,
	}
}

func renderDiff(out io.Writer, diff diffEnvelope, format outputFormat) error {
	switch format {
	case outputFormatJSON:
		return writeJSONDiff(out, diff)
	case outputFormatText, outputFormatMarkdown:
		return writeTextDiff(out, diff, format == outputFormatMarkdown)
	default:
		return fmt.Errorf("unsupported output format %q", format)
	}
}

func writeJSONDiff(out io.Writer, diff diffEnvelope) error {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)

	return encoder.Encode(diff)
}

func writeTextDiff(out io.Writer, diff diffEnvelope, isMarkdown bool) error {
	err := writeDiffHeading(out, isMarkdown)
	if err != nil {
		return err
	}

	err = writeDiffSummary(out, diff, isMarkdown)
	if err != nil {
		return err
	}

	return writeNewIssueSection(out, diff.New, isMarkdown)
}

func writeDiffHeading(out io.Writer, isMarkdown bool) error {
	if !isMarkdown {
		return nil
	}

	_, err := fmt.Fprintln(out, "## Regression Diff")
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(out)

	return err
}

func writeDiffSummary(out io.Writer, diff diffEnvelope, isMarkdown bool) error {
	scoreStr := formatDiffScore(diff)

	if isMarkdown {
		_, err := fmt.Fprintf(
			out,
			"- **Score**: %s\n- **New issues**: %d\n- **Fixed issues**: %d\n- **Unchanged**: %d\n\n",
			scoreStr,
			diff.Delta.NewIssues,
			diff.Delta.FixedIssues,
			diff.Delta.UnchangedIssues,
		)

		return err
	}

	_, err := fmt.Fprintf(
		out,
		"Score: %s\nNew issues: %d\nFixed issues: %d\nUnchanged: %d\n\n",
		scoreStr,
		diff.Delta.NewIssues,
		diff.Delta.FixedIssues,
		diff.Delta.UnchangedIssues,
	)

	return err
}

func formatDiffScore(diff diffEnvelope) string {
	if diff.Baseline.Score == nil || diff.Current.Score == nil {
		return "N/A"
	}

	delta := *diff.Current.Score - *diff.Baseline.Score
	sign := ""

	if delta > 0 {
		sign = "+"
	}

	return fmt.Sprintf("%d → %d (%s%d)", *diff.Baseline.Score, *diff.Current.Score, sign, delta)
}

func writeNewIssueSection(out io.Writer, issues []report.IssueDetail, isMarkdown bool) error {
	if len(issues) == 0 {
		_, err := fmt.Fprintln(out, "No new issues detected.")

		return err
	}

	err := writeNewIssueHeader(out, isMarkdown)
	if err != nil {
		return err
	}

	for _, issue := range issues {
		err = writeDiffIssue(out, issue, isMarkdown)
		if err != nil {
			return err
		}
	}

	return nil
}

func writeNewIssueHeader(out io.Writer, isMarkdown bool) error {
	if !isMarkdown {
		_, err := fmt.Fprintln(out, "New Issues (Regressions):")

		return err
	}

	_, err := fmt.Fprintln(out, "### New Issues (Regressions)")
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(out)

	return err
}

func writeDiffIssue(out io.Writer, issue report.IssueDetail, isMarkdown bool) error {
	if isMarkdown {
		return writeMarkdownDiffIssue(out, issue)
	}

	return writePlainDiffIssue(out, issue)
}

func writeMarkdownDiffIssue(out io.Writer, issue report.IssueDetail) error {
	_, err := fmt.Fprintf(
		out,
		"- [%s] %s | scanner=%s | rule=%s | page=%s\n",
		issue.Severity,
		issue.Title,
		issue.Scanner,
		issue.RuleId,
		issue.PageUrl,
	)
	if err != nil {
		return err
	}

	selector := firstIssueSelector(issue)
	if selector == "" {
		return nil
	}

	_, err = fmt.Fprintf(out, "  - Selector: `%s`\n", selector)

	return err
}

func writePlainDiffIssue(out io.Writer, issue report.IssueDetail) error {
	_, err := fmt.Fprintf(
		out,
		"- [%s] %s (%s/%s)\n  %s\n",
		issue.Severity,
		issue.Title,
		issue.Scanner,
		issue.RuleId,
		issue.PageUrl,
	)
	if err != nil {
		return err
	}

	selector := firstIssueSelector(issue)
	if selector == "" {
		return nil
	}

	_, err = fmt.Fprintf(out, "  Selector: %s\n", selector)

	return err
}

func firstIssueSelector(issue report.IssueDetail) string {
	if len(issue.Occurrences) == 0 {
		return ""
	}

	selector := issue.Occurrences[0].Selector
	if selector == nil {
		return ""
	}

	return *selector
}

func loadReportFile(path string) (reportEnvelope, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return reportEnvelope{}, fmt.Errorf("read %s: %w", path, err)
	}

	data = sanitizeScoreGrade(data)

	var env reportEnvelope

	err = json.Unmarshal(data, &env)
	if err != nil {
		return reportEnvelope{}, fmt.Errorf("parse %s: %w", path, err)
	}

	return env, nil
}
