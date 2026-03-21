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
			baselinePath := args[0]
			currentTarget := args[1]

			baselineEnv, err := loadReportFile(baselinePath)
			if err != nil {
				return exitCodeError{Code: 2, Err: fmt.Errorf("baseline: %w", err)}
			}

			var currentEnv reportEnvelope
			var currentJobID string
			var currentFile string

			if strings.HasPrefix(currentTarget, "http://") || strings.HasPrefix(currentTarget, "https://") {
				client := NewClient(root.apiURL, root.apiKey, nil)

				var modules []string
				for _, sc := range baselineEnv.Report.Scanners {
					modules = append(modules, sc.Id)
				}

				req := SubmitJobRequest{
					URLs:                []string{currentTarget},
					Modules:             modules,
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
					return exitCodeError{Code: 2, Err: err}
				}

				currentEnv = reportEnvelope{
					Job:    jobMeta{ID: status.ID},
					Report: doc,
				}
				currentJobID = status.ID
			} else {
				currentFile = currentTarget
				loaded, err := loadReportFile(currentTarget)
				if err != nil {
					return exitCodeError{Code: 2, Err: fmt.Errorf("current: %w", err)}
				}
				currentEnv = loaded
			}

			diff := computeDiff(baselinePath, baselineEnv.Report, currentJobID, currentFile, currentEnv.Report)

			regressed := false
			if failOnRegression {
				if (diff.Delta.ScoreDelta != nil && *diff.Delta.ScoreDelta < 0) || diff.Delta.NewIssues > 0 {
					regressed = true
				}
			}

			if failOnNew != "" && len(diff.New) > 0 {
				if failOnNew == "any" {
					regressed = true
				} else {
					hasSeverity, err := hasIssuesAtOrAbove(diff.New, failOnNew)
					if err != nil {
						return exitCodeError{Code: 2, Err: err}
					}
					if hasSeverity {
						regressed = true
					}
				}
			}

			diff.Regressed = regressed

			format, err := root.outputFormat()
			if err != nil {
				return exitCodeError{Code: 2, Err: err}
			}

			if err := renderDiff(cmd.OutOrStdout(), diff, format); err != nil {
				return exitCodeError{Code: 2, Err: err}
			}

			if regressed {
				return exitCodeError{Code: 1}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&failOnNew, "fail-on-new", "", "Exit 1 if any NEW issue meets threshold (critical, serious, moderate, minor, info) or 'any'")
	cmd.Flags().Lookup("fail-on-new").NoOptDefVal = "any"
	cmd.Flags().BoolVar(&failOnRegression, "fail-on-regression", false, "Exit 1 if score dropped or new issues appeared")
	cmd.Flags().DurationVar(&timeout, "timeout", 5*time.Minute, "Max wait time for live scan")
	cmd.Flags().BoolVar(&noStream, "no-stream", false, "Poll instead of SSE for live scan")
	cobra.CheckErr(cmd.Flags().MarkHidden("no-stream"))

	return cmd
}

func computeDiff(
	baselineFile string, baseline report.UnifiedReportV2,
	currentJobID, currentFile string, current report.UnifiedReportV2,
) diffEnvelope {
	baselineMap := make(map[string]report.IssueDetail)
	for _, issue := range baseline.Issues {
		baselineMap[issue.Id] = issue
	}

	currentMap := make(map[string]report.IssueDetail)
	for _, issue := range current.Issues {
		currentMap[issue.Id] = issue
	}

	newIssues := make([]report.IssueDetail, 0)
	fixedIssues := make([]report.IssueDetail, 0)
	var unchangedCount int

	for id, issue := range currentMap {
		if _, exists := baselineMap[id]; !exists {
			newIssues = append(newIssues, issue)
		} else {
			unchangedCount++
		}
	}

	for id, issue := range baselineMap {
		if _, exists := currentMap[id]; !exists {
			fixedIssues = append(fixedIssues, issue)
		}
	}

	var scoreDelta *int
	if baseline.Summary.Score != nil && current.Summary.Score != nil {
		delta := *current.Summary.Score - *baseline.Summary.Score
		scoreDelta = &delta
	}

	return diffEnvelope{
		Schema: "stageflow-cli/diff@v1",
		Baseline: diffBaselineMeta{
			File:        baselineFile,
			Score:       baseline.Summary.Score,
			TotalIssues: baseline.Summary.TotalIssues,
		},
		Current: diffCurrentMeta{
			JobID:       currentJobID,
			File:        currentFile,
			Score:       current.Summary.Score,
			TotalIssues: current.Summary.TotalIssues,
		},
		Delta: diffDelta{
			ScoreDelta:      scoreDelta,
			NewIssues:       len(newIssues),
			FixedIssues:     len(fixedIssues),
			UnchangedIssues: unchangedCount,
		},
		New:   newIssues,
		Fixed: fixedIssues,
	}
}

func renderDiff(out io.Writer, diff diffEnvelope, format outputFormat) error {
	switch format {
	case outputFormatJSON:
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		encoder.SetEscapeHTML(false)
		return encoder.Encode(diff)
	case outputFormatText, outputFormatMarkdown:
		return writeTextDiff(out, diff, format == outputFormatMarkdown)
	default:
		return fmt.Errorf("unsupported output format %q", format)
	}
}

func writeTextDiff(out io.Writer, diff diffEnvelope, isMarkdown bool) error {
	if isMarkdown {
		fmt.Fprintln(out, "## Regression Diff")
		fmt.Fprintln(out)
	}

	scoreStr := "N/A"
	if diff.Baseline.Score != nil && diff.Current.Score != nil {
		delta := *diff.Current.Score - *diff.Baseline.Score
		sign := ""
		if delta > 0 {
			sign = "+"
		}
		scoreStr = fmt.Sprintf("%d \u2192 %d (%s%d)", *diff.Baseline.Score, *diff.Current.Score, sign, delta)
	}

	if isMarkdown {
		fmt.Fprintf(out, "- **Score**: %s\n", scoreStr)
		fmt.Fprintf(out, "- **New issues**: %d\n", diff.Delta.NewIssues)
		fmt.Fprintf(out, "- **Fixed issues**: %d\n", diff.Delta.FixedIssues)
		fmt.Fprintf(out, "- **Unchanged**: %d\n\n", diff.Delta.UnchangedIssues)
	} else {
		fmt.Fprintf(out, "Score: %s\n", scoreStr)
		fmt.Fprintf(out, "New issues: %d\n", diff.Delta.NewIssues)
		fmt.Fprintf(out, "Fixed issues: %d\n", diff.Delta.FixedIssues)
		fmt.Fprintf(out, "Unchanged: %d\n\n", diff.Delta.UnchangedIssues)
	}

	if len(diff.New) > 0 {
		if isMarkdown {
			fmt.Fprintln(out, "### New Issues (Regressions)")
			fmt.Fprintln(out)
		} else {
			fmt.Fprintln(out, "New Issues (Regressions):")
		}

		for _, issue := range diff.New {
			if isMarkdown {
				fmt.Fprintf(out, "- [%s] %s | scanner=%s | rule=%s | page=%s\n",
					issue.Severity, issue.Title, issue.Scanner, issue.RuleId, issue.PageUrl)
				if len(issue.Occurrences) > 0 {
					occ := issue.Occurrences[0]
					if occ.Selector != nil && *occ.Selector != "" {
						fmt.Fprintf(out, "  - Selector: `%s`\n", *occ.Selector)
					}
				}
			} else {
				fmt.Fprintf(out, "- [%s] %s (%s/%s)\n  %s\n",
					issue.Severity, issue.Title, issue.Scanner, issue.RuleId, issue.PageUrl)
				if len(issue.Occurrences) > 0 {
					occ := issue.Occurrences[0]
					if occ.Selector != nil && *occ.Selector != "" {
						fmt.Fprintf(out, "  Selector: %s\n", *occ.Selector)
					}
				}
			}
		}
	} else {
		fmt.Fprintln(out, "No new issues detected.")
	}

	return nil
}

func loadReportFile(path string) (reportEnvelope, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return reportEnvelope{}, fmt.Errorf("read %s: %w", path, err)
	}

	data = sanitizeScoreGrade(data)

	var env reportEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return reportEnvelope{}, fmt.Errorf("parse %s: %w", path, err)
	}

	return env, nil
}
