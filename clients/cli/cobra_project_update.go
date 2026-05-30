package main

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mattboback/stageflow/clients/cli/internal/urlcheck"
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

			body, bodyErr := buildProjectUpdateBody(cmd, root, name, urls, scanners)
			if bodyErr != nil {
				return bodyErr
			}

			client := newAPICommandClient(root)

			p, err := client.UpdateProject(cmd.Context(), slug, body)
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

func buildProjectUpdateBody(
	cmd *cobra.Command,
	root *rootOptions,
	name string,
	urls []string,
	scanners []string,
) (map[string]any, error) {
	body := make(map[string]any)

	if cmd.Flags().Changed("name") {
		body["name"] = name
	}

	if cmd.Flags().Changed("url") {
		normalizedURLs, normalizeErr := urlcheck.NormalizeTargets(urls)
		if normalizeErr != nil {
			return nil, exitCodeError{Code: 2, Err: normalizeErr}
		}

		validateErr := urlcheck.ValidateLocalTargets(root.apiURL, normalizedURLs)
		if validateErr != nil {
			return nil, exitCodeError{Code: 2, Err: validateErr}
		}

		body["urls"] = normalizedURLs
	}

	if cmd.Flags().Changed("scanner") {
		normalizedScanners, normalizeErr := normalizeProjectScannerFlags(scanners)
		if normalizeErr != nil {
			return nil, exitCodeError{Code: 2, Err: normalizeErr}
		}

		body["scanners"] = normalizedScanners
	}

	if len(body) == 0 {
		return nil, exitCodeError{
			Code: 2,
			Err:  errors.New("at least one flag (--name, --url, --scanner) is required"),
		}
	}

	return body, nil
}
