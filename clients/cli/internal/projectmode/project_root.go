package projectmode

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func ResolveProjectRoot(projectArg string) (string, error) {
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

	gitRoot, ok, err := FindGitRoot(absStart)
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

func FindGitRoot(startDir string) (string, bool, error) {
	dir := startDir

	for {
		// Support both a real .git directory and a .git file
		// (worktrees/submodules). An empty directory named .git is ignored so
		// temporary parent directories cannot accidentally capture unrelated
		// project paths.
		dotGit := filepath.Join(dir, ".git")

		isGitRoot, err := isGitMarker(dotGit)
		if err != nil {
			return "", false, err
		}

		if isGitRoot {
			return dir, true, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false, nil
		}

		dir = parent
	}
}

func isGitMarker(dotGit string) (bool, error) {
	info, err := os.Stat(dotGit)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("failed to stat %s: %w", dotGit, err)
	}

	if info.IsDir() {
		headPath := filepath.Join(dotGit, "HEAD")

		_, statErr := os.Stat(headPath)
		if statErr == nil {
			return true, nil
		}

		if errors.Is(statErr, os.ErrNotExist) {
			return false, nil
		}

		return false, fmt.Errorf("failed to stat %s: %w", headPath, statErr)
	}

	data, err := os.ReadFile(dotGit)
	if err != nil {
		return false, fmt.Errorf("failed to read %s: %w", dotGit, err)
	}

	return len(data) >= len("gitdir:") && string(data[:len("gitdir:")]) == "gitdir:", nil
}
