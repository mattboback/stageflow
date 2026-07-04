package main

import (
	"github.com/spf13/cobra"

	"github.com/mattboback/stageflow/clients/cli/internal/stack"
)

type stackCmdOptions struct {
	env     string
	project string
}

func newStackCmd(getenv getenvFunc) *cobra.Command {
	cmdOpts := &stackCmdOptions{}

	cmd := &cobra.Command{
		Use:   "stack",
		Short: "Manage the local self-hosted StageFlow compose stack",
		Long: "Start, stop, and inspect the Podman Compose stack used to self-host " +
			"StageFlow locally — the same compose files `just dev`/`just demo` drive.\n\n" +
			"Run from inside a stageflow checkout with `.env` configured and job images " +
			"already built (`just images`); `stageflow stack` manages the compose " +
			"lifecycle, it does not build images or scaffold config.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.PersistentFlags().StringVar(&cmdOpts.env, "env", stack.EnvDev, "Compose overlay: dev or local")
	cmd.PersistentFlags().StringVar(
		&cmdOpts.project,
		"project",
		"",
		"Compose project name (default: $COMPOSE_PROJECT_NAME or stageflow_dev)",
	)

	cmd.AddCommand(
		newStackUpCmd(getenv, cmdOpts),
		newStackDownCmd(getenv, cmdOpts),
		newStackStatusCmd(getenv, cmdOpts),
	)

	return cmd
}

func resolveStackOptions(getenv getenvFunc, cmdOpts *stackCmdOptions, services []string) (stack.Options, error) {
	root, err := stack.FindRoot(".")
	if err != nil {
		return stack.Options{}, err
	}

	opts := stack.NewOptions(getenv)
	opts.Root = root
	opts.Env = cmdOpts.env
	opts.Services = services

	if cmdOpts.project != "" {
		opts.Project = cmdOpts.project
	}

	return opts, nil
}

func newStackUpCmd(getenv getenvFunc, cmdOpts *stackCmdOptions) *cobra.Command {
	return &cobra.Command{
		Use:                   "up [service...]",
		Short:                 "Start the local stack (podman compose up -d)",
		DisableFlagsInUseLine: true,
		Args:                  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts, err := resolveStackOptions(getenv, cmdOpts, args)
			if err != nil {
				return err
			}

			return stack.Up(cmd.Context(), opts, cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
}

func newStackDownCmd(getenv getenvFunc, cmdOpts *stackCmdOptions) *cobra.Command {
	return &cobra.Command{
		Use:                   "down",
		Short:                 "Stop the local stack (podman compose down)",
		DisableFlagsInUseLine: true,
		Args:                  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, err := resolveStackOptions(getenv, cmdOpts, nil)
			if err != nil {
				return err
			}

			return stack.Down(cmd.Context(), opts, cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
}

func newStackStatusCmd(getenv getenvFunc, cmdOpts *stackCmdOptions) *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:                   "status",
		Short:                 "Show compose service status (podman compose ps)",
		DisableFlagsInUseLine: true,
		Args:                  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, err := resolveStackOptions(getenv, cmdOpts, nil)
			if err != nil {
				return err
			}

			return stack.Status(cmd.Context(), opts, cmd.OutOrStdout(), cmd.ErrOrStderr(), jsonOutput)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Print status as JSON")

	return cmd
}
