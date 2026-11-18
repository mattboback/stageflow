package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
	"github.com/mattboback/stageflow/clients/cli/internal/authintake"
	"github.com/mattboback/stageflow/clients/cli/internal/exitcode"
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
		authStatePath  string
		authRecipePath string
		allowPartial   bool
		discover       bool
		profile        string
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
				authStatePath:  authStatePath,
				authRecipePath: authRecipePath,
				allowPartial:   allowPartial,
				discover:       discover,
				profile:        profile,
				reportOpts:     reportOpts,
			}

			return runScanCmd(cmd, root, opts, args)
		},
	}

	cmd.Flags().StringSliceVar(
		&scanners,
		"scanner",
		nil,
		"Scanner module (repeatable or comma-separated; default: all scanners)",
	)
	cmd.Flags().StringVar(&profile, "profile", "production", "Scan assessment profile: production or staging")
	cmd.Flags().
		BoolVar(&discover, "discover", false, "Discover public website pages before scanning (one seed; no authentication)")
	cmd.Flags().BoolVar(&screenshot, "screenshot", true, "Capture screenshots")
	cmd.Flags().BoolVar(
		&allowPrivate,
		"allow-private-targets",
		false,
		"Allow private/loopback targets (requires API instance to permit it)",
	)
	cmd.Flags().DurationVar(&timeout, "timeout", defaultScanWaitTimeout, "Max wait time including queue and scan")
	cmd.Flags().StringVar(
		&authStatePath,
		"auth-state",
		"",
		"Path to a Playwright storage-state JSON captured via stageflow auth capture. "+
			"Stored in a private job file and mounted only into authenticated scanners.",
	)
	cmd.Flags().StringVar(
		&authRecipePath,
		"auth-recipe",
		"",
		"Path to a YAML/JSON form-auth recipe (Provenance.auth.form shape). "+
			"Step values may be literals or {from_env: NAME} resolved from the CLI environment; unset variables fail.",
	)
	cmd.MarkFlagsMutuallyExclusive("auth-state", "auth-recipe")
	cmd.Flags().BoolVar(
		&allowPartial,
		"allow-partial",
		false,
		"Exit 0 even when a scanner did not finish (by default a partial report exits 2)",
	)
	render.BindReportFlags(cmd, &reportOpts, true)

	return cmd
}

type scanCommandOptions struct {
	scanners       []string
	screenshot     bool
	allowPrivate   bool
	timeout        time.Duration
	authStatePath  string
	authRecipePath string
	allowPartial   bool
	discover       bool
	profile        string
	reportOpts     render.ReportFlags
}

func runScanCmd(cmd *cobra.Command, root *rootOptions, opts scanCommandOptions, args []string) error {
	if opts.profile != "" && opts.profile != "production" && opts.profile != "staging" {
		return exitcode.Error{Code: 2, Err: errors.New("--profile must be production or staging")}
	}

	if opts.reportOpts.MaxIssues < 0 {
		return exitcode.Error{Code: 2, Err: errors.New("--max-issues must be >= 0")}
	}

	if opts.discover {
		discovered, err := discoverScanTargets(cmd, root, opts, args)
		if err != nil {
			return err
		}

		args = discovered
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

	req, err := buildScanRequest(
		root.apiURL,
		opts,
		args,
		cmd.Flags().Changed("allow-private-targets"),
		cmd.ErrOrStderr(),
	)
	if err != nil {
		return err
	}

	if opts.profile == "staging" {
		req.Profile = opts.profile
	}

	client := apiclient.NewClient(root.apiURL, root.apiKey, nil)

	result, err := scanflow.SubmitURLsAndWait(
		cmd.Context(),
		client,
		req,
		opts.timeout,
		scanflow.WaitOptions{Progress: cmd.ErrOrStderr()},
	)
	if err != nil {
		return exitcode.Error{Code: 2, Err: err}
	}

	return renderScanReport(cmd, root, opts, result.Status, result.Report)
}

func discoverScanTargets(
	cmd *cobra.Command,
	root *rootOptions,
	opts scanCommandOptions,
	args []string,
) ([]string, error) {
	if opts.authStatePath != "" || opts.authRecipePath != "" || opts.allowPrivate {
		return nil, exitcode.Error{
			Code: 2,
			Err: errors.New(
				"--discover supports public pages only; authentication and private-target flags are not supported",
			),
		}
	}

	seed, err := publicDiscoverySeed(args)
	if err != nil {
		return nil, exitcode.Error{Code: 2, Err: err}
	}

	found, err := apiclient.NewClient(root.apiURL, root.apiKey, nil).DiscoverPages(cmd.Context(), seed)
	if err != nil {
		return nil, exitcode.Error{Code: 2, Err: err}
	}

	writeDiscoveryNotice(cmd.ErrOrStderr(), found)

	if len(found.URLs) == 0 {
		return nil, exitcode.Error{Code: 2, Err: errors.New("discovery found no scannable HTML pages")}
	}

	return found.URLs, nil
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

	renderErr := render.WrapError(
		render.UnifiedReport(cmd.OutOrStdout(), root.apiURL, status, doc, opts.reportOpts.RenderOptions(format)),
	)

	incomplete, warnErr := render.WarnIfPartial(cmd.ErrOrStderr(), doc)
	if warnErr != nil {
		return exitcode.Error{Code: 2, Err: warnErr}
	}

	// A partial report cannot pass a --fail-on gate: the missing scanner's
	// findings were never checked. That outranks the gate's own result.
	if len(incomplete) > 0 && !opts.allowPartial {
		return exitcode.Error{Code: 2, Err: fmt.Errorf(
			"partial report: %s did not finish; pass --allow-partial to accept it",
			strings.Join(incomplete, ", "),
		)}
	}

	return renderErr
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

	resp, err := client.SubmitZipJob(
		opCtx,
		target.ZipPath,
		modules,
		opts.screenshot,
		map[string]string{"profile": defaultProfile(opts.profile)},
	)
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
		scanflow.WaitOptions{Progress: cmd.ErrOrStderr()},
	)
	if err != nil {
		return exitcode.Error{Code: 2, Err: err}
	}

	return renderScanReport(cmd, root, opts, result.Status, result.Report)
}

func buildScanRequest(
	apiURL string,
	opts scanCommandOptions,
	args []string,
	allowPrivateExplicit bool,
	stderr io.Writer,
) (apiclient.SubmitJobRequest, error) {
	urls, err := urlcheck.NormalizeTargets(args)
	if err != nil {
		return apiclient.SubmitJobRequest{}, exitcode.Error{Code: 2, Err: err}
	}

	modules, err := normalizeScannerList(opts.scanners)
	if err != nil {
		return apiclient.SubmitJobRequest{}, exitcode.Error{Code: 2, Err: err}
	}

	validateErr := urlcheck.ValidateLocalTargets(apiURL, urls)
	if validateErr != nil {
		return apiclient.SubmitJobRequest{}, exitcode.Error{Code: 2, Err: validateErr}
	}

	authInput, hasAuth, authErr := authintake.LoadFromFlags(opts.authStatePath, opts.authRecipePath)
	if authErr != nil {
		return apiclient.SubmitJobRequest{}, exitcode.Error{Code: 2, Err: authErr}
	}

	req := apiclient.SubmitJobRequest{
		URLs:       urls,
		Modules:    modules,
		Screenshot: opts.screenshot,
		AllowPrivateTargets: effectiveAllowPrivateTargets(
			urls,
			opts.allowPrivate,
			allowPrivateExplicit,
			stderr,
		),
	}
	if hasAuth {
		req.Auth = authInput

		fmt.Fprintf(
			stderr,
			"Attaching auth block (mode=%s) to scan submission.\n",
			authNoticeMode(opts),
		)
	}

	return req, nil
}

func effectiveAllowPrivateTargets(
	urls []string,
	allowPrivate bool,
	allowPrivateExplicit bool,
	stderr io.Writer,
) bool {
	if allowPrivate || allowPrivateExplicit || !urlcheck.ContainsPrivateTargets(urls) {
		return allowPrivate
	}

	fmt.Fprintln(
		stderr,
		"Detected private/loopback targets; setting allow_private_targets=true.",
	)

	return true
}

// Use fixed labels: a malformed recipe mode is untrusted input, validated by the API.
func authNoticeMode(opts scanCommandOptions) string {
	if opts.authStatePath != "" {
		return "storage_state"
	}

	return "form"
}

func defaultProfile(value string) string {
	if value == "staging" {
		return value
	}

	return "production"
}
