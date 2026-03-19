package main

import (
	"io"

	"github.com/spf13/cobra"
)

type rootOptions struct {
	apiURL           string
	apiKey           string
	outputFormatRaw  string
	jsonOutputCompat bool
}

func newRootCmd(getenv getenvFunc, stdout, stderr io.Writer) *cobra.Command {
	opts := &rootOptions{
		apiURL: envOr(getenv, "STAGEFLOW_API_URL", "http://localhost:8080"),
		apiKey: envOr(getenv, "STAGEFLOW_API_KEY", ""),
	}

	rootCmd := &cobra.Command{
		Use:           "stageflow",
		Short:         "StageFlow CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(*cobra.Command, []string) error {
			_, err := opts.outputFormat()

			return err
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)

	rootCmd.PersistentFlags().StringVar(&opts.apiURL, "api", opts.apiURL, "API base URL")
	rootCmd.PersistentFlags().StringVar(&opts.apiKey, "api-key", opts.apiKey, "API key")
	rootCmd.PersistentFlags().StringVar(
		&opts.outputFormatRaw,
		"format",
		string(outputFormatText),
		"Output format: text, markdown, or json",
	)
	rootCmd.PersistentFlags().BoolVar(
		&opts.jsonOutputCompat,
		"json",
		false,
		"Output JSON instead of plain text",
	)
	cobra.CheckErr(rootCmd.PersistentFlags().MarkDeprecated("json", "use --format=json instead"))

	rootCmd.AddCommand(newScanCmd(opts))
	rootCmd.AddCommand(newAICmd(opts))
	rootCmd.AddCommand(newProjectCmd(opts, getenv))
	rootCmd.AddCommand(newReportCmd(opts))
	rootCmd.AddCommand(newScannersCmd(opts))
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newCompletionCmd())
	rootCmd.AddCommand(newDocsCmd())

	return rootCmd
}

func (r *rootOptions) outputFormat() (outputFormat, error) {
	return normalizeOutputFormat(r.outputFormatRaw, r.jsonOutputCompat)
}
