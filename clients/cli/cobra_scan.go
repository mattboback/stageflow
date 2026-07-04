package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
	"github.com/mattboback/stageflow/clients/cli/internal/authintake"
	"github.com/mattboback/stageflow/clients/cli/internal/exitcode"
	"github.com/mattboback/stageflow/clients/cli/internal/projectmode"
	"github.com/mattboback/stageflow/clients/cli/internal/render"
	"github.com/mattboback/stageflow/clients/cli/internal/urlcheck"
)

func newScanCmd(root *rootOptions) *cobra.Command {
	var (
		scanners       []string
		screenshot     = true
		allowPrivate   bool
		timeout        time.Duration
		noStream       bool
		authStatePath  string
		authRecipePath string
		reportOpts     render.ReportFlags
	)

	cmd := &cobra.Command{
		Use:                   "scan <url>...",
		Short:                 "Scan one or more URLs and report the results",
		DisableFlagsInUseLine: true,
		Args:                  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := scanCommandOptions{
				scanners:       scanners,
				screenshot:     screenshot,
				allowPrivate:   allowPrivate,
				timeout:        timeout,
				noStream:       noStream,
				authStatePath:  authStatePath,
				authRecipePath: authRecipePath,
				reportOpts:     reportOpts,
			}

			return runScanCmd(cmd, root, opts, args)
		},
	}

	cmd.Flags().StringSliceVar(
		&scanners,
		"scanner",
		projectmode.DefaultScanScannerList(),
		"Scanner module (repeatable or comma-separated)",
	)
	cmd.Flags().BoolVar(&screenshot, "screenshot", true, "Capture screenshots")
	cmd.Flags().BoolVar(
		&allowPrivate,
		"allow-private-targets",
		false,
		"Allow private/loopback targets (requires API instance to permit it)",
	)
	cmd.Flags().DurationVar(&timeout, "timeout", 5*time.Minute, "Max wait time")
	cmd.Flags().StringVar(
		&authStatePath,
		"auth-state",
		"",
		"Path to a Playwright storage-state JSON captured via stageflow auth capture. "+
			"Uploaded under the job's MinIO prefix and referenced from Provenance.auth.",
	)
	cmd.Flags().StringVar(
		&authRecipePath,
		"auth-recipe",
		"",
		"Path to a YAML/JSON form-auth recipe (Provenance.auth.form shape). "+
			"Step values may be literal strings or {from_env: NAME} references.",
	)
	cmd.MarkFlagsMutuallyExclusive("auth-state", "auth-recipe")
	render.BindReportFlags(cmd, &reportOpts, true)
	cmd.Flags().BoolVar(&noStream, "no-stream", false, "Poll instead of SSE")
	cobra.CheckErr(cmd.Flags().MarkHidden("no-stream"))

	return cmd
}

type scanCommandOptions struct {
	scanners       []string
	screenshot     bool
	allowPrivate   bool
	timeout        time.Duration
	noStream       bool
	authStatePath  string
	authRecipePath string
	reportOpts     render.ReportFlags
}

func runScanCmd(cmd *cobra.Command, root *rootOptions, opts scanCommandOptions, args []string) error {
	if opts.reportOpts.MaxIssues < 0 {
		return exitcode.Error{Code: 2, Err: errors.New("--max-issues must be >= 0")}
	}

	req, err := buildScanRequest(cmd, root, opts, args)
	if err != nil {
		return err
	}

	client := apiclient.NewClient(root.apiURL, root.apiKey, nil)

	status, doc, err := runScanJob(
		cmd.Context(),
		client,
		req,
		opts.timeout,
		cmd.ErrOrStderr(),
		opts.noStream,
	)
	if err != nil {
		return exitcode.Error{Code: 2, Err: err}
	}

	format, err := root.renderFormat()
	if err != nil {
		return exitcode.Error{Code: 2, Err: err}
	}

	return render.WrapError(
		render.UnifiedReport(cmd.OutOrStdout(), root.apiURL, status, doc, opts.reportOpts.RenderOptions(format)),
	)
}

func buildScanRequest(
	cmd *cobra.Command,
	root *rootOptions,
	opts scanCommandOptions,
	args []string,
) (apiclient.SubmitJobRequest, error) {
	urls, err := urlcheck.NormalizeTargets(args)
	if err != nil {
		return apiclient.SubmitJobRequest{}, exitcode.Error{Code: 2, Err: err}
	}

	modules, err := normalizeScannerList(opts.scanners)
	if err != nil {
		return apiclient.SubmitJobRequest{}, exitcode.Error{Code: 2, Err: err}
	}

	validateErr := urlcheck.ValidateLocalTargets(root.apiURL, urls)
	if validateErr != nil {
		return apiclient.SubmitJobRequest{}, exitcode.Error{Code: 2, Err: validateErr}
	}

	authInput, hasAuth, authErr := authintake.LoadFromFlags(opts.authStatePath, opts.authRecipePath)
	if authErr != nil {
		return apiclient.SubmitJobRequest{}, exitcode.Error{Code: 2, Err: authErr}
	}

	req := apiclient.SubmitJobRequest{
		URLs:                urls,
		Modules:             modules,
		Screenshot:          opts.screenshot,
		AllowPrivateTargets: effectiveAllowPrivateTargets(cmd, urls, opts.allowPrivate),
	}
	if hasAuth {
		req.Auth = authInput

		fmt.Fprintf(
			cmd.ErrOrStderr(),
			"Attaching auth block (mode=%s) to scan submission.\n",
			authInput.Mode,
		)
	}

	return req, nil
}

func effectiveAllowPrivateTargets(cmd *cobra.Command, urls []string, allowPrivate bool) bool {
	if allowPrivate || cobraFlagChanged(cmd, "allow-private-targets") || !urlcheck.ContainsPrivateTargets(urls) {
		return allowPrivate
	}

	fmt.Fprintln(
		cmd.ErrOrStderr(),
		"Detected private/loopback targets; setting allow_private_targets=true.",
	)

	return true
}
