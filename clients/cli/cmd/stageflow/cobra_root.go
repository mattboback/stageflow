package main

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/mattboback/stageflow/clients/cli/internal/render"
)

type rootOptions struct {
	apiURL          string
	apiKey          string
	outputFormatRaw string
}

func newRootCmd(getenv getenvFunc, stdout, stderr io.Writer) *cobra.Command {
	opts := &rootOptions{
		apiURL: envOr(getenv, "STAGEFLOW_API_URL", "https://stageflow.org"),
		apiKey: envOr(getenv, "STAGEFLOW_API_KEY", ""),
	}

	rootCmd := &cobra.Command{
		Use:   "stageflow",
		Short: "StageFlow CLI",
		Long: "StageFlow CLI — scan web pages for accessibility, SEO, links, and security issues.\n\n" +
			"There are a few ways to scan:\n\n" +
			"  stageflow scan <url>        one-off scan of any URL\n" +
			"  stageflow scan <dir|zip>    scan a local build directory or ZIP archive\n" +
			"  stageflow diff base.json current.json   compare saved scan reports\n\n" +
			"Exit codes: 0 success, 1 policy failure (--fail-on threshold or regression),\n" +
			"2 usage or API error, or a scan whose report is partial because a scanner\n" +
			"did not finish (see --allow-partial).",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(*cobra.Command, []string) error {
			_, err := opts.renderFormat()

			return err
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)

	rootCmd.PersistentFlags().StringVar(&opts.apiURL, "api", opts.apiURL, "API base URL (env: STAGEFLOW_API_URL)")
	rootCmd.PersistentFlags().StringVar(&opts.apiKey, "api-key", opts.apiKey, "API key (env: STAGEFLOW_API_KEY)")
	rootCmd.PersistentFlags().StringVar(
		&opts.outputFormatRaw,
		"format",
		string(render.FormatText),
		"Output format: text, markdown, or json",
	)

	rootCmd.AddCommand(newScanCmd(opts))
	rootCmd.AddCommand(newCheckCmd(opts))
	rootCmd.AddCommand(newDiscoverCmd(opts))
	rootCmd.AddCommand(newDiffCmd(opts))
	rootCmd.AddCommand(newReportCmd(opts))
	rootCmd.AddCommand(newScannersCmd(opts))
	rootCmd.AddCommand(newAuthCmd(opts))
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newCompletionCmd())

	return rootCmd
}

func (r *rootOptions) renderFormat() (render.Format, error) {
	return render.NormalizeFormat(r.outputFormatRaw)
}
