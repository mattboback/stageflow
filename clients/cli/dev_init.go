package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mattboback/stageflow/clients/cli/internal/exitcode"
	"github.com/mattboback/stageflow/clients/cli/internal/projectmode"
	"github.com/mattboback/stageflow/clients/cli/internal/render"
)

type devInitEnvelope struct {
	Schema      string   `json:"schema"`
	ProjectRoot string   `json:"projectRoot"`
	ConfigPath  string   `json:"configPath"`
	GuidePath   string   `json:"guidePath"`
	Created     bool     `json:"created"`
	NextSteps   []string `json:"nextSteps"`
}

func loadOrBootstrapProjectConfig(
	projectRoot string,
	apiURL string,
	out io.Writer,
) (projectmode.Config, string, bool, error) {
	cfg, cfgPath, err := projectmode.LoadConfig(projectRoot)
	if err == nil {
		return cfg, cfgPath, false, nil
	}

	var missingErr projectmode.MissingConfigError
	if !errors.As(err, &missingErr) {
		return projectmode.Config{}, "", false, err
	}

	configPath, guidePath, _, scaffoldErr := scaffoldProjectBootstrap(projectRoot, apiURL)
	if scaffoldErr != nil {
		return projectmode.Config{}, "", false, scaffoldErr
	}

	printProjectBootstrapHelp(out, projectRoot, configPath, guidePath)

	return projectmode.Config{}, "", true, nil
}

func scaffoldProjectBootstrap(projectRoot string, apiURL string) (string, string, bool, error) {
	configPath := filepath.Join(projectRoot, ".stageflow", "config.yaml")
	guidePath := filepath.Join(projectRoot, ".stageflow", "README.md")

	configExists, err := pathExists(configPath)
	if err != nil {
		return "", "", false, fmt.Errorf("stat %s: %w", configPath, err)
	}

	guideExists, err := pathExists(guidePath)
	if err != nil {
		return "", "", false, fmt.Errorf("stat %s: %w", guidePath, err)
	}

	configPath, err = projectmode.ScaffoldConfig(projectRoot, apiURL)
	if err != nil {
		return "", "", false, err
	}

	guidePath, err = projectmode.ScaffoldGuide(projectRoot)
	if err != nil {
		return "", "", false, err
	}

	return configPath, guidePath, !configExists || !guideExists, nil
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}

	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	return false, err
}

func printProjectBootstrapHelp(
	out io.Writer,
	projectRoot string,
	configPath string,
	guidePath string,
) {
	relPath, err := filepath.Rel(projectRoot, configPath)
	if err != nil {
		relPath = configPath
	}

	relGuidePath, relGuideErr := filepath.Rel(projectRoot, guidePath)
	if relGuideErr != nil {
		relGuidePath = guidePath
	}

	fmt.Fprintln(out, "Created StageFlow dev-loop bootstrap:")
	fmt.Fprintf(out, "- %s\n", configPath)
	fmt.Fprintf(out, "- %s\n\n", guidePath)
	fmt.Fprintln(out, "Next steps:")
	fmt.Fprintf(out, "1. Read %s\n", relGuidePath)
	fmt.Fprintf(out, "2. Edit %s and update:\n", relPath)
	fmt.Fprintln(out, "   - dev.start.cmd (your dev start command)")
	fmt.Fprintln(out, "   - dev.ready.url (the URL your dev server serves)")
	fmt.Fprintln(out, "   - scan.urls (the pages you want scanned)")
	fmt.Fprintln(out, "3. For localhost/private target scans, run your local StageFlow stack:")
	fmt.Fprintln(out, "   - just dev up local")
	fmt.Fprintln(out, "   - just dev init local")
	fmt.Fprintln(out, "4. Re-run: stageflow dev scan")
}

func hasScaffoldPlaceholderDevCommand(cfg projectmode.Config) bool {
	for _, item := range cfg.Dev.Start.Cmd {
		if strings.TrimSpace(item) == projectmode.ScaffoldDevStartCommandPlaceholder {
			return true
		}
	}

	return false
}

func ensureProjectConfigReady(projectRoot string, cfgPath string, cfg projectmode.Config) error {
	if !hasScaffoldPlaceholderDevCommand(cfg) {
		return nil
	}

	guidePath := filepath.Join(projectRoot, ".stageflow", "README.md")

	return fmt.Errorf(
		"project config is not set up yet: update dev.start.cmd in %s, then follow %s",
		cfgPath,
		guidePath,
	)
}

func writeDevInitJSON(
	out io.Writer,
	projectRoot, configPath, guidePath string,
	created bool,
) error {
	payload := devInitEnvelope{
		Schema:      "stageflow-cli/dev-init@v1",
		ProjectRoot: projectRoot,
		ConfigPath:  configPath,
		GuidePath:   guidePath,
		Created:     created,
		NextSteps: []string{
			"Read .stageflow/README.md.",
			"Update dev.start.cmd, dev.ready.url, and scan.urls in .stageflow/config.yaml.",
			"Optional: set stageflow.project to link a remote project for baseline diffs.",
			"For localhost/private scans, start the local StageFlow stack with just dev up local and just dev init local.",
			"Run stageflow dev doctor, then stageflow dev scan.",
		},
	}

	return writeJSONEnvelope(out, payload)
}

func cobraFlagChanged(cmd *cobra.Command, name string) bool {
	if flag := cmd.Flags().Lookup(name); flag != nil && flag.Changed {
		return true
	}

	if flag := cmd.InheritedFlags().Lookup(name); flag != nil && flag.Changed {
		return true
	}

	return false
}
func runDevInitCommand(
	cmd *cobra.Command,
	args []string,
	root *rootOptions,
) error {
	projectArg := ""
	if len(args) == 1 {
		projectArg = args[0]
	}

	projectRoot, err := projectmode.ResolveProjectRoot(projectArg)
	if err != nil {
		return exitcode.Error{Code: 2, Err: err}
	}

	configPath, guidePath, created, err := scaffoldProjectBootstrap(projectRoot, root.apiURL)
	if err != nil {
		return exitcode.Error{Code: 2, Err: err}
	}

	format, formatErr := root.renderFormat()
	if formatErr != nil {
		return exitcode.Error{Code: 2, Err: formatErr}
	}

	if !created {
		if format == render.FormatJSON {
			return writeDevInitJSON(cmd.OutOrStdout(), projectRoot, configPath, guidePath, false)
		}

		fmt.Fprintln(cmd.OutOrStdout(), "StageFlow project bootstrap already exists:")
		fmt.Fprintf(cmd.OutOrStdout(), "- %s\n", configPath)
		fmt.Fprintf(cmd.OutOrStdout(), "- %s\n", guidePath)

		return nil
	}

	if format == render.FormatJSON {
		return writeDevInitJSON(cmd.OutOrStdout(), projectRoot, configPath, guidePath, true)
	}

	printProjectBootstrapHelp(cmd.OutOrStdout(), projectRoot, configPath, guidePath)

	return nil
}
