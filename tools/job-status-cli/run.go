package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
)

const defaultAPIURL = "http://localhost:8081"

type getenvFunc func(string) string

func run(args []string, getenv getenvFunc, client *http.Client, stdout, stderr io.Writer) int {
	if len(args) < 2 {
		if err := printUsage(stdout); err != nil {
			_, _ = fmt.Fprintf(stderr, "Failed to write usage: %v\n", err)
		}

		return 1
	}

	apiURL := getenv("ORCHESTRATOR_ADMIN_URL")
	if apiURL == "" {
		apiURL = defaultAPIURL
	}

	ctx := context.Background()

	return runCommand(ctx, args[1], args[2:], apiURL, client, stdout, stderr)
}

func runCommand(
	ctx context.Context,
	command string,
	commandArgs []string,
	apiURL string,
	client *http.Client,
	stdout io.Writer,
	stderr io.Writer,
) int {
	switch command {
	case "jobs":
		return runJobsCommand(ctx, commandArgs, apiURL, client, stdout, stderr)
	case "events":
		return runEventsCommand(ctx, commandArgs, apiURL, client, stdout, stderr)
	case "pods":
		return runPodsCommand(ctx, commandArgs, apiURL, client, stdout, stderr)
	case "status":
		return runStatusCommand(ctx, commandArgs, apiURL, client, stdout, stderr)
	case "help", "-h", "--help":
		if err := printUsage(stdout); err != nil {
			_, _ = fmt.Fprintf(stderr, "Failed to write usage: %v\n", err)
		}

		return 0
	default:
		_, _ = fmt.Fprintf(stderr, "Unknown command: %s\n\n", command)
		if err := printUsage(stdout); err != nil {
			_, _ = fmt.Fprintf(stderr, "Failed to write usage: %v\n", err)
		}

		return 1
	}
}

func runEventsCommand(
	ctx context.Context,
	args []string,
	apiURL string,
	client *http.Client,
	stdout io.Writer,
	stderr io.Writer,
) int {
	cmd := flag.NewFlagSet("events", flag.ContinueOnError)
	cmd.SetOutput(stderr)

	limit := cmd.Int("limit", 500, "Maximum number of events to display")
	offset := cmd.Int("offset", 0, "Number of events to skip")
	showPayload := cmd.Bool("payload", false, "Print event payload JSON")

	if err := cmd.Parse(args); err != nil {
		_, _ = fmt.Fprintf(stderr, "Failed to parse events command flags: %v\n", err)

		return 1
	}

	rest := cmd.Args()
	if len(rest) < 1 || rest[0] == "" {
		_, _ = fmt.Fprintln(stderr, "Missing job id")
		if err := printUsage(stdout); err != nil {
			_, _ = fmt.Fprintf(stderr, "Failed to write usage: %v\n", err)
		}

		return 1
	}

	if err := showJobEvents(ctx, client, stdout, apiURL, rest[0], jobEventsOptions{
		limit:       *limit,
		offset:      *offset,
		showPayload: *showPayload,
	}); err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: %v\n", err)

		return 1
	}

	return 0
}

func runJobsCommand(
	ctx context.Context,
	args []string,
	apiURL string,
	client *http.Client,
	stdout io.Writer,
	stderr io.Writer,
) int {
	cmd := flag.NewFlagSet("jobs", flag.ContinueOnError)
	cmd.SetOutput(stderr)

	state := cmd.String(
		"state",
		"",
		"Filter by job state (PENDING, EXTRACTING, READY_TO_SCAN, SCANNING, COMPLETING, DONE, FAILED)",
	)
	limit := cmd.Int("limit", 20, "Maximum number of jobs to display")
	offset := cmd.Int("offset", 0, "Number of jobs to skip")

	if err := cmd.Parse(args); err != nil {
		_, _ = fmt.Fprintf(stderr, "Failed to parse jobs command flags: %v\n", err)

		return 1
	}

	if err := listJobs(ctx, client, stdout, apiURL, jobsOptions{
		state:  *state,
		limit:  *limit,
		offset: *offset,
	}); err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: %v\n", err)

		return 1
	}

	return 0
}

func runPodsCommand(
	ctx context.Context,
	args []string,
	apiURL string,
	client *http.Client,
	stdout io.Writer,
	stderr io.Writer,
) int {
	cmd := flag.NewFlagSet("pods", flag.ContinueOnError)
	cmd.SetOutput(stderr)

	if err := cmd.Parse(args); err != nil {
		_, _ = fmt.Fprintf(stderr, "Failed to parse pods command flags: %v\n", err)

		return 1
	}

	if err := listPods(ctx, client, stdout, apiURL); err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: %v\n", err)

		return 1
	}

	return 0
}

func runStatusCommand(
	ctx context.Context,
	args []string,
	apiURL string,
	client *http.Client,
	stdout io.Writer,
	stderr io.Writer,
) int {
	cmd := flag.NewFlagSet("status", flag.ContinueOnError)
	cmd.SetOutput(stderr)

	if err := cmd.Parse(args); err != nil {
		_, _ = fmt.Fprintf(stderr, "Failed to parse status command flags: %v\n", err)

		return 1
	}

	if err := showSystemStatus(ctx, client, stdout, apiURL); err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: %v\n", err)

		return 1
	}

	return 0
}
