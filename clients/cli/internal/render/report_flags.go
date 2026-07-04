package render

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/mattboback/stageflow/clients/cli/internal/exitcode"
)

// Defaults for issue selection in rendered reports.
const (
	DefaultMaxIssues      = 200
	DefaultMaxOccurrences = 3
)

type ReportFlags struct {
	MaxIssues      int
	MaxOccurrences int
	RawSeverities  string
	RawCategories  string
	FailSeverity   string
	SummaryOnly    bool
	GroupBy        string
}

func (o ReportFlags) RenderOptions(format Format) Options {
	return Options{
		Format:         format,
		MaxIssues:      o.MaxIssues,
		MaxOccurrences: o.MaxOccurrences,
		Severities:     splitCSV(o.RawSeverities),
		Categories:     splitCSV(o.RawCategories),
		FailSeverity:   o.FailSeverity,
		SummaryOnly:    o.SummaryOnly,
		GroupBy:        o.GroupBy,
	}
}

func BindReportFlags(cmd *cobra.Command, opts *ReportFlags, hideAdvanced bool) {
	cmd.Flags().
		IntVar(&opts.MaxIssues, "max-issues", DefaultMaxIssues, "Max issues to include in output (0 = unlimited)")
	cmd.Flags().IntVar(
		&opts.MaxOccurrences,
		"max-occurrences",
		DefaultMaxOccurrences,
		"Max occurrences per issue to display (0 = unlimited)",
	)
	cmd.Flags().StringVar(
		&opts.RawSeverities,
		"severity",
		"",
		"Filter displayed findings by severity (comma-separated: critical,serious,moderate,minor,info)",
	)
	cmd.Flags().StringVar(
		&opts.RawCategories,
		"category",
		"",
		"Filter displayed findings by category (comma-separated: accessibility,performance,seo,security,best-practices)",
	)
	cmd.Flags().StringVar(
		&opts.FailSeverity,
		"fail-on",
		"",
		"Exit 1 if any displayed issue is at or above this severity (critical,serious,moderate,minor,info)",
	)
	cmd.Flags().BoolVar(&opts.SummaryOnly, "summary-only", false, "Only show summary, skip detailed findings")
	cmd.Flags().StringVar(
		&opts.GroupBy,
		"group-by",
		"",
		"Group findings by: none, category, scanner (default: category for markdown, none for text)",
	)

	if !hideAdvanced {
		return
	}

	cobra.CheckErr(cmd.Flags().MarkHidden("max-occurrences"))
	cobra.CheckErr(cmd.Flags().MarkHidden("severity"))
	cobra.CheckErr(cmd.Flags().MarkHidden("category"))
	cobra.CheckErr(cmd.Flags().MarkHidden("summary-only"))
	cobra.CheckErr(cmd.Flags().MarkHidden("group-by"))
}

func WrapError(err error) error {
	if err == nil {
		return nil
	}

	var ece exitcode.Error
	if errors.As(err, &ece) && ece.Code == 1 {
		return ece
	}

	return exitcode.Error{Code: 2, Err: err}
}
