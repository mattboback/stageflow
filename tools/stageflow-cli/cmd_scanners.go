package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"slices"
	"strings"
	"text/tabwriter"
)

func runScannersCommand(ctx context.Context, args []string, getenv getenvFunc, stdout, stderr io.Writer) int {
	apiURL, apiKey, outputFormat, exitCode, ok := parseScannersOptions(args, getenv, stderr)
	if !ok {
		return exitCode
	}

	client := NewClient(apiURL, apiKey, nil)

	var response ScannersResponse

	fetchErr := client.getJSON(ctx, "/api/v1/scanners", &response)
	if fetchErr != nil {
		fmt.Fprintf(stderr, "Failed to fetch scanners: %v\n", fetchErr)
		return 2
	}

	renderErr := renderScanners(stdout, response, outputFormat)
	if renderErr != nil {
		fmt.Fprintf(stderr, "Failed to write scanners output: %v\n", renderErr)
		return 2
	}

	return 0
}

func parseScannersOptions(
	args []string,
	getenv getenvFunc,
	stderr io.Writer,
) (apiURL, apiKey, outputFormat string, exitCode int, ok bool) {
	cmd := flag.NewFlagSet("scanners", flag.ContinueOnError)
	cmd.SetOutput(stderr)

	apiURLFlag := cmd.String("api", envOr(getenv, "STAGEFLOW_API_URL", "http://localhost:8080"), "API base URL")
	apiKeyFlag := cmd.String("api-key", envOr(getenv, "STAGEFLOW_API_KEY", ""), "API key")
	formatFlag := cmd.String("format", "summary", "Output format: summary, json")

	parseErr := cmd.Parse(args)
	if parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return "", "", "", 0, false
		}

		return "", "", "", 2, false
	}

	if len(cmd.Args()) > 0 {
		fmt.Fprintln(stderr, "Error: scanners does not accept positional arguments")
		return "", "", "", 2, false
	}

	outputFormat, err := validateScannersFormat(*formatFlag)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return "", "", "", 2, false
	}

	return *apiURLFlag, *apiKeyFlag, outputFormat, 0, true
}

func renderScanners(out io.Writer, response ScannersResponse, format string) error {
	switch format {
	case outputFormatJSON:
		return writeScannersJSON(out, response)
	case outputFormatSummary:
		return writeScannersSummary(out, response)
	default:
		return fmt.Errorf("unsupported format %q", format)
	}
}

func writeScannersJSON(out io.Writer, response ScannersResponse) error {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)

	return encoder.Encode(response)
}

func writeScannersSummary(out io.Writer, response ScannersResponse) error {
	slices.SortFunc(response.Scanners, func(a, b ScannerInfo) int {
		return strings.Compare(a.ID, b.ID)
	})

	if _, err := fmt.Fprintf(out, "Scanners (enabled %d/%d)\n\n", response.Enabled, response.Total); err != nil {
		return err
	}

	if len(response.Scanners) == 0 {
		_, err := fmt.Fprintln(out, "No scanners available")
		return err
	}

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(w, "ID\tNAME\tENABLED\tCATEGORIES\tVERSION\tBUILT-IN"); err != nil {
		return err
	}

	for _, scanner := range response.Scanners {
		if _, err := fmt.Fprintf(
			w,
			"%s\t%s\t%t\t%s\t%s\t%t\n",
			scanner.ID,
			scanner.Name,
			scanner.Enabled,
			strings.Join(scanner.Categories, ", "),
			scanner.Version,
			scanner.BuiltIn,
		); err != nil {
			return err
		}
	}

	return w.Flush()
}
