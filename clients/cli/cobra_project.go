package main

import (
	"github.com/spf13/cobra"
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
