package main

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

func newProjectUpdateCmd(root *rootOptions) *cobra.Command {
	var (
		name     string
		urls     []string
		scanners []string
	)

	cmd := &cobra.Command{
		Use:   "update <slug>",
		Short: "Update a remote project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			body := make(map[string]any)

			if cmd.Flags().Changed("name") {
				body["name"] = name
			}

			if cmd.Flags().Changed("url") {
				body["urls"] = urls
			}

			if cmd.Flags().Changed("scanner") {
				body["scanners"] = scanners
			}

			if len(body) == 0 {
				return exitCodeError{
					Code: 2,
					Err:  errors.New("at least one flag (--name, --url, --scanner) is required"),
				}
			}

			client := NewClient(root.apiURL, root.apiKey, nil)

			p, err := client.updateProject(cmd.Context(), slug, body)
			if err != nil {
				return exitCodeError{Code: 2, Err: fmt.Errorf("update project: %w", err)}
			}

			format, err := root.outputFormat()
			if err != nil {
				return exitCodeError{Code: 2, Err: err}
			}

			return printProject(cmd, p, format)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Display name")
	cmd.Flags().StringSliceVar(&urls, "url", nil, "Target URL (repeatable; replaces all URLs)")
	cmd.Flags().StringSliceVar(&scanners, "scanner", nil, "Scanner module (repeatable; replaces all scanners)")

	return cmd
}
