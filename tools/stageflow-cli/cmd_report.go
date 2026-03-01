package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
)

func runReportCommand(ctx context.Context, args []string, getenv getenvFunc, stdout, stderr io.Writer) int {
	cmd := flag.NewFlagSet("report", flag.ContinueOnError)
	cmd.SetOutput(stderr)

	apiURL := cmd.String("api", envOr(getenv, "STAGEFLOW_API_URL", "http://localhost:8080"), "API base URL")
	apiKey := cmd.String("api-key", envOr(getenv, "STAGEFLOW_API_KEY", ""), "API key")
	format := cmd.String("format", "summary", "Output format: summary, json, quiet")
	maxIssues := cmd.Int("max", 0, "Max issues to output (0 = unlimited)")
	severity := cmd.String("severity", "minor", "Minimum severity to include: critical, serious, moderate, minor, info")

	parseErr := cmd.Parse(args)
	if parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return 0
		}

		return 2
	}

	rest := cmd.Args()
	if len(rest) != 1 {
		fmt.Fprintln(stderr, "Error: exactly one job ID is required")
		return 2
	}

	outputFormat, err := validateReportFormat(*format)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}

	if *maxIssues < 0 {
		fmt.Fprintln(stderr, "Error: --max must be >= 0")
		return 2
	}

	minSeverity, err := parseMinimumSeverity(*severity)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}

	client := NewClient(*apiURL, *apiKey, nil)
	jobID := rest[0]

	status, statusErr := fetchJobStatus(ctx, client, jobID)
	if statusErr != nil {
		fmt.Fprintf(stderr, "Failed to fetch job status: %v\n", statusErr)
		return 2
	}

	switch status.State {
	case jobStateDone:
	case jobStateFailed:
		fmt.Fprintf(stderr, "Job failed: %s\n", status.Error)
		return 2
	default:
		fmt.Fprintf(stderr, "Job is not completed yet: %s\n", status.State)
		return 2
	}

	doc, reportErr := fetchReport(ctx, client, jobID)
	if reportErr != nil {
		fmt.Fprintf(stderr, "Failed to fetch report: %v\n", reportErr)
		return 2
	}

	renderErr := renderReport(stdout, status, doc, renderOptions{
		format:      outputFormat,
		maxIssues:   *maxIssues,
		minSeverity: minSeverity,
	})
	if renderErr != nil {
		fmt.Fprintf(stderr, "Failed to render report: %v\n", renderErr)
		return 2
	}

	return 0
}
