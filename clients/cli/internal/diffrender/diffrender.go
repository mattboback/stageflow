// Package diffrender owns the CLI-facing Diff envelope types along with all
// rendering and pure regression evaluation logic used by `stageflow diff`.
package diffrender

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
	"github.com/mattboback/stageflow/libs/go/diff"
)

// Envelope is the user-visible JSON shape for a diff result.
type Envelope struct {
	Schema    string               `json:"schema"`
	Baseline  BaselineMeta         `json:"baseline"`
	Current   CurrentMeta          `json:"current"`
	Delta     Delta                `json:"delta"`
	New       []report.IssueDetail `json:"new"`
	Fixed     []report.IssueDetail `json:"fixed"`
	Regressed bool                 `json:"regressed"`
	// SkippedScanners did not finish in one of the reports and were not
	// compared.
	SkippedScanners      []string `json:"skippedScanners,omitempty"`
	IncompatibleIdentity bool     `json:"incompatibleIdentity,omitempty"`
}

type BaselineMeta struct {
	File        string `json:"file"`
	Score       *int   `json:"score,omitempty"`
	TotalIssues int    `json:"totalIssues"`
}

type CurrentMeta struct {
	JobID       string `json:"jobId,omitempty"`
	File        string `json:"file,omitempty"`
	Score       *int   `json:"score,omitempty"`
	TotalIssues int    `json:"totalIssues"`
}

type Delta struct {
	ScoreDelta      *int `json:"scoreDelta,omitempty"`
	NewIssues       int  `json:"newIssues"`
	FixedIssues     int  `json:"fixedIssues"`
	UnchangedIssues int  `json:"unchangedIssues"`
}

// Format selects a rendering style.
type Format int

const (
	FormatJSON Format = iota
	FormatText
	FormatMarkdown
)

// FromResult converts a lib-level diff.Result into the CLI envelope.
func FromResult(r diff.Result, baselineFile, currentFile string) Envelope {
	return Envelope{
		Schema: r.Schema,
		Baseline: BaselineMeta{
			File:        baselineFile,
			Score:       r.Baseline.Score,
			TotalIssues: r.Baseline.TotalIssues,
		},
		Current: CurrentMeta{
			JobID:       r.Current.JobID,
			File:        currentFile,
			Score:       r.Current.Score,
			TotalIssues: r.Current.TotalIssues,
		},
		Delta: Delta{
			ScoreDelta:      r.Delta.ScoreDelta,
			NewIssues:       r.Delta.NewIssues,
			FixedIssues:     r.Delta.FixedIssues,
			UnchangedIssues: r.Delta.UnchangedIssues,
		},
		New:                  r.New,
		Fixed:                r.Fixed,
		SkippedScanners:      r.SkippedScanners,
		IncompatibleIdentity: r.IncompatibleIdentity,
	}
}

// IsRegressed reports whether a diff envelope indicates the score dropped or
// new issues appeared.
func IsRegressed(e Envelope) bool {
	return (e.Delta.ScoreDelta != nil && *e.Delta.ScoreDelta < 0) || e.Delta.NewIssues > 0
}

// SeverityChecker abstracts severity gating so callers can wire in their own
// implementation without dragging it into this package.
type SeverityChecker func(issues []report.IssueDetail, minSeverity string) (bool, error)

// EvaluateRegression applies the --fail-on-regression and --fail-on-new flags
// to a diff envelope, returning whether the CLI should exit nonzero.
func EvaluateRegression(
	e Envelope,
	failOnRegression bool,
	failOnNew string,
	hasSeverity SeverityChecker,
) (bool, error) {
	regressed := failOnRegression && IsRegressed(e)

	if failOnNew == "" || len(e.New) == 0 {
		return regressed, nil
	}

	if failOnNew == "any" {
		return true, nil
	}

	if hasSeverity == nil {
		return false, errors.New("fail-on-new severity gate requires a severity checker")
	}

	flagged, err := hasSeverity(e.New, failOnNew)
	if err != nil {
		return false, err
	}

	return regressed || flagged, nil
}

// Render writes the diff envelope to out in the requested format.
func Render(out io.Writer, env Envelope, format Format) error {
	switch format {
	case FormatJSON:
		return writeJSON(out, env)
	case FormatText, FormatMarkdown:
		return writeText(out, env, format == FormatMarkdown)
	default:
		return errors.New("unsupported output format")
	}
}

func writeJSON(out io.Writer, env Envelope) error {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)

	return encoder.Encode(env)
}

func writeText(out io.Writer, env Envelope, isMarkdown bool) error {
	if err := writeHeading(out, isMarkdown); err != nil {
		return err
	}

	if env.IncompatibleIdentity {
		line := "Baseline identity format is incompatible. Review a fresh scan and save it as the replacement baseline."
		if isMarkdown {
			line = "> " + line
		}

		_, err := fmt.Fprintln(out, line)

		return err
	}

	if err := writeSummary(out, env, isMarkdown); err != nil {
		return err
	}

	if err := writeSkippedScanners(out, env.SkippedScanners, isMarkdown); err != nil {
		return err
	}

	return writeNewIssueSection(out, env.New, isMarkdown)
}

func writeSkippedScanners(out io.Writer, skipped []string, isMarkdown bool) error {
	if len(skipped) == 0 {
		return nil
	}

	line := fmt.Sprintf(
		"Not compared: %s did not finish in one of the reports, so its findings are left out.",
		strings.Join(skipped, ", "),
	)
	if isMarkdown {
		line = "> " + line
	}

	_, err := fmt.Fprintf(out, "%s\n\n", line)

	return err
}

func writeHeading(out io.Writer, isMarkdown bool) error {
	if !isMarkdown {
		return nil
	}

	if _, err := fmt.Fprintln(out, "## Regression Diff"); err != nil {
		return err
	}

	_, err := fmt.Fprintln(out)

	return err
}

func writeSummary(out io.Writer, env Envelope, isMarkdown bool) error {
	scoreStr := formatScore(env)

	if isMarkdown {
		_, err := fmt.Fprintf(
			out,
			"- **Score**: %s\n- **New issues**: %d\n- **Fixed issues**: %d\n- **Unchanged**: %d\n\n",
			scoreStr,
			env.Delta.NewIssues,
			env.Delta.FixedIssues,
			env.Delta.UnchangedIssues,
		)

		return err
	}

	_, err := fmt.Fprintf(
		out,
		"Score: %s\nNew issues: %d\nFixed issues: %d\nUnchanged: %d\n\n",
		scoreStr,
		env.Delta.NewIssues,
		env.Delta.FixedIssues,
		env.Delta.UnchangedIssues,
	)

	return err
}

func formatScore(env Envelope) string {
	if len(env.SkippedScanners) > 0 || env.IncompatibleIdentity || env.Delta.ScoreDelta == nil ||
		env.Baseline.Score == nil ||
		env.Current.Score == nil {
		return "N/A"
	}

	delta := *env.Delta.ScoreDelta
	sign := ""

	if delta > 0 {
		sign = "+"
	}

	return fmt.Sprintf("%d → %d (%s%d)", *env.Baseline.Score, *env.Current.Score, sign, delta)
}

func writeNewIssueSection(out io.Writer, issues []report.IssueDetail, isMarkdown bool) error {
	if len(issues) == 0 {
		_, err := fmt.Fprintln(out, "No new issues detected.")

		return err
	}

	if err := writeNewIssueHeader(out, isMarkdown); err != nil {
		return err
	}

	for _, issue := range issues {
		if err := writeIssue(out, issue, isMarkdown); err != nil {
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

	if _, err := fmt.Fprintln(out, "### New Issues (Regressions)"); err != nil {
		return err
	}

	_, err := fmt.Fprintln(out)

	return err
}

func writeIssue(out io.Writer, issue report.IssueDetail, isMarkdown bool) error {
	if isMarkdown {
		return writeMarkdownIssue(out, issue)
	}

	return writePlainIssue(out, issue)
}

func writeMarkdownIssue(out io.Writer, issue report.IssueDetail) error {
	if _, err := fmt.Fprintf(
		out,
		"- [%s] %s | scanner=%s | rule=%s | page=%s\n",
		issue.Severity,
		issue.Title,
		issue.Scanner,
		issue.RuleId,
		issue.PageUrl,
	); err != nil {
		return err
	}

	selector := firstSelector(issue)
	if selector == "" {
		return nil
	}

	_, err := fmt.Fprintf(out, "  - Selector: `%s`\n", selector)

	return err
}

func writePlainIssue(out io.Writer, issue report.IssueDetail) error {
	if _, err := fmt.Fprintf(
		out,
		"- [%s] %s (%s/%s)\n  %s\n",
		issue.Severity,
		issue.Title,
		issue.Scanner,
		issue.RuleId,
		issue.PageUrl,
	); err != nil {
		return err
	}

	selector := firstSelector(issue)
	if selector == "" {
		return nil
	}

	_, err := fmt.Fprintf(out, "  Selector: %s\n", selector)

	return err
}

func firstSelector(issue report.IssueDetail) string {
	if len(issue.Occurrences) == 0 {
		return ""
	}

	selector := issue.Occurrences[0].Selector
	if selector == nil {
		return ""
	}

	return *selector
}
