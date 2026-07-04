package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mattboback/stageflow/clients/cli/internal/exitcode"
	"github.com/mattboback/stageflow/clients/cli/internal/projectmode"
	"github.com/mattboback/stageflow/clients/cli/internal/render"
	"github.com/mattboback/stageflow/clients/cli/internal/urlcheck"
)

type devDoctorEnvelope struct {
	Schema                  string               `json:"schema"`
	ProjectRoot             string               `json:"projectRoot"`
	ConfigPath              string               `json:"configPath"`
	Passed                  bool                 `json:"passed"`
	APIURL                  string               `json:"apiUrl"`
	URLs                    []string             `json:"urls"`
	AutoAllowPrivateTargets bool                 `json:"autoAllowPrivateTargets"`
	RemoteProject           remoteProjectLink    `json:"remoteProject"`
	Checks                  []projectDoctorCheck `json:"checks"`
}

// remoteProjectLink describes the remote project (if any) this repo links to
// via stageflow.project, plus ready-to-run commands for its baseline loop.
type remoteProjectLink struct {
	Configured             bool   `json:"configured"`
	Slug                   string `json:"slug,omitempty"`
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
	stageflowCfg projectmode.StageflowConfig,
	autoAllowPrivateTargets bool,
	checks []projectDoctorCheck,
) error {
	payload := devDoctorEnvelope{
		Schema:                  "stageflow-cli/dev-doctor@v1",
		ProjectRoot:             projectRoot,
		ConfigPath:              configPath,
		Passed:                  allProjectDoctorChecksPassed(checks),
		APIURL:                  apiURL,
		URLs:                    urls,
		AutoAllowPrivateTargets: autoAllowPrivateTargets,
		RemoteProject:           buildRemoteProjectLink(stageflowCfg),
		Checks:                  checks,
	}

	return writeJSONEnvelope(out, payload)
}

func allProjectDoctorChecksPassed(checks []projectDoctorCheck) bool {
	for _, check := range checks {
		switch check.Status {
		case "passed", "skipped":
			continue
		default:
			return false
		}
	}

	return true
}

func buildRemoteProjectLink(cfg projectmode.StageflowConfig) remoteProjectLink {
	slug := strings.TrimSpace(cfg.Project)
	if slug == "" {
		return remoteProjectLink{}
	}

	return remoteProjectLink{
		Configured:             true,
		Slug:                   slug,
		RecommendedScanCommand: fmt.Sprintf("stageflow project scan %s --format json", slug),
		PromoteCommandTemplate: fmt.Sprintf("stageflow project promote %s --job-id <job-id>", slug),
	}
}

func writeJSONEnvelope(out io.Writer, payload any) error {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)

	return encoder.Encode(payload)
}

func loadProjectDoctorConfig(projectRoot, projectArg string) (projectmode.Config, string, error) {
	cfg, cfgPath, err := projectmode.LoadConfig(projectRoot)
	if err == nil {
		return cfg, cfgPath, nil
	}

	var missingErr projectmode.MissingConfigError
	if !errors.As(err, &missingErr) {
		return projectmode.Config{}, "", err
	}

	hint := "stageflow dev init"
	if projectArg != "" {
		hint = fmt.Sprintf("stageflow dev init %s", projectArg)
	}

	return projectmode.Config{}, "", fmt.Errorf("project config not found under %s; run `%s`", projectRoot, hint)
}

func writeProjectDoctorResult(
	out io.Writer,
	projectRoot, cfgPath, apiURL string,
	urls []string,
	stageflowCfg projectmode.StageflowConfig,
	autoAllowPrivateTargets bool,
	checks []projectDoctorCheck,
	format render.Format,
	message string,
) error {
	if format == render.FormatJSON {
		return writeProjectDoctorJSON(
			out,
			projectRoot,
			cfgPath,
			apiURL,
			urls,
			stageflowCfg,
			autoAllowPrivateTargets,
			checks,
		)
	}

	fmt.Fprintln(out, message)

	return nil
}

func runDevDoctorCommand(
	cmd *cobra.Command,
	args []string,
	root *rootOptions,
	getenv getenvFunc,
	opts *devDoctorCmdOptions,
) error {
	if opts.Timeout <= 0 {
		return exitcode.Error{Code: 2, Err: errors.New("--timeout must be > 0")}
	}

	projectArg := ""
	if len(args) == 1 {
		projectArg = args[0]
	}

	projectRoot, err := projectmode.ResolveProjectRoot(projectArg)
	if err != nil {
		return exitcode.Error{Code: 2, Err: err}
	}

	cfg, cfgPath, err := loadProjectDoctorConfig(projectRoot, projectArg)
	if err != nil {
		return exitcode.Error{Code: 2, Err: err}
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "Using project config: %s\n", cfgPath)

	if !opts.SkipDev {
		configReadyErr := ensureProjectConfigReady(projectRoot, cfgPath, cfg)
		if configReadyErr != nil {
			return exitcode.Error{Code: 2, Err: configReadyErr}
		}
	}

	apiURL, _ := resolveProjectStageflow(cmd, root, cfg, getenv)

	_, urls, err := buildProjectSubmitJobRequest(cfg.Scan)
	if err != nil {
		return exitcode.Error{Code: 2, Err: err}
	}

	validateErr := urlcheck.ValidateLocalTargets(apiURL, urls)
	if validateErr != nil {
		return exitcode.Error{Code: 2, Err: validateErr}
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

	format, formatErr := root.renderFormat()
	if formatErr != nil {
		return exitcode.Error{Code: 2, Err: formatErr}
	}

	if opts.SkipDev {
		checks = append(checks, projectDoctorCheck{
			Name:    "dev-readiness",
			Status:  "skipped",
			Message: "Skipped by --skip-dev.",
		})

		return writeProjectDoctorResult(
			cmd.OutOrStdout(),
			projectRoot,
			cfgPath,
			apiURL,
			urls,
			cfg.Stageflow,
			autoAllowPrivateTargets,
			checks,
			format,
			"Doctor checks passed (config + scan preflight).",
		)
	}

	doctorCtx, cancel := context.WithTimeout(cmd.Context(), opts.Timeout)
	defer cancel()

	proc, cleanup, err := runProjectDev(doctorCtx, projectRoot, cfg.Dev, cmd.ErrOrStderr())
	if err != nil {
		return exitcode.Error{Code: 2, Err: err}
	}
	defer cleanup()

	readyErr := projectmode.WaitForReady(doctorCtx, proc, cfg.Dev.Ready, cmd.ErrOrStderr())
	if readyErr != nil {
		return exitcode.Error{Code: 2, Err: fmt.Errorf("dev readiness failed: %w", readyErr)}
	}

	checks = append(checks, projectDoctorCheck{
		Name:    "dev-readiness",
		Status:  "passed",
		Message: "Started the dev command and observed a ready HTTP response.",
	})

	return writeProjectDoctorResult(
		cmd.OutOrStdout(),
		projectRoot,
		cfgPath,
		apiURL,
		urls,
		cfg.Stageflow,
		autoAllowPrivateTargets,
		checks,
		format,
		"Doctor checks passed (config + scan preflight + dev readiness).",
	)
}
