package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
	"github.com/mattboback/stageflow/clients/cli/internal/diffrender"
	"github.com/mattboback/stageflow/clients/cli/internal/exitcode"
	"github.com/mattboback/stageflow/clients/cli/internal/render"
	"github.com/mattboback/stageflow/clients/cli/internal/scanflow"
	"github.com/mattboback/stageflow/libs/go/diff"
)

func newCheckCmd(root *rootOptions) *cobra.Command {
	var (
		projectPath, output string
		timeout             time.Duration
	)

	cmd := &cobra.Command{
		Use:   "check --project project.json --output run-directory",
		Short: "Repeat a browser project's public-page scan and compare its accepted baseline",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if projectPath == "" || output == "" {
				return exitcode.Error{Code: 2, Err: errors.New("--project and --output are required")}
			}

			if timeout <= 0 {
				return exitcode.Error{Code: 2, Err: errors.New("--timeout must be positive")}
			}

			err := runProjectCheck(cmd, root, projectPath, output, timeout)

			var coded exitcode.Error
			if err != nil && !errors.As(err, &coded) {
				return exitcode.Error{Code: 2, Err: err}
			}

			return err
		},
	}
	cmd.Flags().StringVar(&projectPath, "project", "", "Browser project export (stageflow/local-project@v2 or @v1)")
	cmd.Flags().StringVar(&output, "output", "", "New directory for report.json, summary.md and baseline comparisons")
	cmd.Flags().DurationVar(&timeout, "timeout", defaultScanWaitTimeout, "Max wait time including queue and scan")

	return cmd
}

func runProjectCheck(cmd *cobra.Command, root *rootOptions, projectPath, output string, timeout time.Duration) error {
	project, req, err := loadCheckProject(projectPath)
	if err != nil {
		return err
	}
	// Reserve a private directory before submitting; never overwrite a prior run.
	if err = os.Mkdir(output, 0o700); err != nil {
		return fmt.Errorf("create new output directory: %w", err)
	}

	result, err := scanflow.SubmitURLsAndWait(
		cmd.Context(),
		apiclient.NewClient(root.apiURL, root.apiKey, nil),
		req,
		timeout,
		scanflow.WaitOptions{Progress: cmd.ErrOrStderr()},
	)
	if err != nil {
		return err
	}

	for _, outputFile := range []struct {
		name   string
		format render.Format
	}{{"report.json", render.FormatJSON}, {"summary.md", render.FormatMarkdown}} {
		err = writeCheckFile(output, outputFile.name, func(w io.Writer) error {
			return render.UnifiedReport(
				w,
				root.apiURL,
				result.Status,
				result.Report,
				render.Options{Format: outputFile.format, GatePolicy: "actionable"},
			)
		})
		if err != nil {
			return err
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Saved complete report to %s\n", filepath.Join(output, "report.json"))

	incomplete, err := render.WarnIfPartial(cmd.ErrOrStderr(), result.Report)
	if err != nil {
		return err
	}

	if project.Baseline == nil {
		if !completeCheckCoverage(result.Report) {
			return errors.New("incomplete scan coverage: no baseline accepted")
		}

		if err = project.configuration.validateReport(result.Report); err != nil {
			return err
		}

		fmt.Fprintln(
			cmd.OutOrStdout(),
			"No accepted baseline; import report.json into the browser project to review this first run.",
		)

		return nil
	}

	return writeCheckComparison(cmd, project, projectPath, output, result, len(incomplete) > 0)
}

func writeCheckComparison(
	cmd *cobra.Command,
	project checkProject,
	projectPath, output string,
	result scanflow.Result,
	partial bool,
) error {
	comparison := diff.ComputeDiff(project.Baseline.JobID, project.Baseline.Report, result.Status.ID, result.Report)
	envelope := diffrender.FromResult(comparison, projectPath, filepath.Join(output, "report.json"))

	gate, err := actionableDiffGate("actionable", project.Baseline.Report, result.Report, envelope)
	if err != nil {
		return err
	}
	// check's regression field and exit policy both describe actionable findings.
	envelope.Regressed = diffrender.IsRegressed(gate)

	for _, outputFile := range []struct {
		name   string
		format diffrender.Format
	}{{"diff.json", diffrender.FormatJSON}, {"diff.md", diffrender.FormatMarkdown}} {
		err = writeCheckFile(
			output,
			outputFile.name,
			func(w io.Writer) error { return diffrender.Render(w, envelope, outputFile.format) },
		)
		if err != nil {
			return err
		}
	}

	fmt.Fprintln(
		cmd.OutOrStdout(),
		"Comparison saved with actionable regression policy; raw findings remain in the report and diff.",
	)

	if err = project.configuration.validateReport(result.Report); err != nil {
		return err
	}

	if partial || comparison.IncompatibleIdentity || comparison.ContextLimited ||
		len(comparison.SkippedScanners) > 0 ||
		len(comparison.NotComparedPages) > 0 {
		return errors.New("incomplete or incompatible comparison; inspect diff.json before accepting a new baseline")
	}

	if envelope.Regressed {
		return exitcode.Error{Code: 1}
	}

	return nil
}

func writeCheckFile(directory, name string, write func(io.Writer) error) error {
	file, err := os.OpenFile(filepath.Join(directory, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}

	writeErr := write(file)

	return errors.Join(writeErr, file.Close())
}
