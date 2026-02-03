package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultAPIURL = "http://localhost:8080"

type getenvFunc func(string) string

func run(args []string, getenv getenvFunc, client *http.Client, stdout, stderr io.Writer) int {
	cmd := flag.NewFlagSet("suite-runner", flag.ContinueOnError)
	cmd.SetOutput(stderr)

	suitePath := cmd.String("suite", "suite.yml", "Path to suite YAML file")
	apiURL := cmd.String("api", envOr(getenv, "PLATFORM_API_BASE_URL", defaultAPIURL), "Platform API base URL")

	if err := cmd.Parse(args[1:]); err != nil {
		_, _ = fmt.Fprintf(stderr, "Failed to parse flags: %v\n", err)

		return 1
	}

	suite, err := loadSuite(*suitePath)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Failed to read suite: %v\n", err)

		return 1
	}

	applyDefaults(&suite)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(suite.TimeoutSec)*time.Second,
	)
	defer cancel()

	outcomes := make([]jobOutcome, 0, len(suite.Domains))
	for _, domain := range suite.Domains {
		jobID, err := submitJob(ctx, client, *apiURL, domain, suite.Modules, suite.Screenshot)
		if err != nil {
			outcomes = append(outcomes, jobOutcome{
				Domain: domain,
				JobID:  "-",
				State:  "FAILED",
				Error:  err.Error(),
				Passed: false,
			})

			continue
		}

		outcome := streamJob(
			ctx,
			client,
			*apiURL,
			domain,
			jobID,
			time.Duration(suite.StreamRetry)*time.Second,
			suite.Thresholds,
		)
		outcomes = append(outcomes, outcome)
	}

	printSummary(outcomes, suite.Thresholds, stdout)

	for _, o := range outcomes {
		if !o.Passed {
			return 1
		}
	}

	return 0
}

func envOr(getenv getenvFunc, key, fallback string) string {
	if v := getenv(key); v != "" {
		return v
	}

	return fallback
}
