package main

import "github.com/mattboback/stageflow/clients/cli/internal/projectmode"

func resolveProjectRoot(projectArg string) (string, error) {
	return projectmode.ResolveProjectRoot(projectArg)
}

func findGitRoot(startDir string) (string, bool, error) {
	return projectmode.FindGitRoot(startDir)
}
