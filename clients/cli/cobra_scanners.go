package main

import (
	"fmt"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
	"github.com/spf13/cobra"
)

func newScannersCmd(root *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "scanners",
		Short:                 "List available scanners",
		DisableFlagsInUseLine: true,
		Args:                  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client := apiclient.NewClient(root.apiURL, root.apiKey, nil)

			var response apiclient.ScannersResponse
			if err := client.GetJSON(cmd.Context(), "/api/v1/scanners", &response); err != nil {
				return exitCodeError{Code: 2, Err: fmt.Errorf("fetch scanners: %w", err)}
			}

			format, err := root.outputFormat()
			if err != nil {
				return exitCodeError{Code: 2, Err: err}
			}

			renderErr := renderScanners(cmd.OutOrStdout(), response, format)
			if renderErr != nil {
				return exitCodeError{Code: 2, Err: renderErr}
			}

			return nil
		},
	}

	return cmd
}
