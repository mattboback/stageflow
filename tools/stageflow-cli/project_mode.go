package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

func executeRunProjectMode(
	ctx context.Context,
	options runOptions,
	parseInfo runParseInfo,
	getenv getenvFunc,
	stdout, stderr io.Writer,
) int {
	projectRoot, err := resolveProjectRoot(parseInfo.projectPath)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}

	cfg, cfgPath, err := loadProjectConfig(projectRoot)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}

	fmt.Fprintf(stderr, "Using project config: %s\n", cfgPath)

	merged, err := applyProjectConfig(options, parseInfo.setFlags, cfg, getenv)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}

	totalCtx, cancel := context.WithTimeout(ctx, merged.timeout)
	defer cancel()

	err = runCommandSteps(totalCtx, projectRoot, cfg.Dev.Up, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "Dev setup failed: %v\n", err)
		return 2
	}

	proc, err := startDevServer(totalCtx, projectRoot, cfg.Dev.Start, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "Dev start failed: %v\n", err)
		return 2
	}

	cleanupDev := func() {
		stopDevServer(proc, cfg.Dev.Stop, stderr)

		if len(cfg.Dev.Down) > 0 {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cleanupCancel()

			cleanupErr := runCommandSteps(cleanupCtx, projectRoot, cfg.Dev.Down, stderr)
			if cleanupErr != nil {
				fmt.Fprintf(stderr, "Dev teardown failed: %v\n", cleanupErr)
			}
		}
	}
	defer cleanupDev()

	err = waitForReady(totalCtx, proc, cfg.Dev.Ready, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "Dev readiness failed: %v\n", err)
		return 2
	}

	remaining := remainingDuration(totalCtx)
	if remaining <= 0 {
		fmt.Fprintln(stderr, "Run timed out before scan could start")
		return 2
	}

	scanOptions := merged
	scanOptions.timeout = remaining

	err = validateLocalTargets(scanOptions.apiURL, scanOptions.urls)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}

	outWriter, closeOut, err := maybeTeeOutput(stdout, scanOptions.outPath)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}
	defer closeOut()

	return executeRunCommand(totalCtx, scanOptions, outWriter, stderr)
}

func applyProjectConfig(
	flagOptions runOptions,
	setFlags map[string]bool,
	cfg projectConfig,
	getenv getenvFunc,
) (runOptions, error) {
	options := flagOptions

	options.urls = cfg.Scan.URLs

	applyProjectStageflowConfig(&options, setFlags, cfg.Stageflow, getenv)

	if err := applyProjectScanConfig(&options, setFlags, cfg.Scan); err != nil {
		return runOptions{}, err
	}

	applyProjectThresholds(&options, setFlags, cfg.Scan.Thresholds)

	return options, nil
}

func applyProjectStageflowConfig(
	options *runOptions,
	setFlags map[string]bool,
	cfg projectStageflowCfg,
	getenv getenvFunc,
) {
	if !setFlags["api"] && strings.TrimSpace(cfg.APIURL) != "" {
		options.apiURL = strings.TrimSpace(cfg.APIURL)
	}

	if setFlags["api-key"] {
		return
	}

	apiKeyEnv := strings.TrimSpace(cfg.APIKeyEnv)
	if apiKeyEnv == "" {
		return
	}

	options.apiKey = strings.TrimSpace(getenv(apiKeyEnv))
}

func applyProjectScanConfig(options *runOptions, setFlags map[string]bool, cfg projectScanCfg) error {
	if err := applyProjectScanModules(options, setFlags, cfg); err != nil {
		return err
	}

	applyProjectScanToggles(options, setFlags, cfg)

	if err := applyProjectScanTimeout(options, setFlags, cfg); err != nil {
		return err
	}

	if err := applyProjectScanOutput(options, setFlags, cfg); err != nil {
		return err
	}

	return nil
}

func applyProjectScanModules(options *runOptions, setFlags map[string]bool, cfg projectScanCfg) error {
	if setFlags["scanners"] || strings.TrimSpace(cfg.Scanners) == "" {
		return nil
	}

	modules, err := parseModules(cfg.Scanners)
	if err != nil {
		return fmt.Errorf("invalid scan.scanners: %w", err)
	}

	options.modules = modules

	return nil
}

func applyProjectScanToggles(options *runOptions, setFlags map[string]bool, cfg projectScanCfg) {
	if !setFlags["screenshot"] && cfg.Screenshot != nil {
		options.screenshot = *cfg.Screenshot
	}

	if !setFlags["allow-private-targets"] && cfg.AllowPrivateTargets != nil {
		options.allowPrivate = *cfg.AllowPrivateTargets
	}
}

func applyProjectScanTimeout(options *runOptions, setFlags map[string]bool, cfg projectScanCfg) error {
	if setFlags["timeout"] {
		return nil
	}

	d, ok, err := configDuration(cfg.Timeout)
	if err != nil {
		return fmt.Errorf("invalid scan.timeout: %w", err)
	}

	if !ok {
		return nil
	}

	options.timeout = d

	return nil
}

func applyProjectScanOutput(options *runOptions, setFlags map[string]bool, cfg projectScanCfg) error {
	if !setFlags["format"] && strings.TrimSpace(cfg.Format) != "" {
		outputFormat, err := validateReportFormat(cfg.Format)
		if err != nil {
			return err
		}

		options.format = outputFormat
	}

	if !setFlags["max"] && cfg.MaxIssues != nil {
		if *cfg.MaxIssues < 0 {
			return errors.New("scan.max_issues must be >= 0")
		}

		options.maxIssues = *cfg.MaxIssues
	}

	if !setFlags["severity"] && strings.TrimSpace(cfg.Severity) != "" {
		minSeverity, err := parseMinimumSeverity(cfg.Severity)
		if err != nil {
			return err
		}

		options.minSeverity = minSeverity
	}

	return nil
}

func applyProjectThresholds(options *runOptions, setFlags map[string]bool, cfg projectThresholdsCfg) {
	if !setFlags["threshold-critical"] && cfg.Critical != nil {
		options.thresholdCritical = *cfg.Critical
	}

	if !setFlags["threshold-serious"] && cfg.Serious != nil {
		options.thresholdSerious = *cfg.Serious
	}

	if !setFlags["threshold-total"] && cfg.Total != nil {
		options.thresholdTotal = *cfg.Total
	}
}

func remainingDuration(ctx context.Context) time.Duration {
	deadline, ok := ctx.Deadline()
	if !ok {
		return 0
	}

	return time.Until(deadline)
}
