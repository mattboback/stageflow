package main

import (
	"errors"

	"github.com/spf13/cobra"
)

type reportCommandOptions struct {
	maxIssues      int
	maxOccurrences int
	rawSeverities  string
	rawCategories  string
	failSeverity   string
	summaryOnly    bool
	groupBy        string
}

func (o reportCommandOptions) renderOptions(format outputFormat) reportRenderOptions {
	return reportRenderOptions{
		Format:         format,
		MaxIssues:      o.maxIssues,
		MaxOccurrences: o.maxOccurrences,
		Severities:     splitCSV(o.rawSeverities),
		Categories:     splitCSV(o.rawCategories),
		FailSeverity:   o.failSeverity,
		SummaryOnly:    o.summaryOnly,
		GroupBy:        o.groupBy,
	}
}

func bindReportFlags(cmd *cobra.Command, opts *reportCommandOptions, hideAdvanced bool) {
	cmd.Flags().
		IntVar(&opts.maxIssues, "max-issues", defaultMaxIssues, "Max issues to include in output (0 = unlimited)")
	cmd.Flags().IntVar(
		&opts.maxOccurrences,
		"max-occurrences",
		defaultMaxOccurrences,
		"Max occurrences per issue to display (0 = unlimited)",
	)
	cmd.Flags().StringVar(
		&opts.rawSeverities,
		"severity",
		"",
		"Filter displayed findings by severity (comma-separated: critical,serious,moderate,minor,info)",
	)
	cmd.Flags().StringVar(
		&opts.rawCategories,
		"category",
		"",
		"Filter displayed findings by category (comma-separated: accessibility,performance,seo,security,best-practices)",
	)
	cmd.Flags().StringVar(
		&opts.failSeverity,
		"fail-on",
		"",
		"Exit 1 if any displayed issue is at or above this severity (critical,serious,moderate,minor,info)",
	)
	cmd.Flags().StringVar(&opts.failSeverity, "fail-severity", "", "Alias for --fail-on")
	cmd.Flags().BoolVar(&opts.summaryOnly, "summary-only", false, "Only show summary, skip detailed findings")
	cmd.Flags().StringVar(
		&opts.groupBy,
		"group-by",
		"",
		"Group findings by: none, category, scanner (default: category for markdown, none for text)",
	)
	cobra.CheckErr(cmd.Flags().MarkHidden("fail-severity"))

	if !hideAdvanced {
		return
	}

	cobra.CheckErr(cmd.Flags().MarkHidden("max-occurrences"))
	cobra.CheckErr(cmd.Flags().MarkHidden("severity"))
	cobra.CheckErr(cmd.Flags().MarkHidden("category"))
	cobra.CheckErr(cmd.Flags().MarkHidden("summary-only"))
	cobra.CheckErr(cmd.Flags().MarkHidden("group-by"))
}

func wrapRenderError(err error) error {
	if err == nil {
		return nil
	}

	var ece exitCodeError
	if errors.As(err, &ece) && ece.Code == 1 {
		return ece
	}

	return exitCodeError{Code: 2, Err: err}
}
