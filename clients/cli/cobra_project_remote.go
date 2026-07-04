package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"os"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
	"github.com/mattboback/stageflow/clients/cli/internal/exitcode"
	"github.com/mattboback/stageflow/clients/cli/internal/render"

	"github.com/mattboback/stageflow/clients/cli/internal/projectmode"
	"github.com/mattboback/stageflow/clients/cli/internal/urlcheck"
)

func newProjectCmd(root *rootOptions, getenv getenvFunc) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage remote projects and scan them against their baselines",
		Long: "Manage projects registered on a StageFlow API.\n\n" +
			"A project stores target URLs, scanner configuration, and a promoted baseline\n" +
			"server-side. `stageflow project scan` runs a scan against those URLs and diffs\n" +
			"the results against the baseline, making regressions visible in CI.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(
		newProjectScanCmd(root, getenv),
		newProjectCreateCmd(root),
		newProjectListCmd(root),
		newProjectShowCmd(root),
		newProjectUpdateCmd(root),
		newProjectDeleteCmd(root),
		newProjectPromoteCmd(root),
	)

	return cmd
}

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

func newProjectCreateCmd(root *rootOptions) *cobra.Command {
	var (
		name     string
		urls     []string
		scanners []string
	)

	cmd := &cobra.Command{
		Use:   "create <slug>",
		Short: "Create a remote project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			if name == "" {
				name = slug
			}

			if len(urls) == 0 {
				return exitcode.Error{Code: 2, Err: errors.New("at least one --url is required")}
			}

			normalizedURLs, err := urlcheck.NormalizeTargets(urls)
			if err != nil {
				return exitcode.Error{Code: 2, Err: err}
			}

			validateErr := urlcheck.ValidateLocalTargets(root.apiURL, normalizedURLs)
			if validateErr != nil {
				return exitcode.Error{Code: 2, Err: validateErr}
			}

			normalizedScanners, err := normalizeScannerList(scanners)
			if err != nil {
				return exitcode.Error{Code: 2, Err: err}
			}

			client := newAPICommandClient(root)

			p, err := client.CreateProject(cmd.Context(), slug, name, normalizedURLs, normalizedScanners)
			if err != nil {
				return exitcode.Error{Code: 2, Err: fmt.Errorf("create project: %w", err)}
			}

			format, err := root.renderFormat()
			if err != nil {
				return exitcode.Error{Code: 2, Err: err}
			}

			return printProject(cmd, p, format)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Display name (defaults to slug)")
	cmd.Flags().StringSliceVar(&urls, "url", nil, "Target URL (repeatable)")
	cmd.Flags().StringSliceVar(&scanners, "scanner", nil, "Scanner module (repeatable; omit for all)")

	return cmd
}

func normalizeScannerList(scanners []string) ([]string, error) {
	if len(scanners) == 0 {
		return nil, nil
	}

	normalized := make([]string, 0, len(scanners))
	for _, scanner := range scanners {
		trimmed := strings.TrimSpace(scanner)
		if trimmed == "" {
			return nil, errors.New("scanner list contains an empty module name")
		}

		normalized = append(normalized, trimmed)
	}

	return normalized, nil
}

func newProjectListCmd(root *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List remote projects",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client := newAPICommandClient(root)

			projects, err := client.ListProjects(cmd.Context())
			if err != nil {
				return exitcode.Error{Code: 2, Err: fmt.Errorf("list projects: %w", err)}
			}

			format, err := root.renderFormat()
			if err != nil {
				return exitcode.Error{Code: 2, Err: err}
			}

			if format == render.FormatJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")

				return enc.Encode(projects)
			}

			if len(projects) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No projects found.")

				return nil
			}

			for _, p := range projects {
				baseline := "-"
				if p.BaselineJobID != "" {
					baseline = abbreviateID(p.BaselineJobID, 8)
				}

				fmt.Fprintf(cmd.OutOrStdout(), "%-20s %-30s baseline=%s  urls=%d\n",
					p.Slug, p.Name, baseline, len(p.URLs))
			}

			return nil
		},
	}
}

func newProjectShowCmd(root *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "show <slug>",
		Short: "Show remote project details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPICommandClient(root)

			p, err := client.GetProject(cmd.Context(), args[0])
			if err != nil {
				return exitcode.Error{Code: 2, Err: fmt.Errorf("get project: %w", err)}
			}

			format, err := root.renderFormat()
			if err != nil {
				return exitcode.Error{Code: 2, Err: err}
			}

			return printProject(cmd, p, format)
		},
	}
}

func newProjectDeleteCmd(root *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <slug>",
		Short: "Delete a remote project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPICommandClient(root)

			if err := client.DeleteProject(cmd.Context(), args[0]); err != nil {
				return exitcode.Error{Code: 2, Err: fmt.Errorf("delete project: %w", err)}
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Deleted project %q.\n", args[0])

			return nil
		},
	}
}

func newProjectPromoteCmd(root *rootOptions) *cobra.Command {
	var jobID string

	cmd := &cobra.Command{
		Use:   "promote <slug>",
		Short: "Promote a scan job to be the project baseline",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if jobID == "" {
				return exitcode.Error{Code: 2, Err: errors.New("--job-id is required")}
			}

			client := newAPICommandClient(root)

			if err := client.PromoteBaseline(cmd.Context(), args[0], jobID); err != nil {
				return exitcode.Error{Code: 2, Err: fmt.Errorf("promote baseline: %w", err)}
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Baseline for %q set to job %s.\n", args[0], jobID)

			return nil
		},
	}

	cmd.Flags().StringVar(&jobID, "job-id", "", "Job ID to promote as baseline")

	return cmd
}

func printProject(cmd *cobra.Command, p apiclient.RemoteProject, format render.Format) error {
	if format == render.FormatJSON {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")

		return enc.Encode(p)
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Slug:     %s\n", p.Slug)
	fmt.Fprintf(out, "Name:     %s\n", p.Name)
	fmt.Fprintf(out, "URLs:     %s\n", strings.Join(p.URLs, ", "))

	if len(p.Scanners) > 0 {
		fmt.Fprintf(out, "Scanners: %s\n", strings.Join(p.Scanners, ", "))
	}

	if p.BaselineJobID != "" {
		fmt.Fprintf(out, "Baseline: %s\n", p.BaselineJobID)
	}

	return nil
}

func abbreviateID(id string, maxLen int) string {
	if maxLen <= 0 || len(id) <= maxLen {
		return id
	}

	return id[:maxLen]
}
