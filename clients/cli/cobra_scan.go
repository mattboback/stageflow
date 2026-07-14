package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
	"github.com/mattboback/stageflow/clients/cli/internal/authintake"
	"github.com/mattboback/stageflow/clients/cli/internal/exitcode"
	"github.com/mattboback/stageflow/clients/cli/internal/projectmode"
	"github.com/mattboback/stageflow/clients/cli/internal/render"
	"github.com/mattboback/stageflow/clients/cli/internal/scanflow"
	"github.com/mattboback/stageflow/clients/cli/internal/staticsite"
	"github.com/mattboback/stageflow/clients/cli/internal/urlcheck"
	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
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
		Use:   "scan <url>... | scan <dir|zip>",
		Short: "Scan URLs, or upload a local build directory / ZIP archive and scan it",
		Long: "Scan one or more URLs and report the results.\n\n" +
			"When the argument is a local directory or .zip file, it is uploaded to the\n" +
			"API's ZIP intake and served from an isolated static server for scanning —\n" +
			"no dev server or public URL required (e.g. `stageflow scan ./dist`).",
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

	pathTargets := 0

	for _, arg := range args {
		if staticsite.IsPathTarget(arg) {
			pathTargets++
		}
	}

	if pathTargets > 0 {
		if len(args) > 1 {
			return exitcode.Error{Code: 2, Err: errors.New(
				"a directory/ZIP scan takes exactly one target and cannot be mixed with URLs",
			)}
		}

		return runStaticScan(cmd, root, opts, args[0])
	}

	req, err := buildScanRequest(cmd, root, opts, args)
	if err != nil {
		return err
	}

	client := apiclient.NewClient(root.apiURL, root.apiKey, nil)

	result, err := scanflow.SubmitURLsAndWait(
		cmd.Context(),
		client,
		req,
		opts.timeout,
		scanflow.WaitOptions{Progress: cmd.ErrOrStderr(), NoStream: opts.noStream},
	)
	if err != nil {
		return exitcode.Error{Code: 2, Err: err}
	}

	return renderScanReport(cmd, root, opts, result.Status, result.Report)
}

func renderScanReport(
	cmd *cobra.Command,
	root *rootOptions,
	opts scanCommandOptions,
	status apiclient.JobStatus,
	doc report.UnifiedReportV2,
) error {
	format, err := root.renderFormat()
	if err != nil {
		return exitcode.Error{Code: 2, Err: err}
	}

	return render.WrapError(
		render.UnifiedReport(cmd.OutOrStdout(), root.apiURL, status, doc, opts.reportOpts.RenderOptions(format)),
	)
}

// runStaticScan uploads a local directory or ZIP archive to the platform's
// ZIP intake and follows the job like a URL scan.
func runStaticScan(cmd *cobra.Command, root *rootOptions, opts scanCommandOptions, path string) error {
	if opts.authStatePath != "" || opts.authRecipePath != "" {
		return exitcode.Error{Code: 2, Err: errors.New(
			"--auth-state/--auth-recipe do not apply to directory/ZIP scans: the site is served statically",
		)}
	}

	modules, err := normalizeScannerList(opts.scanners)
	if err != nil {
		return exitcode.Error{Code: 2, Err: err}
	}

	target, err := staticsite.Package(path)
	if err != nil {
		return exitcode.Error{Code: 2, Err: err}
	}
	defer target.Cleanup()

	if !target.HasRootIndex {
		fmt.Fprintf(
			cmd.ErrOrStderr(),
			"Warning: %s has no top-level index.html; the scan may find nothing to serve.\n",
			path,
		)
	}

	client := apiclient.NewClient(root.apiURL, root.apiKey, nil)

	opCtx, cancel := context.WithTimeout(cmd.Context(), opts.timeout)
	defer cancel()

	resp, err := client.SubmitZipJob(opCtx, target.ZipPath, modules, opts.screenshot)
	if err != nil {
		return exitcode.Error{Code: 2, Err: fmt.Errorf("submit zip job: %w", err)}
	}

	jobID, err := scanflow.RequireJobID(resp)
	if err != nil {
		return exitcode.Error{Code: 2, Err: fmt.Errorf("submit zip job: %w", err)}
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "Uploaded %s\nJob submitted: %s\nWaiting for completion...\n", path, jobID)

	result, err := scanflow.WaitForReport(
		opCtx,
		client,
		jobID,
		scanflow.WaitOptions{Progress: cmd.ErrOrStderr(), NoStream: opts.noStream},
	)
	if err != nil {
		return exitcode.Error{Code: 2, Err: err}
	}

	return renderScanReport(cmd, root, opts, result.Status, result.Report)
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
