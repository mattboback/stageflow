package main

import (
	"time"

	"github.com/spf13/cobra"
)

func newDevCmd(root *rootOptions, getenv getenvFunc) *cobra.Command {
	scanOpts := &devScanCmdOptions{
		Timeout: 10 * time.Minute,
	}
	doctorOpts := &devDoctorCmdOptions{
		Timeout: 2 * time.Minute,
	}

	cmd := &cobra.Command{
		Use:   "dev",
		Short: "Scan your local dev server from .stageflow/config.yaml",
		Long: "Manage the local dev-server scan loop configured in .stageflow/config.yaml.\n\n" +
			"Run `stageflow dev init` once to scaffold the config, `stageflow dev doctor` to\n" +
			"validate it, and `stageflow dev scan` to start your dev server, scan it, and\n" +
			"report the results.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	initCmd := &cobra.Command{
		Use:                   "init [path]",
		Short:                 "Scaffold .stageflow/config.yaml and a setup guide",
		DisableFlagsInUseLine: true,
		Args:                  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDevInitCommand(cmd, args, root)
		},
	}

	doctorCmd := &cobra.Command{
		Use:                   "doctor [path]",
		Short:                 "Validate the config and dev-server readiness without scanning",
		DisableFlagsInUseLine: true,
		Args:                  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDevDoctorCommand(cmd, args, root, getenv, doctorOpts)
		},
	}
	doctorCmd.Flags().DurationVar(
		&doctorOpts.Timeout,
		"timeout",
		doctorOpts.Timeout,
		"Max total time for doctor checks",
	)
	doctorCmd.Flags().BoolVar(&doctorOpts.SkipDev, "skip-dev", false, "Skip starting dev server and readiness checks")

	scanCmd := &cobra.Command{
		Use:                   "scan [path]",
		Short:                 "Start the dev server, scan it, and report results",
		DisableFlagsInUseLine: true,
		Args:                  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDevScanCommand(cmd, args, root, getenv, scanOpts)
		},
	}
	scanCmd.Flags().DurationVar(&scanOpts.Timeout, "timeout", scanOpts.Timeout, "Max total time (dev + scan)")
	bindReportFlags(scanCmd, &scanOpts.Report, true)
	scanCmd.Flags().BoolVar(&scanOpts.NoStream, "no-stream", false, "Poll instead of SSE")
	cobra.CheckErr(scanCmd.Flags().MarkHidden("no-stream"))

	cmd.AddCommand(initCmd, doctorCmd, scanCmd)

	return cmd
}

type devScanCmdOptions struct {
	Timeout  time.Duration
	NoStream bool
	Report   reportCommandOptions
}

type devDoctorCmdOptions struct {
	Timeout time.Duration
	SkipDev bool
}
