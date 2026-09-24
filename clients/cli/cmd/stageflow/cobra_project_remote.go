package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
	"github.com/mattboback/stageflow/clients/cli/internal/exitcode"
	"github.com/mattboback/stageflow/clients/cli/internal/render"

	"github.com/mattboback/stageflow/clients/cli/internal/urlcheck"
)

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
