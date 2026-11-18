package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mattboback/stageflow/clients/cli/internal/buildinfo"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:                   "version",
		Short:                 "Print version information",
		DisableFlagsInUseLine: true,
		Args:                  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintln(cmd.OutOrStdout(), buildinfo.FormatVersion())
		},
	}
}
