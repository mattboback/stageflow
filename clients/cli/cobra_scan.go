package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
	"github.com/mattboback/stageflow/clients/cli/internal/urlcheck"
)

func newScanCmd(root *rootOptions) *cobra.Command {
	var (
		rawScanners  string
		screenshot   bool
		allowPrivate bool
		timeout      time.Duration
		noStream     bool
		projectSlug  string
		reportOpts   reportCommandOptions
	)

	cmd := &cobra.Command{
		Use:                   "scan [url...] [--project <slug>]",
		Short:                 "Submit a scan job and wait for results",
		DisableFlagsInUseLine: true,
		Args:                  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectSlug != "" {
				return runRemoteProjectScan(cmd, root, projectSlug, timeout, noStream, reportOpts)
			}

			if len(args) == 0 {
				return exitCodeError{Code: 2, Err: errors.New("at least one URL is required (or use --project)")}
			}

			if reportOpts.maxIssues < 0 {
				return exitCodeError{Code: 2, Err: errors.New("--max-issues must be >= 0")}
			}

			urls, err := urlcheck.NormalizeTargets(args)
			if err != nil {
				return exitCodeError{Code: 2, Err: err}
			}

			modules, err := urlcheck.ParseModules(rawScanners)
			if err != nil {
				return exitCodeError{Code: 2, Err: err}
			}

			validateErr := urlcheck.ValidateLocalTargets(root.apiURL, urls)
			if validateErr != nil {
				return exitCodeError{Code: 2, Err: validateErr}
			}

			effectiveAllowPrivate := allowPrivate
			if !cobraFlagChanged(cmd, "allow-private-targets") && urlcheck.ContainsPrivateTargets(urls) {
				effectiveAllowPrivate = true

				fmt.Fprintln(
					cmd.ErrOrStderr(),
					"Detected private/loopback targets; setting allow_private_targets=true.",
				)
			}

			client := apiclient.NewClient(root.apiURL, root.apiKey, nil)

			status, doc, err := runScanJob(
				cmd.Context(),
				client,
				apiclient.SubmitJobRequest{
					URLs:                urls,
					Modules:             modules,
					Screenshot:          screenshot,
					AllowPrivateTargets: effectiveAllowPrivate,
				},
				timeout,
				cmd.ErrOrStderr(),
				noStream,
			)
			if err != nil {
				return exitCodeError{Code: 2, Err: err}
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

	cmd.Flags().StringVar(&rawScanners, "scanners", defaultScanScanners, "Comma-separated scanner modules")
	cmd.Flags().BoolVar(&screenshot, "screenshot", false, "Capture screenshots")
	cmd.Flags().BoolVar(
		&allowPrivate,
		"allow-private-targets",
		false,
		"Allow private/loopback targets (requires API instance to permit it)",
	)
	cmd.Flags().DurationVar(&timeout, "timeout", 5*time.Minute, "Max wait time")
	cmd.Flags().StringVar(&projectSlug, "project", "", "Scan using a registered project's URLs and config")
	bindReportFlags(cmd, &reportOpts, true)
	cmd.Flags().BoolVar(&noStream, "no-stream", false, "Poll instead of SSE")
	cobra.CheckErr(cmd.Flags().MarkHidden("no-stream"))

	return cmd
}
