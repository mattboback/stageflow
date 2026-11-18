package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
	"github.com/mattboback/stageflow/clients/cli/internal/exitcode"
	"github.com/mattboback/stageflow/clients/cli/internal/render"
	"github.com/mattboback/stageflow/clients/cli/internal/staticsite"
	"github.com/mattboback/stageflow/clients/cli/internal/urlcheck"
)

func publicDiscoverySeed(args []string) (string, error) {
	if len(args) != 1 || staticsite.IsPathTarget(args[0]) {
		return "", errors.New("discovery requires exactly one public HTTP(S) site URL")
	}

	raw := strings.TrimSpace(args[0])
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}

	parsed, parseErr := url.Parse(raw)
	if parseErr != nil || parsed.User != nil {
		return "", errors.New("discovery needs a valid URL without embedded credentials")
	}

	if strings.HasSuffix(strings.ToLower(parsed.Hostname()), ".localhost") {
		return "", errors.New("discovery supports public targets only")
	}

	urls, err := urlcheck.NormalizeTargets([]string{raw})
	if err != nil {
		return "", err
	}

	if urlcheck.ContainsPrivateTargets(urls) {
		return "", errors.New("discovery supports public targets only")
	}

	return urls[0], nil
}

func newDiscoverCmd(root *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "discover <url>",
		Short: "Preview public website pages from sitemaps and HTML links",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			seed, err := publicDiscoverySeed(args)
			if err != nil {
				return exitcode.Error{Code: 2, Err: err}
			}

			client := apiclient.NewClient(root.apiURL, root.apiKey, nil)

			result, err := client.DiscoverPages(cmd.Context(), seed)
			if err != nil {
				return exitcode.Error{Code: 2, Err: err}
			}

			format, err := root.renderFormat()
			if err != nil {
				return err
			}

			if format == render.FormatJSON {
				err = json.NewEncoder(cmd.OutOrStdout()).Encode(result)
			} else {
				for _, target := range result.URLs {
					if _, err = fmt.Fprintln(cmd.OutOrStdout(), target); err != nil {
						return err
					}
				}

				writeDiscoveryNotice(cmd.ErrOrStderr(), result)
			}

			if err != nil {
				return err
			}

			if len(result.URLs) == 0 {
				return exitcode.Error{Code: 2, Err: errors.New("discovery found no scannable HTML pages")}
			}

			return nil
		},
	}
}
func writeDiscoveryNotice(out io.Writer, result apiclient.DiscoveryResult) {
	fmt.Fprintf(out, "Discovered %d public HTML pages.\n", len(result.URLs))

	if !result.Complete {
		fmt.Fprintln(out, "Discovery coverage is incomplete; only the listed pages will be scanned.")
	}

	for _, warning := range result.Warnings {
		fmt.Fprintln(out, "Discovery:", warning)
	}
}
