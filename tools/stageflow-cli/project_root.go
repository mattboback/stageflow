package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func resolveProjectRoot(projectArg string) (string, error) {
	startDir := projectArg
	if startDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to resolve working directory: %w", err)
		}

		startDir = wd
	}

	absStart, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path for %q: %w", startDir, err)
	}

	info, err := os.Stat(absStart)
	if err != nil {
		return "", fmt.Errorf("failed to stat project path %q: %w", absStart, err)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("project path is not a directory: %s", absStart)
	}

	gitRoot, ok, err := findGitRoot(absStart)
	if err != nil {
		return "", err
	}

	if ok {
		return gitRoot, nil
	}

	if projectArg == "" {
		return "", errors.New("current directory is not inside a git repo; pass an explicit project path")
	}

	return absStart, nil
}

func findGitRoot(startDir string) (string, bool, error) {
	dir := startDir

	for {
		// Support both a .git directory and a .git file (worktrees/submodules).
		dotGit := filepath.Join(dir, ".git")
		if _, err := os.Stat(dotGit); err == nil {
			return dir, true, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", false, fmt.Errorf("failed to stat %s: %w", dotGit, err)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false, nil
		}

		dir = parent
	}
}
