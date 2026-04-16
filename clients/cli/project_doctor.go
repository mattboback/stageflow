package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/mattboback/stageflow/clients/cli/internal/urlcheck"
	"github.com/spf13/cobra"
)

type projectDoctorEnvelope struct {
	Schema                  string               `json:"schema"`
	ProjectRoot             string               `json:"projectRoot"`
	ConfigPath              string               `json:"configPath"`
	Passed                  bool                 `json:"passed"`
	APIURL                  string               `json:"apiUrl"`
	URLs                    []string             `json:"urls"`
	AutoAllowPrivateTargets bool                 `json:"autoAllowPrivateTargets"`
	HostedMemory            projectHostedMemory  `json:"hostedMemory"`
	Checks                  []projectDoctorCheck `json:"checks"`
}

type projectHostedMemory struct {
	Configured             bool   `json:"configured"`
	ProjectSlug            string `json:"projectSlug,omitempty"`
	APIURL                 string `json:"apiUrl,omitempty"`
	RecommendedScanCommand string `json:"recommendedScanCommand,omitempty"`
	PromoteCommandTemplate string `json:"promoteCommandTemplate,omitempty"`
}

type projectDoctorCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

func writeProjectDoctorJSON(
	out io.Writer,
	projectRoot, configPath, apiURL string,
	urls []string,
	stageflowCfg projectStageflowCfg,
	autoAllowPrivateTargets bool,
	checks []projectDoctorCheck,
) error {
	payload := projectDoctorEnvelope{
		Schema:                  "stageflow-cli/project-doctor@v1",
		ProjectRoot:             projectRoot,
		ConfigPath:              configPath,
		Passed:                  true,
		APIURL:                  apiURL,
		URLs:                    urls,
		AutoAllowPrivateTargets: autoAllowPrivateTargets,
		HostedMemory:            buildProjectHostedMemory(stageflowCfg),
		Checks:                  checks,
	}

	return writeJSONEnvelope(out, payload)
}

func buildProjectHostedMemory(cfg projectStageflowCfg) projectHostedMemory {
	projectSlug := strings.TrimSpace(cfg.RemoteProject)
	apiURL := strings.TrimSpace(cfg.RemoteAPIURL)
	configured := projectSlug != ""

	hosted := projectHostedMemory{
		Configured:  configured,
		ProjectSlug: projectSlug,
		APIURL:      apiURL,
	}
	if !configured {
		return hosted
	}

	hosted.RecommendedScanCommand = buildHostedProjectScanCommand(projectSlug, apiURL)
	hosted.PromoteCommandTemplate = buildHostedProjectPromoteCommand(projectSlug, apiURL)

	return hosted
}

func buildHostedProjectScanCommand(projectSlug string, apiURL string) string {
	command := fmt.Sprintf("stageflow scan --project %s --format json", projectSlug)
	if trimmedAPIURL := strings.TrimSpace(apiURL); trimmedAPIURL != "" {
		command += fmt.Sprintf(" --api %s", trimmedAPIURL)
	}

	return command
}

func buildHostedProjectPromoteCommand(projectSlug string, apiURL string) string {
	command := fmt.Sprintf("stageflow project promote %s --job-id <job-id>", projectSlug)
	if trimmedAPIURL := strings.TrimSpace(apiURL); trimmedAPIURL != "" {
		command += fmt.Sprintf(" --api %s", trimmedAPIURL)
	}

	return command
}

func writeJSONEnvelope(out io.Writer, payload any) error {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)

	return encoder.Encode(payload)
}

//nolint:gocyclo
func runProjectDoctorCommand(
	cmd *cobra.Command,
	args []string,
	root *rootOptions,
	getenv getenvFunc,
	opts *projectDoctorCmdOptions,
) error {
	if opts.Timeout <= 0 {
		return exitCodeError{Code: 2, Err: errors.New("--timeout must be > 0")}
	}

	projectArg := ""
	if len(args) == 1 {
		projectArg = args[0]
	}

	projectRoot, err := resolveProjectRoot(projectArg)
	if err != nil {
		return exitCodeError{Code: 2, Err: err}
	}

	cfg, cfgPath, err := loadProjectConfig(projectRoot)
	if err != nil {
		var missingErr missingProjectConfigError
		if errors.As(err, &missingErr) {
			hint := "stageflow project init"
			if projectArg != "" {
				hint = fmt.Sprintf("stageflow project init %s", projectArg)
			}

			return exitCodeError{
				Code: 2,
				Err:  fmt.Errorf("project config not found under %s; run `%s`", projectRoot, hint),
			}
		}

		return exitCodeError{Code: 2, Err: err}
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "Using project config: %s\n", cfgPath)

	if !opts.SkipDev {
		configReadyErr := ensureProjectConfigReady(projectRoot, cfgPath, cfg)
		if configReadyErr != nil {
			return exitCodeError{Code: 2, Err: configReadyErr}
		}
	}

	apiURL, _ := resolveProjectStageflow(cmd, root, cfg, getenv)

	_, urls, err := buildProjectSubmitJobRequest(cfg.Scan)
	if err != nil {
		return exitCodeError{Code: 2, Err: err}
	}

	validateErr := urlcheck.ValidateLocalTargets(apiURL, urls)
	if validateErr != nil {
		return exitCodeError{Code: 2, Err: validateErr}
	}

	autoAllowPrivateTargets := cfg.Scan.AllowPrivateTargets == nil && urlcheck.ContainsPrivateTargets(urls)
	if autoAllowPrivateTargets {
		fmt.Fprintln(
			cmd.ErrOrStderr(),
			"Detected private/loopback targets; project scans will auto-enable allow_private_targets=true.",
		)
	}

	checks := []projectDoctorCheck{
		{Name: "config", Status: "passed", Message: "Loaded and validated project config."},
		{Name: "scan-preflight", Status: "passed", Message: "Validated scan targets and scanner configuration."},
	}

	format, formatErr := root.outputFormat()
	if formatErr != nil {
		return exitCodeError{Code: 2, Err: formatErr}
	}

	if opts.SkipDev {
		checks = append(checks, projectDoctorCheck{
			Name:    "dev-readiness",
			Status:  "skipped",
			Message: "Skipped by --skip-dev.",
		})

		if format == outputFormatJSON {
			return writeProjectDoctorJSON(cmd.OutOrStdout(), projectRoot, cfgPath, apiURL, urls, cfg.Stageflow, autoAllowPrivateTargets, checks)
		}

		fmt.Fprintln(cmd.OutOrStdout(), "Doctor checks passed (config + scan preflight).")

		return nil
	}

	doctorCtx, cancel := context.WithTimeout(cmd.Context(), opts.Timeout)
	defer cancel()

	proc, cleanup, err := runProjectDev(doctorCtx, projectRoot, cfg.Dev, cmd.ErrOrStderr())
	if err != nil {
		return exitCodeError{Code: 2, Err: err}
	}
	defer cleanup()

	readyErr := waitForReady(doctorCtx, proc, cfg.Dev.Ready, cmd.ErrOrStderr())
	if readyErr != nil {
		return exitCodeError{Code: 2, Err: fmt.Errorf("dev readiness failed: %w", readyErr)}
	}

	checks = append(checks, projectDoctorCheck{
		Name:    "dev-readiness",
		Status:  "passed",
		Message: "Started the dev command and observed a ready HTTP response.",
	})

	if format == outputFormatJSON {
		return writeProjectDoctorJSON(cmd.OutOrStdout(), projectRoot, cfgPath, apiURL, urls, cfg.Stageflow, autoAllowPrivateTargets, checks)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Doctor checks passed (config + scan preflight + dev readiness).")

	return nil
}
