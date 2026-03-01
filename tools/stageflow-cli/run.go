package main

import (
	"context"
	"fmt"
	"io"
)

type getenvFunc func(string) string

func run(args []string, getenv getenvFunc, stdout, stderr io.Writer) int {
	if len(args) < 2 {
		printUsage(stdout)
		return 1
	}

	command := args[1]
	commandArgs := args[2:]

	ctx := context.Background()

	switch command {
	case "run":
		return runRunCommand(ctx, commandArgs, getenv, stdout, stderr)
	case "report":
		return runReportCommand(ctx, commandArgs, getenv, stdout, stderr)
	case "scanners":
		return runScannersCommand(ctx, commandArgs, getenv, stdout, stderr)
	case "help", "-h", "--help":
		printUsage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "Unknown command: %s\n\n", command)
		printUsage(stdout)

		return 1
	}
}

func printUsage(out io.Writer) {
	fmt.Fprintln(out, "StageFlow CLI")
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "Usage:")
	fmt.Fprintln(out, "  stageflow <command> [arguments]")
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "Commands:")
	fmt.Fprintln(out, "  run        Submit a scan job and wait for results")
	fmt.Fprintln(out, "  report     Fetch and display results for an existing job")
	fmt.Fprintln(out, "  scanners   List available scanners")
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "Use 'stageflow <command> -h' for more information about a command.")
}
