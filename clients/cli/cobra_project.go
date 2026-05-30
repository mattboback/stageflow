package main

import (
	"time"

	"github.com/spf13/cobra"
)

func newProjectCmd(root *rootOptions, getenv getenvFunc) *cobra.Command {
	runOpts := &projectCmdOptions{
		Timeout: 10 * time.Minute,
	}
	doctorOpts := &projectDoctorCmdOptions{
		Timeout: 2 * time.Minute,
	}
	hostedOpts := &projectHostedCmdOptions{
		Timeout: 5 * time.Minute,
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
	bindReportFlags(cmd, &runOpts.Report, false)
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

	hostedCmd := &cobra.Command{
		Use:                   "hosted [path]",
		Short:                 "Run hosted project scan from .stageflow config",
		DisableFlagsInUseLine: true,
		Args:                  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProjectHostedCommand(cmd, args, root, getenv, hostedOpts)
		},
	}
	hostedCmd.Flags().DurationVar(&hostedOpts.Timeout, "timeout", hostedOpts.Timeout, "Max wait time")
	bindReportFlags(hostedCmd, &hostedOpts.Report, false)
	hostedCmd.Flags().BoolVar(&hostedOpts.NoStream, "no-stream", false, "Poll instead of SSE")
	cobra.CheckErr(hostedCmd.Flags().MarkHidden("no-stream"))

	cmd.AddCommand(initCmd, doctorCmd)
	cmd.AddCommand(hostedCmd)
	cmd.AddCommand(newProjectRemoteCmd(root)...)

	return cmd
}

type projectCmdOptions struct {
	Timeout  time.Duration
	NoStream bool
	Report   reportCommandOptions
}

type projectDoctorCmdOptions struct {
	Timeout time.Duration
	SkipDev bool
}

type projectHostedCmdOptions struct {
	Timeout  time.Duration
	NoStream bool
	Report   reportCommandOptions
}
