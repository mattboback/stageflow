package main

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

func newReportCmd(root *rootOptions) *cobra.Command {
	var reportOpts reportCommandOptions

	cmd := &cobra.Command{
		Use:                   "report <job-id>",
		Short:                 "Fetch and display results for an existing job",
		DisableFlagsInUseLine: true,
		Args:                  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if reportOpts.maxIssues < 0 {
				return exitCodeError{Code: 2, Err: errors.New("--max-issues must be >= 0")}
			}

			client := newAPICommandClient(root)
			jobID := args[0]

			status, err := fetchJobStatus(cmd.Context(), client, jobID)
			if err != nil {
				return exitCodeError{Code: 2, Err: fmt.Errorf("fetch job status: %w", err)}
			}

			switch status.State {
			case jobStateDone:
			case jobStateFailed:
				return exitCodeError{Code: 2, Err: fmt.Errorf("job failed: %s", status.Error)}
			default:
				return exitCodeError{Code: 2, Err: fmt.Errorf("job is not completed yet: %s", status.State)}
			}

			doc, err := fetchReport(cmd.Context(), client, jobID)
			if err != nil {
				return exitCodeError{Code: 2, Err: fmt.Errorf("fetch report: %w", err)}
			}

			format, err := root.outputFormat()
			if err != nil {
				return exitCodeError{Code: 2, Err: err}
			}

			return wrapRenderError(
				renderUnifiedReport(cmd.OutOrStdout(), root.apiURL, status, doc, reportOpts.renderOptions(format)),
			)
		},
	}
	bindReportFlags(cmd, &reportOpts, false)

	return cmd
}
