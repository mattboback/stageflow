package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/mattboback/stageflow/clients/cli/internal/exitcode"
	"github.com/mattboback/stageflow/clients/cli/internal/projectmode"
	"github.com/mattboback/stageflow/clients/cli/internal/render"
)

type projectScanCmdOptions struct {
	Timeout  time.Duration
	NoStream bool
	Report   render.ReportFlags
}

func newProjectScanCmd(root *rootOptions, getenv getenvFunc) *cobra.Command {
	opts := &projectScanCmdOptions{
		Timeout: 5 * time.Minute,
	}

	cmd := &cobra.Command{
		Use:   "scan [slug]",
		Short: "Scan a remote project and diff against its baseline",
		Long: "Run a scan of a remote project's configured URLs and compare the results\n" +
			"against its promoted baseline.\n\n" +
			"Pass the project slug as an argument, or omit it to use `stageflow.project`\n" +
			"from .stageflow/config.yaml in the current repository.",
		DisableFlagsInUseLine: true,
		Args:                  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProjectScanCommand(cmd, args, root, getenv, opts)
		},
	}

	cmd.Flags().DurationVar(&opts.Timeout, "timeout", opts.Timeout, "Max wait time")
	render.BindReportFlags(cmd, &opts.Report, true)
	cmd.Flags().BoolVar(&opts.NoStream, "no-stream", false, "Poll instead of SSE")
	cobra.CheckErr(cmd.Flags().MarkHidden("no-stream"))

	return cmd
}

func runProjectScanCommand(
	cmd *cobra.Command,
	args []string,
	root *rootOptions,
	getenv getenvFunc,
	opts *projectScanCmdOptions,
) error {
	if opts.Timeout <= 0 {
		return exitcode.Error{Code: 2, Err: errors.New("--timeout must be > 0")}
	}

	if opts.Report.MaxIssues < 0 {
		return exitcode.Error{Code: 2, Err: errors.New("--max-issues must be >= 0")}
	}

	scanRoot := *root

	slug := ""
	if len(args) == 1 {
		slug = strings.TrimSpace(args[0])
	}

	if slug == "" {
		projectRoot, err := resolveProjectScanRoot()
		if err != nil {
			return exitcode.Error{Code: 2, Err: err}
		}

		cfg, cfgPath, err := projectmode.LoadScanConfig(projectRoot)
		if err != nil {
			return exitcode.Error{Code: 2, Err: err}
		}

		slug = strings.TrimSpace(cfg.Stageflow.Project)
		scanRoot.apiURL, scanRoot.apiKey = resolveProjectStageflow(cmd, root, cfg, getenv)

		fmt.Fprintf(cmd.ErrOrStderr(), "Using project %q from %s\n", slug, cfgPath)
	}

	return runRemoteProjectScan(cmd, &scanRoot, slug, opts.Timeout, opts.NoStream, opts.Report)
}

// resolveProjectScanRoot picks where to look for .stageflow/config.yaml when
// no slug is given: the enclosing git root if there is one, else the current
// directory. Unlike the dev commands, not being inside a git repo is fine —
// the missing-config error already tells the user to pass a slug instead.
func resolveProjectScanRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to resolve working directory: %w", err)
	}

	gitRoot, ok, err := projectmode.FindGitRoot(wd)
	if err != nil {
		return "", err
	}

	if ok {
		return gitRoot, nil
	}

	return wd, nil
}
