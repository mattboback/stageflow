package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
	"github.com/mattboback/stageflow/clients/cli/internal/projectmode"
	"github.com/mattboback/stageflow/clients/cli/internal/urlcheck"
	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
)

func resolveProjectStageflow(
	cmd *cobra.Command,
	root *rootOptions,
	cfg projectConfig,
	getenv getenvFunc,
) (string, string) {
	apiURL := root.apiURL
	if !cobraFlagChanged(cmd, "api") && cfg.Stageflow.APIURL != "" {
		apiURL = cfg.Stageflow.APIURL
	}

	apiKey := root.apiKey
	if !cobraFlagChanged(cmd, "api-key") && cfg.Stageflow.APIKeyEnv != "" {
		apiKey = getenv(cfg.Stageflow.APIKeyEnv)
	}

	return apiURL, apiKey
}

func resolveProjectTimeout(cmd *cobra.Command, defaultTimeout time.Duration, rawTimeout string) (time.Duration, error) {
	if cobraFlagChanged(cmd, "timeout") {
		return defaultTimeout, nil
	}

	d, ok, durationErr := configDuration(rawTimeout)
	if durationErr != nil {
		return 0, fmt.Errorf("invalid scan.timeout: %w", durationErr)
	}

	if ok {
		return d, nil
	}

	return defaultTimeout, nil
}

func runProjectDev(
	ctx context.Context,
	projectRoot string,
	cfg projectDevCfg,
	stderr io.Writer,
) (*runningProcess, func(), error) {
	setupErr := runCommandSteps(ctx, projectRoot, cfg.Up, stderr)
	if setupErr != nil {
		return nil, func() {}, fmt.Errorf("dev setup failed: %w", setupErr)
	}

	proc, err := startDevServer(ctx, projectRoot, cfg.Start, stderr)
	if err != nil {
		return nil, func() {}, fmt.Errorf("dev start failed: %w", err)
	}

	cleanup := func() {
		stopDevServer(proc, cfg.Stop, stderr)

		if len(cfg.Down) == 0 {
			return
		}

		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cleanupCancel()

		downErr := runCommandSteps(cleanupCtx, projectRoot, cfg.Down, stderr)
		if downErr != nil {
			fmt.Fprintf(stderr, "Dev teardown failed: %v\n", downErr)
		}
	}

	return proc, cleanup, nil
}

func runProjectScan(
	ctx context.Context,
	apiURL string,
	apiKey string,
	cfg projectScanCfg,
	timeout time.Duration,
	stderr io.Writer,
	noStream bool,
) (apiclient.JobStatus, report.UnifiedReportV2, error) {
	scanReq, urls, err := buildProjectSubmitJobRequest(cfg)
	if err != nil {
		return apiclient.JobStatus{}, report.UnifiedReportV2{}, err
	}

	if cfg.AllowPrivateTargets == nil && urlcheck.ContainsPrivateTargets(urls) {
		scanReq.AllowPrivateTargets = true

		fmt.Fprintln(stderr, "Detected private/loopback targets; setting allow_private_targets=true.")
	}

	validateErr := urlcheck.ValidateLocalTargets(apiURL, urls)
	if validateErr != nil {
		return apiclient.JobStatus{}, report.UnifiedReportV2{}, validateErr
	}

	client := apiclient.NewClient(apiURL, apiKey, nil)

	return runScanJob(ctx, client, scanReq, timeout, stderr, noStream)
}

func buildProjectSubmitJobRequest(cfg projectScanCfg) (apiclient.SubmitJobRequest, []string, error) {
	scanners := cfg.Scanners
	if scanners == "" {
		scanners = defaultScanScanners
	}

	modules, err := urlcheck.ParseModules(scanners)
	if err != nil {
		return apiclient.SubmitJobRequest{}, nil, fmt.Errorf("invalid scan.scanners: %w", err)
	}

	urls, err := urlcheck.NormalizeTargets(cfg.URLs)
	if err != nil {
		return apiclient.SubmitJobRequest{}, nil, err
	}

	screenshot := false
	if cfg.Screenshot != nil {
		screenshot = *cfg.Screenshot
	}

	allowPrivate := false
	if cfg.AllowPrivateTargets != nil {
		allowPrivate = *cfg.AllowPrivateTargets
	}

	return apiclient.SubmitJobRequest{
		URLs:                urls,
		Modules:             modules,
		Screenshot:          screenshot,
		AllowPrivateTargets: allowPrivate,
	}, urls, nil
}
func runProjectCommand(
	cmd *cobra.Command,
	args []string,
	root *rootOptions,
	getenv getenvFunc,
	opts *projectCmdOptions,
) error {
	if opts.MaxIssues < 0 {
		return exitCodeError{Code: 2, Err: errors.New("--max-issues must be >= 0")}
	}

	projectArg := ""
	if len(args) == 1 {
		projectArg = args[0]
	}

	projectRoot, err := projectmode.ResolveProjectRoot(projectArg)
	if err != nil {
		return exitCodeError{Code: 2, Err: err}
	}

	cfg, cfgPath, bootstrapped, err := loadOrBootstrapProjectConfig(projectRoot, root.apiURL, cmd.OutOrStdout())
	if err != nil {
		return exitCodeError{Code: 2, Err: err}
	}

	if bootstrapped {
		return nil
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "Using project config: %s\n", cfgPath)

	configReadyErr := ensureProjectConfigReady(projectRoot, cfgPath, cfg)
	if configReadyErr != nil {
		return exitCodeError{Code: 2, Err: configReadyErr}
	}

	apiURL, apiKey := resolveProjectStageflow(cmd, root, cfg, getenv)

	totalTimeout, err := resolveProjectTimeout(cmd, opts.Timeout, cfg.Scan.Timeout)
	if err != nil {
		return exitCodeError{Code: 2, Err: err}
	}

	totalCtx, cancel := context.WithTimeout(cmd.Context(), totalTimeout)
	defer cancel()

	proc, cleanup, err := runProjectDev(totalCtx, projectRoot, cfg.Dev, cmd.ErrOrStderr())
	if err != nil {
		return exitCodeError{Code: 2, Err: err}
	}
	defer cleanup()

	readyErr := waitForReady(totalCtx, proc, cfg.Dev.Ready, cmd.ErrOrStderr())
	if readyErr != nil {
		return exitCodeError{Code: 2, Err: fmt.Errorf("dev readiness failed: %w", readyErr)}
	}

	remaining := remainingDuration(totalCtx)
	if remaining <= 0 {
		return exitCodeError{Code: 2, Err: errors.New("timed out before scan could start")}
	}

	status, doc, err := runProjectScan(
		totalCtx,
		apiURL,
		apiKey,
		cfg.Scan,
		remaining,
		cmd.ErrOrStderr(),
		opts.NoStream,
	)
	if err != nil {
		return exitCodeError{Code: 2, Err: err}
	}

	format, err := root.outputFormat()
	if err != nil {
		return exitCodeError{Code: 2, Err: err}
	}

	renderErr := renderUnifiedReport(cmd.OutOrStdout(), apiURL, status, doc, reportRenderOptions{
		Format:    format,
		MaxIssues: opts.MaxIssues,
	})
	if renderErr != nil {
		return exitCodeError{Code: 2, Err: renderErr}
	}

	return nil
}
