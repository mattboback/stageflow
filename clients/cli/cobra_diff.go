package main

import (
	"encoding/json"
	"fmt"
	"io"
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
	"github.com/mattboback/stageflow/libs/go/diff"
)

// diffEnvelope is re-exported from internal/diffrender so the rest of the
// main package (notably cobra_scan.go) can continue to reference it unchanged.
type diffEnvelope = diffrender.Envelope

func newDiffCmd(root *rootOptions) *cobra.Command {
	var (
		failOnNew        string
		failOnRegression bool
		timeout          time.Duration
		noStream         bool
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
				noStream,
				failOnNew,
				failOnRegression,
			)
		},
	}

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
	cmd.Flags().DurationVar(&timeout, "timeout", 5*time.Minute, "Max wait time for live scan")
	cmd.Flags().BoolVar(&noStream, "no-stream", false, "Poll instead of SSE for live scan")
	cobra.CheckErr(cmd.Flags().MarkHidden("no-stream"))

	return cmd
}

func runDiffCommand(
	cmd *cobra.Command,
	root *rootOptions,
	baselinePath, currentTarget string,
	timeout time.Duration,
	noStream bool,
	failOnNew string,
	failOnRegression bool,
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
		noStream,
	)
	if err != nil {
		return exitcode.Error{Code: 2, Err: err}
	}

	result := diff.ComputeDiff("", baselineEnv.Report, currentJobID, currentEnv.Report)
	d := diffFromResult(result, baselinePath, currentFile)

	regressed, err := evaluateDiffRegression(d, failOnRegression, failOnNew)
	if err != nil {
		return exitcode.Error{Code: 2, Err: err}
	}

	d.Regressed = regressed

	format, err := root.renderFormat()
	if err != nil {
		return exitcode.Error{Code: 2, Err: err}
	}

	err = renderDiff(cmd.OutOrStdout(), d, format)
	if err != nil {
		return exitcode.Error{Code: 2, Err: err}
	}

	if regressed {
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
	noStream bool,
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

	currentEnv, jobID, err := runLiveDiffScan(cmd, root, baselineEnv, currentTarget, timeout, noStream)
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
	noStream bool,
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

	result, err := scanflow.SubmitURLsAndWait(
		cmd.Context(),
		client,
		req,
		timeout,
		scanflow.WaitOptions{Progress: cmd.ErrOrStderr(), NoStream: noStream},
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

func evaluateDiffRegression(d diffEnvelope, failOnRegression bool, failOnNew string) (bool, error) {
	return diffrender.EvaluateRegression(d, failOnRegression, failOnNew, render.HasIssuesAtOrAbove)
}

func isDiffRegressed(d diffEnvelope) bool {
	return diffrender.IsRegressed(d)
}

func diffFromResult(r diff.Result, baselineFile, currentFile string) diffEnvelope {
	return diffrender.FromResult(r, baselineFile, currentFile)
}

func renderDiff(out io.Writer, d diffEnvelope, format render.Format) error {
	f, err := diffRenderFormat(format)
	if err != nil {
		return err
	}

	return diffrender.Render(out, d, f)
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

	data = apiclient.SanitizeReportJSON(data)

	var env render.ReportEnvelope

	err = json.Unmarshal(data, &env)
	if err != nil {
		return render.ReportEnvelope{}, fmt.Errorf("parse %s: %w", path, err)
	}

	return env, nil
}
