package main

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
	"github.com/mattboback/stageflow/clients/cli/internal/exitcode"
	"github.com/mattboback/stageflow/clients/cli/internal/render"
)

func newReportCmd(root *rootOptions) *cobra.Command {
	var reportOpts render.ReportFlags

	cmd := &cobra.Command{
		Use:                   "report <job-id>",
		Short:                 "Fetch and display results for an existing job",
		DisableFlagsInUseLine: true,
		Args:                  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if reportOpts.MaxIssues < 0 {
				return exitcode.Error{Code: 2, Err: errors.New("--max-issues must be >= 0")}
			}

			client := newAPICommandClient(root)
			jobID := args[0]

			status, err := render.FetchJobStatus(cmd.Context(), client, jobID)
			if err != nil {
				return exitcode.Error{Code: 2, Err: fmt.Errorf("fetch job status: %w", err)}
			}

			switch status.State {
			case apiclient.JobStateDone:
			case apiclient.JobStateFailed:
				return exitcode.Error{Code: 2, Err: fmt.Errorf("job failed: %s", status.Error)}
			default:
				return exitcode.Error{Code: 2, Err: fmt.Errorf("job is not completed yet: %s", status.State)}
			}

			doc, err := render.FetchReport(cmd.Context(), client, jobID)
			if err != nil {
				return exitcode.Error{Code: 2, Err: fmt.Errorf("fetch report: %w", err)}
			}

			format, err := root.renderFormat()
			if err != nil {
				return exitcode.Error{Code: 2, Err: err}
			}

			return render.WrapError(
				render.UnifiedReport(cmd.OutOrStdout(), root.apiURL, status, doc, reportOpts.RenderOptions(format)),
			)
		},
	}
	render.BindReportFlags(cmd, &reportOpts, false)

	return cmd
}
