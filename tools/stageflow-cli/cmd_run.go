package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"
)

type stringSlice []string

type runOptions struct {
	urls              []string
	modules           []string
	screenshot        bool
	allowPrivate      bool
	apiURL            string
	apiKey            string
	timeout           time.Duration
	format            string
	outPath           string
	maxIssues         int
	minSeverity       severityLevel
	thresholdCritical int
	thresholdSerious  int
	thresholdTotal    int
	noStream          bool
}

type runParseInfo struct {
	projectPath string
	setFlags    map[string]bool
}

func (i *stringSlice) String() string {
	return strings.Join(*i, ", ")
}

func (i *stringSlice) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func runRunCommand(ctx context.Context, args []string, getenv getenvFunc, stdout, stderr io.Writer) int {
	options, parseInfo, exitCode, ok := parseRunOptions(args, getenv, stderr)
	if !ok {
		return exitCode
	}

	return executeRun(ctx, options, parseInfo, getenv, stdout, stderr)
}

func parseRunOptions(args []string, getenv getenvFunc, stderr io.Writer) (runOptions, runParseInfo, int, bool) {
	cmd := flag.NewFlagSet("run", flag.ContinueOnError)
	cmd.SetOutput(stderr)

	var urls stringSlice
	cmd.Var(&urls, "url", "Target URL to scan (repeatable). If omitted, uses .stageflow/config.yaml")

	scanners := cmd.String("scanners", "axe", "Comma-separated scanner modules")
	screenshot := cmd.Bool("screenshot", false, "Capture screenshots")
	allowPrivateTargets := cmd.Bool(
		"allow-private-targets",
		false,
		"Allow private/loopback targets (requires API instance to permit it)",
	)

	apiURL := cmd.String("api", envOr(getenv, "STAGEFLOW_API_URL", "http://localhost:8080"), "API base URL")
	apiKey := cmd.String("api-key", envOr(getenv, "STAGEFLOW_API_KEY", ""), "API key")

	timeout := cmd.Duration("timeout", 5*time.Minute, "Max wait time")
	format := cmd.String("format", "summary", "Output format: summary, json, quiet")
	outPath := cmd.String("out", "", "Write output to a file in addition to stdout")
	maxIssues := cmd.Int("max", 0, "Max issues to output (0 = unlimited)")
	severity := cmd.String("severity", "minor", "Minimum severity to include: critical, serious, moderate, minor, info")

	threshCritical := cmd.Int("threshold-critical", -1, "Fail if critical issues exceed N")
	threshSerious := cmd.Int("threshold-serious", -1, "Fail if serious issues exceed N")
	threshTotal := cmd.Int("threshold-total", -1, "Fail if total issues exceed N")
	noStream := cmd.Bool("no-stream", false, "Poll instead of SSE")

	parseErr := cmd.Parse(args)
	if parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return runOptions{}, runParseInfo{}, 0, false
		}

		return runOptions{}, runParseInfo{}, 2, false
	}

	projectArgs := cmd.Args()
	if len(projectArgs) > 1 {
		fmt.Fprintln(stderr, "Error: run accepts at most one positional argument (project path)")
		return runOptions{}, runParseInfo{}, 2, false
	}

	outputFormat, err := validateReportFormat(*format)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return runOptions{}, runParseInfo{}, 2, false
	}

	if *maxIssues < 0 {
		fmt.Fprintln(stderr, "Error: --max must be >= 0")
		return runOptions{}, runParseInfo{}, 2, false
	}

	minSeverity, err := parseMinimumSeverity(*severity)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return runOptions{}, runParseInfo{}, 2, false
	}

	modules, err := parseModules(*scanners)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return runOptions{}, runParseInfo{}, 2, false
	}

	setFlags := map[string]bool{}

	cmd.Visit(func(f *flag.Flag) {
		setFlags[f.Name] = true
	})

	projectPath := ""
	if len(projectArgs) == 1 {
		projectPath = projectArgs[0]
	}

	return runOptions{
			urls:              urls,
			modules:           modules,
			screenshot:        *screenshot,
			allowPrivate:      *allowPrivateTargets,
			apiURL:            *apiURL,
			apiKey:            *apiKey,
			timeout:           *timeout,
			format:            outputFormat,
			outPath:           strings.TrimSpace(*outPath),
			maxIssues:         *maxIssues,
			minSeverity:       minSeverity,
			thresholdCritical: *threshCritical,
			thresholdSerious:  *threshSerious,
			thresholdTotal:    *threshTotal,
			noStream:          *noStream,
		}, runParseInfo{
			projectPath: projectPath,
			setFlags:    setFlags,
		}, 0, true
}

func executeRun(
	ctx context.Context,
	options runOptions,
	parseInfo runParseInfo,
	getenv getenvFunc,
	stdout, stderr io.Writer,
) int {
	if len(options.urls) == 0 {
		return executeRunProjectMode(ctx, options, parseInfo, getenv, stdout, stderr)
	}

	return executeRunScanOnly(ctx, options, stdout, stderr)
}

func executeRunScanOnly(ctx context.Context, options runOptions, stdout, stderr io.Writer) int {
	if err := validateLocalTargets(options.apiURL, options.urls); err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}

	outWriter, closeOut, err := maybeTeeOutput(stdout, options.outPath)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}
	defer closeOut()

	return executeRunCommand(ctx, options, outWriter, stderr)
}

func executeRunCommand(ctx context.Context, options runOptions, stdout, stderr io.Writer) int {
	client := NewClient(options.apiURL, options.apiKey, nil)

	opCtx, cancel := context.WithTimeout(ctx, options.timeout)
	defer cancel()

	req := SubmitJobRequest{
		URLs:                options.urls,
		Modules:             options.modules,
		Screenshot:          options.screenshot,
		AllowPrivateTargets: options.allowPrivate,
	}

	var resp SubmitJobResponse

	submitErr := client.postJSON(opCtx, "/api/v1/jobs/urls", req, &resp)
	if submitErr != nil {
		fmt.Fprintf(stderr, "Failed to submit job: %v\n", submitErr)
		return 2
	}

	jobID := resp.JobID
	fmt.Fprintf(stderr, "Job submitted: %s\nWaiting for completion...\n", jobID)

	waitErr := waitJobState(opCtx, client, jobID, stderr, options.noStream)
	if waitErr != nil {
		fmt.Fprintf(stderr, "Wait failed: %v\n", waitErr)
		return 2
	}

	status, statusErr := fetchJobStatus(opCtx, client, jobID)
	if statusErr != nil {
		fmt.Fprintf(stderr, "Failed to fetch final job status: %v\n", statusErr)
		return 2
	}

	if status.State != jobStateDone {
		fmt.Fprintf(stderr, "Job finished with non-DONE state: %s. Error: %s\n", status.State, status.Error)
		return 2
	}

	doc, reportErr := fetchReport(opCtx, client, jobID)
	if reportErr != nil {
		fmt.Fprintf(stderr, "Failed to fetch report: %v\n", reportErr)
		return 2
	}

	thresholds := evaluateThresholds(
		doc,
		options.thresholdCritical,
		options.thresholdSerious,
		options.thresholdTotal,
	)

	renderErr := renderReport(stdout, status, doc, renderOptions{
		format:      options.format,
		maxIssues:   options.maxIssues,
		minSeverity: options.minSeverity,
		threshold:   thresholds,
	})
	if renderErr != nil {
		fmt.Fprintf(stderr, "Failed to render report: %v\n", renderErr)
		return 2
	}

	if thresholds.Evaluated && !thresholds.Passed {
		return 1
	}

	return 0
}

func envOr(getenv getenvFunc, key, fallback string) string {
	if v := getenv(key); v != "" {
		return v
	}

	return fallback
}

func parseModules(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	modules := strings.Split(raw, ",")

	parsed := make([]string, 0, len(modules))
	for _, item := range modules {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			return nil, errors.New("scanner list contains an empty module name")
		}

		parsed = append(parsed, trimmed)
	}

	return parsed, nil
}
