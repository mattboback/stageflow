package main

import (
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"
	"text/tabwriter"
)

func renderScanners(out io.Writer, response ScannersResponse, format outputFormat) error {
	if format == outputFormatJSON {
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		encoder.SetEscapeHTML(false)

		return encoder.Encode(response)
	}

	if format == outputFormatMarkdown {
		return writeScannersMarkdown(out, response)
	}

	return writeScannersSummary(out, response)
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

func writeScannersMarkdown(out io.Writer, response ScannersResponse) error {
	slices.SortFunc(response.Scanners, func(a, b ScannerInfo) int {
		return strings.Compare(a.ID, b.ID)
	})

	if _, err := fmt.Fprintf(out, "# Scanners\n\nEnabled: %d/%d\n\n", response.Enabled, response.Total); err != nil {
		return err
	}

	if len(response.Scanners) == 0 {
		_, err := fmt.Fprintln(out, "No scanners available.")
		return err
	}

	if _, err := fmt.Fprintln(out, "| ID | Name | Enabled | Categories | Version | Built-In |"); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(out, "| --- | --- | --- | --- | --- | --- |"); err != nil {
		return err
	}

	for _, scanner := range response.Scanners {
		if _, err := fmt.Fprintf(
			out,
			"| %s | %s | %t | %s | %s | %t |\n",
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

	return nil
}
