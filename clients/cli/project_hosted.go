package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mattboback/stageflow/clients/cli/internal/projectmode"
)

func validateHostedProjectConfig(cfg projectConfig) error {
	if cfg.Version != 1 {
		return fmt.Errorf("version must be 1 (got %d)", cfg.Version)
	}

	if strings.TrimSpace(cfg.Stageflow.RemoteProject) == "" {
		return errors.New("stageflow.remote_project is required for hosted project mode")
	}

	if err := validateOptionalHTTPURL("stageflow.api_url", cfg.Stageflow.APIURL); err != nil {
		return err
	}

	if err := validateOptionalHTTPURL("stageflow.remote_api_url", cfg.Stageflow.RemoteAPIURL); err != nil {
		return err
	}

	return nil
}

func loadProjectHostedConfig(projectRoot, projectArg string) (projectConfig, string, error) {
	cfg, cfgPath, err := readProjectConfig(projectRoot)
	if err != nil {
		var missingErr missingProjectConfigError
		if errors.As(err, &missingErr) {
			hint := "stageflow project init"
			if projectArg != "" {
				hint = fmt.Sprintf("stageflow project init %s", projectArg)
			}

			return projectConfig{}, "", fmt.Errorf(
				"project config not found under %s; run `%s`, then set stageflow.remote_project",
				projectRoot,
				hint,
			)
		}

		return projectConfig{}, "", err
	}

	if err := validateHostedProjectConfig(cfg); err != nil {
		return projectConfig{}, "", fmt.Errorf("invalid %s: %w", cfgPath, err)
	}

	return cfg, cfgPath, nil
}

func resolveProjectHostedStageflow(
	cmd *cobra.Command,
	root *rootOptions,
	cfg projectConfig,
	getenv getenvFunc,
) (string, string) {
	apiURL := root.apiURL
	if !cobraFlagChanged(cmd, "api") {
		switch {
		case strings.TrimSpace(cfg.Stageflow.RemoteAPIURL) != "":
			apiURL = strings.TrimSpace(cfg.Stageflow.RemoteAPIURL)
		case strings.TrimSpace(cfg.Stageflow.APIURL) != "":
			apiURL = strings.TrimSpace(cfg.Stageflow.APIURL)
		}
	}

	apiKey := root.apiKey
	if !cobraFlagChanged(cmd, "api-key") && cfg.Stageflow.APIKeyEnv != "" {
		apiKey = getenv(cfg.Stageflow.APIKeyEnv)
	}

	return apiURL, apiKey
}

func runProjectHostedCommand(
	cmd *cobra.Command,
	args []string,
	root *rootOptions,
	getenv getenvFunc,
	opts *projectHostedCmdOptions,
) error {
	if opts.Timeout <= 0 {
		return exitCodeError{Code: 2, Err: errors.New("--timeout must be > 0")}
	}

	if opts.Report.maxIssues < 0 {
		return exitCodeError{Code: 2, Err: errors.New("--max-issues must be >= 0")}
	}

	projectArg := ""
	if len(args) == 1 {
		projectArg = args[0]
	}

	projectRoot, err := projectmode.ResolveProjectRoot(projectArg)
	if err != nil {
		return exitCodeError{Code: 2, Err: err}
	}

	cfg, cfgPath, err := loadProjectHostedConfig(projectRoot, projectArg)
	if err != nil {
		return exitCodeError{Code: 2, Err: err}
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "Using project config: %s\n", cfgPath)

	apiURL, apiKey := resolveProjectHostedStageflow(cmd, root, cfg, getenv)
	hostedRoot := *root
	hostedRoot.apiURL = apiURL
	hostedRoot.apiKey = apiKey

	return runRemoteProjectScan(
		cmd,
		&hostedRoot,
		strings.TrimSpace(cfg.Stageflow.RemoteProject),
		opts.Timeout,
		opts.NoStream,
		opts.Report,
	)
}
