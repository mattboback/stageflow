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
		rawScanners    string
		screenshot     = true
		allowPrivate   bool
		timeout        time.Duration
		noStream       bool
		projectSlug    string
		authStatePath  string
		authRecipePath string
		reportOpts     reportCommandOptions
	)

	cmd := &cobra.Command{
		Use:                   "scan [url...] [--project <slug>]",
		Short:                 "Submit a scan job and wait for results",
		DisableFlagsInUseLine: true,
		Args:                  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := scanCommandOptions{
				rawScanners:    rawScanners,
				screenshot:     screenshot,
				allowPrivate:   allowPrivate,
				timeout:        timeout,
				noStream:       noStream,
				projectSlug:    projectSlug,
				authStatePath:  authStatePath,
				authRecipePath: authRecipePath,
				reportOpts:     reportOpts,
			}

			return runScanCmd(cmd, root, opts, args)
		},
	}

	cmd.Flags().StringVar(&rawScanners, "scanners", defaultScanScanners, "Comma-separated scanner modules")
	cmd.Flags().BoolVar(&screenshot, "screenshot", true, "Capture screenshots")
	cmd.Flags().BoolVar(
		&allowPrivate,
		"allow-private-targets",
		false,
		"Allow private/loopback targets (requires API instance to permit it)",
	)
	cmd.Flags().DurationVar(&timeout, "timeout", 5*time.Minute, "Max wait time")
	cmd.Flags().StringVar(&projectSlug, "project", "", "Scan using a registered project's URLs and config")
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
	bindReportFlags(cmd, &reportOpts, true)
	cmd.Flags().BoolVar(&noStream, "no-stream", false, "Poll instead of SSE")
	cobra.CheckErr(cmd.Flags().MarkHidden("no-stream"))

	return cmd
}

type scanCommandOptions struct {
	rawScanners    string
	screenshot     bool
	allowPrivate   bool
	timeout        time.Duration
	noStream       bool
	projectSlug    string
	authStatePath  string
	authRecipePath string
	reportOpts     reportCommandOptions
}

func runScanCmd(cmd *cobra.Command, root *rootOptions, opts scanCommandOptions, args []string) error {
	if opts.projectSlug != "" {
		return runScanProjectCmd(cmd, root, opts)
	}

	return runScanURLCmd(cmd, root, opts, args)
}

func runScanProjectCmd(cmd *cobra.Command, root *rootOptions, opts scanCommandOptions) error {
	if opts.authStatePath != "" || opts.authRecipePath != "" {
		return exitCodeError{
			Code: 2,
			Err: errors.New(
				"--auth-state and --auth-recipe are not supported with --project " +
					"(configure auth on the remote project instead)",
			),
		}
	}

	return runRemoteProjectScan(cmd, root, opts.projectSlug, opts.timeout, opts.noStream, opts.reportOpts)
}

func runScanURLCmd(cmd *cobra.Command, root *rootOptions, opts scanCommandOptions, args []string) error {
	if len(args) == 0 {
		return exitCodeError{Code: 2, Err: errors.New("at least one URL is required (or use --project)")}
	}

	if opts.reportOpts.maxIssues < 0 {
		return exitCodeError{Code: 2, Err: errors.New("--max-issues must be >= 0")}
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
		return exitCodeError{Code: 2, Err: err}
	}

	format, err := root.outputFormat()
	if err != nil {
		return exitCodeError{Code: 2, Err: err}
	}

	return wrapRenderError(
		renderUnifiedReport(cmd.OutOrStdout(), root.apiURL, status, doc, opts.reportOpts.renderOptions(format)),
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
		return apiclient.SubmitJobRequest{}, exitCodeError{Code: 2, Err: err}
	}

	modules, err := urlcheck.ParseModules(opts.rawScanners)
	if err != nil {
		return apiclient.SubmitJobRequest{}, exitCodeError{Code: 2, Err: err}
	}

	validateErr := urlcheck.ValidateLocalTargets(root.apiURL, urls)
	if validateErr != nil {
		return apiclient.SubmitJobRequest{}, exitCodeError{Code: 2, Err: validateErr}
	}

	authInput, hasAuth, authErr := loadAuthInputFromFlags(opts.authStatePath, opts.authRecipePath)
	if authErr != nil {
		return apiclient.SubmitJobRequest{}, exitCodeError{Code: 2, Err: authErr}
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

// loadAuthInputFromFlags resolves --auth-state or --auth-recipe into a
// JobAuthInput payload. Mutual exclusivity is enforced upstream by Cobra; this
// function additionally guards against both being set defensively.
func loadAuthInputFromFlags(authStatePath, authRecipePath string) (*apiclient.JobAuthInput, bool, error) {
	if authStatePath != "" && authRecipePath != "" {
		return nil, false, errors.New("--auth-state and --auth-recipe are mutually exclusive")
	}

	if authStatePath != "" {
		input, err := loadAuthStateFile(authStatePath)
		return input, err == nil, err
	}

	if authRecipePath != "" {
		input, err := loadAuthRecipeFile(authRecipePath)
		return input, err == nil, err
	}

	return nil, false, nil
}
