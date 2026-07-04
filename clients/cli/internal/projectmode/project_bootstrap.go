package projectmode

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type BootstrapSuggestion struct {
	Command            string
	Cwd                string
	CommandSource      string
	Cleanup            string
	CleanupSource      string
	URL                string
	IsDedicatedCommand bool
	IsPlaceholder      bool
}

type projectPackageJSON struct {
	PackageManager string            `json:"packageManager"`
	Scripts        map[string]string `json:"scripts"`
}

type bootstrapCommandCandidate struct {
	recipe    string
	script    string
	command   string
	source    string
	dedicated bool
}

var justRecipePrefixes = map[string]*regexp.Regexp{
	"stageflow-dev":  regexp.MustCompile(`^stageflow-dev\b[^:]*:`),
	"stageflow-down": regexp.MustCompile(`^stageflow-down\b[^:]*:`),
	"dev":            regexp.MustCompile(`^dev\b[^:]*:`),
	"dev-web":        regexp.MustCompile(`^dev-web\b[^:]*:`),
	"run":            regexp.MustCompile(`^run\b[^:]*:`),
}

func DetectBootstrapSuggestion(projectRoot string) (BootstrapSuggestion, error) {
	suggestion := BootstrapSuggestion{
		URL: guessProjectDevURL(projectRoot),
	}

	command, cwd, source, dedicated, found, err := detectProjectBootstrapCommand(projectRoot)
	if err != nil {
		return BootstrapSuggestion{}, err
	}

	if found {
		suggestion.Command = command
		suggestion.Cwd = cwd
		suggestion.CommandSource = source
		suggestion.IsDedicatedCommand = dedicated
	}

	if suggestion.Command == "" {
		suggestion.Command = ScaffoldDevStartCommandPlaceholder
		suggestion.CommandSource = "set `dev.start.cmd` to the command that makes your app reachable locally"
		suggestion.IsPlaceholder = true

		return suggestion, nil
	}

	if !suggestion.IsDedicatedCommand {
		return suggestion, nil
	}

	cleanup, cleanupSource, found, err := detectProjectBootstrapCleanup(projectRoot)
	if err != nil {
		return BootstrapSuggestion{}, err
	}

	if found {
		suggestion.Cleanup = cleanup
		suggestion.CleanupSource = cleanupSource
	}

	return suggestion, nil
}

func detectProjectBootstrapCommand(projectRoot string) (string, string, string, bool, bool, error) {
	candidates := []bootstrapCommandCandidate{
		{
			recipe:  "run",
			command: "just run clients/web",
			source:  "best guess from Justfile recipe `run`",
		},
		{
			recipe:    "stageflow-dev",
			command:   "just stageflow-dev",
			source:    "detected Justfile recipe `stageflow-dev`",
			dedicated: true,
		},
		{
			script:    "stageflow:dev",
			source:    "detected package.json script `stageflow:dev`",
			dedicated: true,
		},
		{
			recipe:  "dev",
			command: "just dev",
			source:  "best guess from Justfile recipe `dev`",
		},
		{
			recipe:  "dev-web",
			command: "just dev-web",
			source:  "best guess from Justfile recipe `dev-web`",
		},
		{
			script: "dev",
			source: "best guess from package.json script `dev`",
		},
		{
			script: "start",
			source: "best guess from package.json script `start`",
		},
	}

	for _, candidate := range candidates {
		command, cwd, found, err := detectBootstrapCommandCandidate(projectRoot, candidate)
		if err != nil {
			return "", "", "", false, false, err
		}

		if found {
			return command, cwd, candidate.source, candidate.dedicated, true, nil
		}
	}

	return "", "", "", false, false, nil
}

func detectBootstrapCommandCandidate(
	projectRoot string,
	candidate bootstrapCommandCandidate,
) (string, string, bool, error) {
	if candidate.recipe == "run" {
		command, found, err := detectJustRunFrontend(projectRoot)
		if err != nil || !found {
			return "", "", false, err
		}

		return command, ".", true, nil
	}

	if candidate.recipe != "" {
		hasRecipe, err := repoHasJustRecipe(projectRoot, candidate.recipe)
		if err != nil {
			return "", "", false, err
		}

		if hasRecipe {
			return candidate.command, ".", true, nil
		}

		return "", "", false, nil
	}

	if candidate.script == "" {
		return "", "", false, nil
	}

	command, cwd, found, err := detectPackageScriptCommand(projectRoot, candidate.script)
	if err != nil || !found {
		return "", "", false, err
	}

	return command, cwd, true, nil
}

func detectProjectBootstrapCleanup(projectRoot string) (string, string, bool, error) {
	hasRecipe, err := repoHasJustRecipe(projectRoot, "stageflow-down")
	if err != nil {
		return "", "", false, err
	}

	if hasRecipe {
		return "just stageflow-down", "detected Justfile recipe `stageflow-down`", true, nil
	}

	command, _, found, err := detectPackageScriptCommand(projectRoot, "stageflow:down")
	if err != nil {
		return "", "", false, err
	}

	if found {
		return command, "detected package.json script `stageflow:down`", true, nil
	}

	return "", "", false, nil
}

func repoHasJustRecipe(projectRoot string, recipe string) (bool, error) {
	pattern, ok := justRecipePrefixes[recipe]
	if !ok {
		return false, fmt.Errorf("unsupported recipe lookup %q", recipe)
	}

	justfilePath, ok, err := findJustfile(projectRoot)
	if err != nil || !ok {
		return false, err
	}

	file, err := os.Open(justfilePath)
	if err != nil {
		return false, fmt.Errorf("open %s: %w", justfilePath, err)
	}

	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continue
		}

		if pattern.MatchString(strings.TrimSpace(line)) {
			return true, nil
		}
	}

	scanErr := scanner.Err()
	if scanErr != nil {
		return false, fmt.Errorf("scan %s: %w", justfilePath, scanErr)
	}

	return false, nil
}

func findJustfile(projectRoot string) (string, bool, error) {
	candidates := []string{
		filepath.Join(projectRoot, "Justfile"),
		filepath.Join(projectRoot, "justfile"),
	}

	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return candidate, true, nil
		}

		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", false, fmt.Errorf("stat %s: %w", candidate, err)
		}
	}

	return "", false, nil
}

func detectJustRunFrontend(projectRoot string) (string, bool, error) {
	hasRunRecipe, err := repoHasJustRecipe(projectRoot, "run")
	if err != nil || !hasRunRecipe {
		return "", false, err
	}

	// Check common frontend directory locations. "clients/web" is the
	// StageFlow monorepo convention; "frontend" is a common generic pattern.
	frontendCandidates := []struct {
		dir     string
		command string
	}{
		{"clients/web", "just run clients/web"},
		{"frontend", "just run frontend"}, // stale-vocab-ok: generic fallback for non-StageFlow projects
	}

	for _, c := range frontendCandidates {
		root := filepath.Join(projectRoot, c.dir)

		packageJSON, ok, loadErr := loadProjectPackageJSON(root)
		if loadErr != nil {
			return "", false, loadErr
		}

		if !ok {
			continue
		}

		if _, exists := packageJSON.Scripts["dev"]; exists {
			return c.command, true, nil
		}
	}

	return "", false, nil
}

func detectPackageScriptCommand(projectRoot string, script string) (string, string, bool, error) {
	for _, subdir := range []string{".", "frontend", "client", "web", "app"} {
		targetDir := projectRoot
		if subdir != "." {
			targetDir = filepath.Join(projectRoot, subdir)
		}

		packageJSON, ok, err := loadProjectPackageJSON(targetDir)
		if err != nil {
			return "", "", false, err
		}

		if !ok {
			continue
		}

		if _, exists := packageJSON.Scripts[script]; exists {
			runner := detectProjectPackageRunner(targetDir, packageJSON)
			return formatPackageScriptCommand(runner, script), subdir, true, nil
		}
	}

	return "", "", false, nil
}

func loadProjectPackageJSON(projectRoot string) (projectPackageJSON, bool, error) {
	packageJSONPath := filepath.Join(projectRoot, "package.json")

	data, err := os.ReadFile(packageJSONPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return projectPackageJSON{}, false, nil
		}

		return projectPackageJSON{}, false, fmt.Errorf("read %s: %w", packageJSONPath, err)
	}

	var packageJSON projectPackageJSON

	unmarshalErr := json.Unmarshal(data, &packageJSON)
	if unmarshalErr != nil {
		return projectPackageJSON{}, false, fmt.Errorf("parse %s: %w", packageJSONPath, unmarshalErr)
	}

	if packageJSON.Scripts == nil {
		packageJSON.Scripts = map[string]string{}
	}

	return packageJSON, true, nil
}

func detectProjectPackageRunner(projectRoot string, packageJSON projectPackageJSON) string {
	if name := strings.ToLower(strings.TrimSpace(packageJSON.PackageManager)); name != "" {
		manager, _, _ := strings.Cut(name, "@")
		switch manager {
		case "bun", "pnpm", "yarn", "npm":
			return manager
		}
	}

	lockfileOrder := []struct {
		Name   string
		Runner string
	}{
		{Name: "bun.lock", Runner: "bun"},
		{Name: "bun.lockb", Runner: "bun"},
		{Name: "pnpm-lock.yaml", Runner: "pnpm"},
		{Name: "yarn.lock", Runner: "yarn"},
		{Name: "package-lock.json", Runner: "npm"},
	}

	for _, lockfile := range lockfileOrder {
		if _, err := os.Stat(filepath.Join(projectRoot, lockfile.Name)); err == nil {
			return lockfile.Runner
		}
	}

	return "npm"
}

func formatPackageScriptCommand(runner string, script string) string {
	switch runner {
	case "bun":
		return fmt.Sprintf("bun run %s", script)
	case "pnpm":
		return fmt.Sprintf("pnpm run %s", script)
	case "yarn":
		return fmt.Sprintf("yarn run %s", script)
	default:
		return fmt.Sprintf("npm run %s", script)
	}
}

func guessProjectDevURL(projectRoot string) string {
	for _, subdir := range []string{"clients/web", "frontend"} {
		candidate := filepath.Join(projectRoot, subdir)
		if candidate != projectRoot && repoLooksLikeViteProject(candidate) {
			return "http://127.0.0.1:5173"
		}
	}

	if repoLooksLikeViteProject(projectRoot) {
		return "http://127.0.0.1:5173"
	}

	return "http://127.0.0.1:3000"
}

func repoLooksLikeViteProject(projectRoot string) bool {
	skippedDirs := map[string]struct{}{
		".git":         {},
		".stageflow":   {},
		"node_modules": {},
		"dist":         {},
		"build":        {},
		"coverage":     {},
	}

	stopWalk := errors.New("stop project walk")

	err := filepath.WalkDir(projectRoot, func(_ string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() {
			if _, skip := skippedDirs[entry.Name()]; skip {
				return filepath.SkipDir
			}

			return nil
		}

		name := entry.Name()
		if strings.HasPrefix(name, "vite.config.") ||
			strings.HasPrefix(name, "astro.config.") ||
			strings.HasPrefix(name, "svelte.config.") {
			return stopWalk
		}

		return nil
	})

	if err != nil && !errors.Is(err, stopWalk) {
		return false
	}

	return errors.Is(err, stopWalk)
}
