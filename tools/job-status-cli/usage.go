package main

import (
	"fmt"
	"io"
)

func printUsage(out io.Writer) error {
	_, err := fmt.Fprint(
		out,
		`Job Status CLI - View job submissions and pod orchestration status

Usage:
  job-status-cli <command> [options]

Commands:
  jobs      List all jobs
  events    Show job event timeline
  pods      List all pods
  status    Show system status summary

Job Options:
  -state string    Filter by job state
  -limit int       Maximum number of jobs to display (default: 20)
  -offset int      Number of jobs to skip (default: 0)

Events Options:
  -limit int       Maximum number of events to display (default: 500)
  -offset int      Number of events to skip (default: 0)
  -payload         Print event payload JSON

Environment Variables:
  ORCHESTRATOR_ADMIN_URL  Admin API URL (default: http://localhost:8081)

Examples:
  job-status-cli jobs
  job-status-cli jobs -state SCANNING
  job-status-cli jobs -limit 50
  job-status-cli events <job-id>
  job-status-cli events <job-id> -payload
  job-status-cli pods
  job-status-cli status
		`,
	)

	return err
}
