package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newScannersCmd(root *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "scanners",
		Short:                 "List available scanners",
		DisableFlagsInUseLine: true,
		Args:                  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client := NewClient(root.apiURL, root.apiKey, nil)

			var response ScannersResponse
			if err := client.getJSON(cmd.Context(), "/api/v1/scanners", &response); err != nil {
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
