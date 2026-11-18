package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
	"github.com/mattboback/stageflow/clients/cli/internal/exitcode"
	"github.com/mattboback/stageflow/clients/cli/internal/render"
)

func newScannersCmd(root *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "scanners",
		Short:                 "List available scanners",
		DisableFlagsInUseLine: true,
		Args:                  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client := newAPICommandClient(root)

			var response apiclient.ScannersResponse
			if err := client.GetJSON(cmd.Context(), "/api/v1/scanners", &response); err != nil {
				return exitcode.Error{Code: 2, Err: fmt.Errorf("fetch scanners: %w", err)}
			}

			format, err := root.renderFormat()
			if err != nil {
				return exitcode.Error{Code: 2, Err: err}
			}

			renderErr := render.Scanners(cmd.OutOrStdout(), response, format)
			if renderErr != nil {
				return exitcode.Error{Code: 2, Err: renderErr}
			}

			return nil
		},
	}

	return cmd
}
