package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
	"github.com/mattboback/stageflow/clients/cli/internal/diffrender"
	"github.com/mattboback/stageflow/clients/cli/internal/exitcode"
	"github.com/mattboback/stageflow/clients/cli/internal/render"
	"github.com/mattboback/stageflow/clients/cli/internal/scanflow"
	"github.com/mattboback/stageflow/clients/cli/internal/urlcheck"
	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
	"github.com/mattboback/stageflow/libs/go/diff"
)

func newDiffCmd(root *rootOptions) *cobra.Command {
	var (
		failOnNew        string
		gatePolicy       string
		failOnRegression bool
		allowPartial     bool
		timeout          time.Duration
	)

	cmd := &cobra.Command{
		Use:                   "diff <baseline.json> <current.json | url>",
		Short:                 "Compare a current scan against a saved baseline",
		DisableFlagsInUseLine: true,
		Args:                  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDiffCommand(
				cmd,
				root,
				args[0],
				args[1],
				timeout,
				failOnNew,
				failOnRegression,
				allowPartial,
			)
		},
	}

	cmd.Flags().StringVar(&gatePolicy, "gate-policy", "raw", "Finding policy for regression gates: raw or actionable")
	cmd.Flags().StringVar(
		&failOnNew,
		"fail-on-new",
		"",
		"Exit 1 if any NEW issue meets threshold "+
			"(critical, serious, moderate, minor, info) or 'any'",
	)
	cmd.Flags().Lookup("fail-on-new").NoOptDefVal = "any"
	cmd.Flags().BoolVar(
		&failOnRegression,
		"fail-on-regression",
		false,
		"Exit 1 if score dropped or new issues appeared",
	)
	cmd.Flags().
		BoolVar(&allowPartial, "allow-partial", false, "Compare completed scanners even when either report has incomplete coverage")
	cmd.Flags().DurationVar(&timeout, "timeout", defaultScanWaitTimeout, "Max wait time including queue and live scan")

	return cmd
}

func runDiffCommand(
	cmd *cobra.Command,
	root *rootOptions,
	baselinePath, currentTarget string,
	timeout time.Duration,
	failOnNew string,
	failOnRegression bool,
	allowPartial bool,
) error {
	baselineEnv, err := loadReportFile(baselinePath)
	if err != nil {
		return exitcode.Error{Code: 2, Err: fmt.Errorf("baseline: %w", err)}
	}

	currentEnv, currentJobID, currentFile, err := loadCurrentDiffTarget(
		cmd,
		root,
		baselineEnv,
		currentTarget,
		timeout,
	)
	if err != nil {
		return exitcode.Error{Code: 2, Err: err}
	}

	result := diff.ComputeDiff("", baselineEnv.Report, currentJobID, currentEnv.Report)
	d := diffrender.FromResult(result, baselinePath, currentFile)

	gatePolicy, policyErr := diffGatePolicy(cmd)
	if policyErr != nil {
		return exitcode.Error{Code: 2, Err: policyErr}
	}

	gateResult, policyErr := actionableDiffGate(gatePolicy, baselineEnv.Report, currentEnv.Report, d)
	if policyErr != nil {
		return exitcode.Error{Code: 2, Err: policyErr}
	}

	shouldFail, err := diffrender.EvaluateRegression(
		gateResult,
		failOnRegression,
		failOnNew,
		render.HasIssuesAtOrAbove,
	)
	if err != nil {
		return exitcode.Error{Code: 2, Err: err}
	}

	d.Regressed = diffrender.IsRegressed(d)

	format, err := root.renderFormat()
	if err != nil {
		return exitcode.Error{Code: 2, Err: err}
	}

	diffFormat, err := diffRenderFormat(format)
	if err != nil {
		return exitcode.Error{Code: 2, Err: err}
	}

	err = diffrender.Render(cmd.OutOrStdout(), d, diffFormat)
	if err != nil {
		return exitcode.Error{Code: 2, Err: err}
	}

	if result.IncompatibleIdentity {
		return exitcode.Error{
			Code: 2,
			Err:  errors.New("baseline identity format is incompatible; scan again and save a fresh baseline"),
		}
	}

	if (len(result.SkippedScanners) > 0 || len(result.NotComparedPages) > 0) && !allowPartial {
		return exitcode.Error{
			Code: 2,
			Err: fmt.Errorf(
				"incomplete comparison: %s; use --allow-partial to accept available coverage",
				strings.Join(result.SkippedScanners, ", "),
			),
		}
	}

	if shouldFail {
		return exitcode.Error{Code: 1}
	}

	return nil
}

func loadCurrentDiffTarget(
	cmd *cobra.Command,
	root *rootOptions,
	baselineEnv render.ReportEnvelope,
	currentTarget string,
	timeout time.Duration,
) (render.ReportEnvelope, string, string, error) {
	if !diffrender.IsRemoteTarget(currentTarget) {
		currentEnv, err := loadReportFile(currentTarget)
		if err != nil {
			return render.ReportEnvelope{}, "", "", fmt.Errorf("current: %w", err)
		}

		return currentEnv, "", currentTarget, nil
	}

	if !isExplicitHTTPURL(currentTarget) {
		info, statErr := os.Stat(currentTarget)
		if statErr == nil && !info.IsDir() {
			currentEnv, err := loadReportFile(currentTarget)
			if err != nil {
				return render.ReportEnvelope{}, "", "", fmt.Errorf("current: %w", err)
			}

			return currentEnv, "", currentTarget, nil
		}
	}

	currentEnv, jobID, err := runLiveDiffScan(cmd, root, baselineEnv, currentTarget, timeout)
	if err != nil {
		return render.ReportEnvelope{}, "", "", err
	}

	return currentEnv, jobID, "", nil
}

func isExplicitHTTPURL(target string) bool {
	lower := strings.ToLower(strings.TrimSpace(target))

	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}

func runLiveDiffScan(
	cmd *cobra.Command,
	root *rootOptions,
	baselineEnv render.ReportEnvelope,
	currentTarget string,
	timeout time.Duration,
) (render.ReportEnvelope, string, error) {
	urls, err := urlcheck.NormalizeTargets([]string{currentTarget})
	if err != nil {
		return render.ReportEnvelope{}, "", err
	}

	validateErr := urlcheck.ValidateLocalTargets(root.apiURL, urls)
	if validateErr != nil {
		return render.ReportEnvelope{}, "", validateErr
	}

	allowPrivateTargets := urlcheck.ContainsPrivateTargets(urls)
	if allowPrivateTargets {
		fmt.Fprintln(
			cmd.ErrOrStderr(),
			"Detected private/loopback targets; setting allow_private_targets=true.",
		)
	}

	client := apiclient.NewClient(root.apiURL, root.apiKey, nil)
	req := apiclient.SubmitJobRequest{
		URLs:                urls,
		Modules:             diffScanModules(baselineEnv),
		AllowPrivateTargets: allowPrivateTargets,
	}

	if baselineEnv.Report.Meta.Profile != nil {
		req.Profile = string(*baselineEnv.Report.Meta.Profile)
	}

	if context := baselineEnv.Report.Meta.ScanContext; context != nil {
		req.ScannerConfigs = context.ScannerConfigurations
	}

	result, err := scanflow.SubmitURLsAndWait(
		cmd.Context(),
		client,
		req,
		timeout,
		scanflow.WaitOptions{Progress: cmd.ErrOrStderr()},
	)
	if err != nil {
		return render.ReportEnvelope{}, "", err
	}

	return render.ReportEnvelope{
		Job:    render.JobMeta{ID: result.Status.ID},
		Report: result.Report,
	}, result.Status.ID, nil
}

func diffScanModules(env render.ReportEnvelope) []string {
	modules := make([]string, 0, len(env.Report.Scanners))
	for _, scanner := range env.Report.Scanners {
		modules = append(modules, scanner.Id)
	}

	return modules
}

func diffRenderFormat(format render.Format) (diffrender.Format, error) {
	switch format {
	case render.FormatJSON:
		return diffrender.FormatJSON, nil
	case render.FormatText:
		return diffrender.FormatText, nil
	case render.FormatMarkdown:
		return diffrender.FormatMarkdown, nil
	default:
		return 0, fmt.Errorf("unsupported output format %q", format)
	}
}

func loadReportFile(path string) (render.ReportEnvelope, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return render.ReportEnvelope{}, fmt.Errorf("read %s: %w", path, err)
	}

	var env render.ReportEnvelope

	err = json.Unmarshal(data, &env)
	if err != nil {
		return render.ReportEnvelope{}, fmt.Errorf("parse %s: %w", path, err)
	}

	if env.Schema != "stageflow-cli/report@v1" || env.Report.Version == "" {
		return render.ReportEnvelope{}, fmt.Errorf(
			"parse %s: expected a scan --format json report envelope (stageflow-cli/report@v1)",
			path,
		)
	}

	if env.Filters.Truncated || env.Report.Summary.TotalIssues != len(env.Report.Issues) {
		return render.ReportEnvelope{}, fmt.Errorf(
			"parse %s: incomplete report; save scan output with --format json --max-issues 0, without severity/category filters or --summary-only",
			path,
		)
	}

	return env, nil
}

func diffGatePolicy(cmd *cobra.Command) (string, error) {
	if cmd.Flags().Lookup("gate-policy") == nil {
		return "raw", nil
	}

	return cmd.Flags().GetString("gate-policy")
}

func actionableDiffGate(
	gatePolicy string,
	baseline, current report.UnifiedReportV2,
	d diffrender.Envelope,
) (diffrender.Envelope, error) {
	gateResult := d
	if gatePolicy != "" && gatePolicy != "raw" && gatePolicy != "actionable" {
		return d, errors.New("--gate-policy must be raw or actionable")
	}

	if gatePolicy == "actionable" {
		gateResult.New = render.ActionableIssues(current, d.New)

		gateResult.Delta.NewIssues = len(gateResult.New)
		if d.Delta.ScoreDelta != nil && baseline.Assessment != nil && current.Assessment != nil &&
			baseline.Assessment.Score != nil &&
			current.Assessment.Score != nil {
			delta := *current.Assessment.Score - *baseline.Assessment.Score
			gateResult.Delta.ScoreDelta = &delta
		}
	}

	return gateResult, nil
}
