package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
)

type projectInitEnvelope struct {
	Schema      string   `json:"schema"`
	ProjectRoot string   `json:"projectRoot"`
	ConfigPath  string   `json:"configPath"`
	GuidePath   string   `json:"guidePath"`
	Created     bool     `json:"created"`
	NextSteps   []string `json:"nextSteps"`
}

type projectDoctorEnvelope struct {
	Schema                  string               `json:"schema"`
	ProjectRoot             string               `json:"projectRoot"`
	ConfigPath              string               `json:"configPath"`
	Passed                  bool                 `json:"passed"`
	APIURL                  string               `json:"apiUrl"`
	URLs                    []string             `json:"urls"`
	AutoAllowPrivateTargets bool                 `json:"autoAllowPrivateTargets"`
	HostedMemory            projectHostedMemory  `json:"hostedMemory"`
	Checks                  []projectDoctorCheck `json:"checks"`
}

type projectHostedMemory struct {
	Configured             bool   `json:"configured"`
	ProjectSlug            string `json:"projectSlug,omitempty"`
	APIURL                 string `json:"apiUrl,omitempty"`
	RecommendedScanCommand string `json:"recommendedScanCommand,omitempty"`
	PromoteCommandTemplate string `json:"promoteCommandTemplate,omitempty"`
}

type projectDoctorCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

func newProjectCmd(root *rootOptions, getenv getenvFunc) *cobra.Command {
	runOpts := &projectCmdOptions{
		Timeout:   10 * time.Minute,
		MaxIssues: defaultMaxIssues,
	}
	doctorOpts := &projectDoctorCmdOptions{
		Timeout: 2 * time.Minute,
	}

	cmd := &cobra.Command{
		Use:                   "project [path]",
		Short:                 "Run Project Mode scan using .stageflow/config.yaml",
		DisableFlagsInUseLine: true,
		Args:                  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProjectCommand(cmd, args, root, getenv, runOpts)
		},
	}

	cmd.Flags().DurationVar(&runOpts.Timeout, "timeout", runOpts.Timeout, "Max total time (dev + scan)")
	cmd.Flags().IntVar(
		&runOpts.MaxIssues,
		"max-issues",
		runOpts.MaxIssues,
		"Max issues to include in output (0 = unlimited)",
	)
	cmd.Flags().BoolVar(&runOpts.NoStream, "no-stream", false, "Poll instead of SSE")
	cobra.CheckErr(cmd.Flags().MarkHidden("no-stream"))

	initCmd := &cobra.Command{
		Use:                   "init [path]",
		Short:                 "Create .stageflow config and setup guide",
		DisableFlagsInUseLine: true,
		Args:                  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProjectInitCommand(cmd, args, root)
		},
	}

	doctorCmd := &cobra.Command{
		Use:                   "doctor [path]",
		Short:                 "Validate project config and dev readiness without scanning",
		DisableFlagsInUseLine: true,
		Args:                  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProjectDoctorCommand(cmd, args, root, getenv, doctorOpts)
		},
	}
	doctorCmd.Flags().DurationVar(
		&doctorOpts.Timeout,
		"timeout",
		doctorOpts.Timeout,
		"Max total time for doctor checks",
	)
	doctorCmd.Flags().BoolVar(&doctorOpts.SkipDev, "skip-dev", false, "Skip starting dev server and readiness checks")

	cmd.AddCommand(initCmd, doctorCmd)
	cmd.AddCommand(newProjectRemoteCmd(root)...)

	return cmd
}

type projectCmdOptions struct {
	Timeout   time.Duration
	MaxIssues int
	NoStream  bool
}

type projectDoctorCmdOptions struct {
	Timeout time.Duration
	SkipDev bool
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

	projectRoot, err := resolveProjectRoot(projectArg)
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

func runProjectInitCommand(
	cmd *cobra.Command,
	args []string,
	root *rootOptions,
) error {
	projectArg := ""
	if len(args) == 1 {
		projectArg = args[0]
	}

	projectRoot, err := resolveProjectRoot(projectArg)
	if err != nil {
		return exitCodeError{Code: 2, Err: err}
	}

	configPath, guidePath, created, err := scaffoldProjectBootstrap(projectRoot, root.apiURL)
	if err != nil {
		return exitCodeError{Code: 2, Err: err}
	}

	if !created {
		format, formatErr := root.outputFormat()
		if formatErr != nil {
			return exitCodeError{Code: 2, Err: formatErr}
		}

		if format == outputFormatJSON {
			return writeProjectInitJSON(cmd.OutOrStdout(), projectRoot, configPath, guidePath, false)
		}

		fmt.Fprintln(cmd.OutOrStdout(), "StageFlow project bootstrap already exists:")
		fmt.Fprintf(cmd.OutOrStdout(), "- %s\n", configPath)
		fmt.Fprintf(cmd.OutOrStdout(), "- %s\n", guidePath)

		return nil
	}

	format, formatErr := root.outputFormat()
	if formatErr != nil {
		return exitCodeError{Code: 2, Err: formatErr}
	}

	if format == outputFormatJSON {
		return writeProjectInitJSON(cmd.OutOrStdout(), projectRoot, configPath, guidePath, true)
	}

	printProjectBootstrapHelp(cmd.OutOrStdout(), projectRoot, configPath, guidePath)

	return nil
}

//nolint:gocyclo
func runProjectDoctorCommand(
	cmd *cobra.Command,
	args []string,
	root *rootOptions,
	getenv getenvFunc,
	opts *projectDoctorCmdOptions,
) error {
	if opts.Timeout <= 0 {
		return exitCodeError{Code: 2, Err: errors.New("--timeout must be > 0")}
	}

	projectArg := ""
	if len(args) == 1 {
		projectArg = args[0]
	}

	projectRoot, err := resolveProjectRoot(projectArg)
	if err != nil {
		return exitCodeError{Code: 2, Err: err}
	}

	cfg, cfgPath, err := loadProjectConfig(projectRoot)
	if err != nil {
		var missingErr missingProjectConfigError
		if errors.As(err, &missingErr) {
			hint := "stageflow project init"
			if projectArg != "" {
				hint = fmt.Sprintf("stageflow project init %s", projectArg)
			}

			return exitCodeError{
				Code: 2,
				Err:  fmt.Errorf("project config not found under %s; run `%s`", projectRoot, hint),
			}
		}

		return exitCodeError{Code: 2, Err: err}
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "Using project config: %s\n", cfgPath)

	if !opts.SkipDev {
		configReadyErr := ensureProjectConfigReady(projectRoot, cfgPath, cfg)
		if configReadyErr != nil {
			return exitCodeError{Code: 2, Err: configReadyErr}
		}
	}

	apiURL, _ := resolveProjectStageflow(cmd, root, cfg, getenv)

	_, urls, err := buildProjectSubmitJobRequest(cfg.Scan)
	if err != nil {
		return exitCodeError{Code: 2, Err: err}
	}

	validateErr := validateLocalTargets(apiURL, urls)
	if validateErr != nil {
		return exitCodeError{Code: 2, Err: validateErr}
	}

	autoAllowPrivateTargets := cfg.Scan.AllowPrivateTargets == nil && containsPrivateTargets(urls)
	if autoAllowPrivateTargets {
		fmt.Fprintln(
			cmd.ErrOrStderr(),
			"Detected private/loopback targets; project scans will auto-enable allow_private_targets=true.",
		)
	}

	checks := []projectDoctorCheck{
		{Name: "config", Status: "passed", Message: "Loaded and validated project config."},
		{Name: "scan-preflight", Status: "passed", Message: "Validated scan targets and scanner configuration."},
	}

	if opts.SkipDev {
		checks = append(checks, projectDoctorCheck{
			Name:    "dev-readiness",
			Status:  "skipped",
			Message: "Skipped by --skip-dev.",
		})

		format, formatErr := root.outputFormat()
		if formatErr != nil {
			return exitCodeError{Code: 2, Err: formatErr}
		}

		if format == outputFormatJSON {
			return writeProjectDoctorJSON(cmd.OutOrStdout(), projectRoot, cfgPath, apiURL, urls, cfg.Stageflow, autoAllowPrivateTargets, checks)
		}

		fmt.Fprintln(cmd.OutOrStdout(), "Doctor checks passed (config + scan preflight).")

		return nil
	}

	doctorCtx, cancel := context.WithTimeout(cmd.Context(), opts.Timeout)
	defer cancel()

	proc, cleanup, err := runProjectDev(doctorCtx, projectRoot, cfg.Dev, cmd.ErrOrStderr())
	if err != nil {
		return exitCodeError{Code: 2, Err: err}
	}
	defer cleanup()

	readyErr := waitForReady(doctorCtx, proc, cfg.Dev.Ready, cmd.ErrOrStderr())
	if readyErr != nil {
		return exitCodeError{Code: 2, Err: fmt.Errorf("dev readiness failed: %w", readyErr)}
	}

	checks = append(checks, projectDoctorCheck{
		Name:    "dev-readiness",
		Status:  "passed",
		Message: "Started the dev command and observed a ready HTTP response.",
	})

	format, formatErr := root.outputFormat()
	if formatErr != nil {
		return exitCodeError{Code: 2, Err: formatErr}
	}

	if format == outputFormatJSON {
		return writeProjectDoctorJSON(cmd.OutOrStdout(), projectRoot, cfgPath, apiURL, urls, cfg.Stageflow, autoAllowPrivateTargets, checks)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Doctor checks passed (config + scan preflight + dev readiness).")

	return nil
}

func loadOrBootstrapProjectConfig(
	projectRoot string,
	apiURL string,
	out io.Writer,
) (projectConfig, string, bool, error) {
	cfg, cfgPath, err := loadProjectConfig(projectRoot)
	if err == nil {
		return cfg, cfgPath, false, nil
	}

	var missingErr missingProjectConfigError
	if !errors.As(err, &missingErr) {
		return projectConfig{}, "", false, err
	}

	configPath, guidePath, _, scaffoldErr := scaffoldProjectBootstrap(projectRoot, apiURL)
	if scaffoldErr != nil {
		return projectConfig{}, "", false, scaffoldErr
	}

	printProjectBootstrapHelp(out, projectRoot, configPath, guidePath)

	return projectConfig{}, "", true, nil
}

func scaffoldProjectBootstrap(projectRoot string, apiURL string) (string, string, bool, error) {
	configPath := filepath.Join(projectRoot, ".stageflow", "config.yaml")
	guidePath := filepath.Join(projectRoot, ".stageflow", "README.md")

	configExists, err := pathExists(configPath)
	if err != nil {
		return "", "", false, fmt.Errorf("stat %s: %w", configPath, err)
	}

	guideExists, err := pathExists(guidePath)
	if err != nil {
		return "", "", false, fmt.Errorf("stat %s: %w", guidePath, err)
	}

	configPath, err = scaffoldProjectConfig(projectRoot, apiURL)
	if err != nil {
		return "", "", false, err
	}

	guidePath, err = scaffoldProjectGuide(projectRoot)
	if err != nil {
		return "", "", false, err
	}

	return configPath, guidePath, !configExists || !guideExists, nil
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}

	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	return false, err
}

func printProjectBootstrapHelp(
	out io.Writer,
	projectRoot string,
	configPath string,
	guidePath string,
) {
	relPath, err := filepath.Rel(projectRoot, configPath)
	if err != nil {
		relPath = configPath
	}

	relGuidePath, relGuideErr := filepath.Rel(projectRoot, guidePath)
	if relGuideErr != nil {
		relGuidePath = guidePath
	}

	fmt.Fprintln(out, "Created StageFlow project bootstrap:")
	fmt.Fprintf(out, "- %s\n", configPath)
	fmt.Fprintf(out, "- %s\n\n", guidePath)
	fmt.Fprintln(out, "Next steps:")
	fmt.Fprintf(out, "1. Read %s\n", relGuidePath)
	fmt.Fprintf(out, "2. Edit %s and update:\n", relPath)
	fmt.Fprintln(out, "   - dev.start.cmd (your dev start command)")
	fmt.Fprintln(out, "   - dev.ready.url (the URL your dev server serves)")
	fmt.Fprintln(out, "   - scan.urls (the pages you want scanned)")
	fmt.Fprintln(out, "3. For localhost/private target scans, run your local StageFlow stack:")
	fmt.Fprintln(out, "   - just dev up local")
	fmt.Fprintln(out, "   - just dev init local")
	fmt.Fprintln(out, "4. Re-run: stageflow project")
}

func hasScaffoldPlaceholderDevCommand(cfg projectConfig) bool {
	for _, item := range cfg.Dev.Start.Cmd {
		if strings.TrimSpace(item) == scaffoldDevStartCommandPlaceholder {
			return true
		}
	}

	return false
}

func ensureProjectConfigReady(projectRoot string, cfgPath string, cfg projectConfig) error {
	if !hasScaffoldPlaceholderDevCommand(cfg) {
		return nil
	}

	guidePath := filepath.Join(projectRoot, ".stageflow", "README.md")

	return fmt.Errorf(
		"project config is not set up yet: update dev.start.cmd in %s, then follow %s",
		cfgPath,
		guidePath,
	)
}

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
) (JobStatus, report.UnifiedReportV2, error) {
	scanReq, urls, err := buildProjectSubmitJobRequest(cfg)
	if err != nil {
		return JobStatus{}, report.UnifiedReportV2{}, err
	}

	if cfg.AllowPrivateTargets == nil && containsPrivateTargets(urls) {
		scanReq.AllowPrivateTargets = true

		fmt.Fprintln(stderr, "Detected private/loopback targets; setting allow_private_targets=true.")
	}

	validateErr := validateLocalTargets(apiURL, urls)
	if validateErr != nil {
		return JobStatus{}, report.UnifiedReportV2{}, validateErr
	}

	client := NewClient(apiURL, apiKey, nil)

	return runScanJob(ctx, client, scanReq, timeout, stderr, noStream)
}

func buildProjectSubmitJobRequest(cfg projectScanCfg) (SubmitJobRequest, []string, error) {
	scanners := cfg.Scanners
	if scanners == "" {
		scanners = defaultScanScanners
	}

	modules, err := parseModules(scanners)
	if err != nil {
		return SubmitJobRequest{}, nil, fmt.Errorf("invalid scan.scanners: %w", err)
	}

	urls, err := normalizeTargetURLs(cfg.URLs)
	if err != nil {
		return SubmitJobRequest{}, nil, err
	}

	screenshot := false
	if cfg.Screenshot != nil {
		screenshot = *cfg.Screenshot
	}

	allowPrivate := false
	if cfg.AllowPrivateTargets != nil {
		allowPrivate = *cfg.AllowPrivateTargets
	}

	return SubmitJobRequest{
		URLs:                urls,
		Modules:             modules,
		Screenshot:          screenshot,
		AllowPrivateTargets: allowPrivate,
	}, urls, nil
}

func writeProjectInitJSON(
	out io.Writer,
	projectRoot, configPath, guidePath string,
	created bool,
) error {
	payload := projectInitEnvelope{
		Schema:      "stageflow-cli/project-init@v1",
		ProjectRoot: projectRoot,
		ConfigPath:  configPath,
		GuidePath:   guidePath,
		Created:     created,
		NextSteps: []string{
			"Read .stageflow/README.md.",
			"Update dev.start.cmd, dev.ready.url, and scan.urls in .stageflow/config.yaml.",
			"Optional: set stageflow.remote_project and stageflow.remote_api_url for the hosted regression-memory step.",
			"For localhost/private scans, start the local StageFlow stack with just dev up local and just dev init local.",
			"Run stageflow project doctor, then stageflow project.",
		},
	}

	return writeJSONEnvelope(out, payload)
}

func writeProjectDoctorJSON(
	out io.Writer,
	projectRoot, configPath, apiURL string,
	urls []string,
	stageflowCfg projectStageflowCfg,
	autoAllowPrivateTargets bool,
	checks []projectDoctorCheck,
) error {
	payload := projectDoctorEnvelope{
		Schema:                  "stageflow-cli/project-doctor@v1",
		ProjectRoot:             projectRoot,
		ConfigPath:              configPath,
		Passed:                  true,
		APIURL:                  apiURL,
		URLs:                    urls,
		AutoAllowPrivateTargets: autoAllowPrivateTargets,
		HostedMemory:            buildProjectHostedMemory(stageflowCfg),
		Checks:                  checks,
	}

	return writeJSONEnvelope(out, payload)
}

func buildProjectHostedMemory(cfg projectStageflowCfg) projectHostedMemory {
	projectSlug := strings.TrimSpace(cfg.RemoteProject)
	apiURL := strings.TrimSpace(cfg.RemoteAPIURL)
	configured := projectSlug != ""

	hosted := projectHostedMemory{
		Configured:  configured,
		ProjectSlug: projectSlug,
		APIURL:      apiURL,
	}
	if !configured {
		return hosted
	}

	hosted.RecommendedScanCommand = buildHostedProjectScanCommand(projectSlug, apiURL)
	hosted.PromoteCommandTemplate = buildHostedProjectPromoteCommand(projectSlug, apiURL)

	return hosted
}

func buildHostedProjectScanCommand(projectSlug string, apiURL string) string {
	command := fmt.Sprintf("stageflow scan --project %s --format json", projectSlug)
	if trimmedAPIURL := strings.TrimSpace(apiURL); trimmedAPIURL != "" {
		command += fmt.Sprintf(" --api %s", trimmedAPIURL)
	}

	return command
}

func buildHostedProjectPromoteCommand(projectSlug string, apiURL string) string {
	command := fmt.Sprintf("stageflow project promote %s --job-id <job-id>", projectSlug)
	if trimmedAPIURL := strings.TrimSpace(apiURL); trimmedAPIURL != "" {
		command += fmt.Sprintf(" --api %s", trimmedAPIURL)
	}

	return command
}

func writeJSONEnvelope(out io.Writer, payload any) error {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)

	return encoder.Encode(payload)
}

func cobraFlagChanged(cmd *cobra.Command, name string) bool {
	if flag := cmd.Flags().Lookup(name); flag != nil && flag.Changed {
		return true
	}

	if flag := cmd.InheritedFlags().Lookup(name); flag != nil && flag.Changed {
		return true
	}

	return false
}
