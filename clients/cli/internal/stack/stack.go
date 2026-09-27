// Package stack drives the local Podman Compose stack (the same compose
// files `just dev`/`just demo` use) so `stageflow stack up|down|status` can
// manage a self-hosted StageFlow deployment without requiring `just`.
package stack

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mattboback/stageflow/clients/cli/internal/projectmode"
)

const (
	// EnvDev and EnvLocal are compatibility aliases for the same Compose file.
	EnvDev   = "dev"
	EnvLocal = "local"

	DefaultProject = "stageflow_dev"
	DefaultPodman  = "podman"
)

// Options configures how stack commands locate and drive the compose stack.
type Options struct {
	Root      string
	Env       string
	EnvFile   string
	Project   string
	PodmanBin string
	Services  []string
}

// NewOptions builds Options from environment defaults, matching the
// justfile's COMPOSE_PROJECT_NAME/PODMAN variable fallbacks.
func NewOptions(getenv func(string) string) Options {
	return Options{
		Env:       EnvDev,
		EnvFile:   getenvOr(getenv, "ENV_FILE", ""),
		Project:   getenvOr(getenv, "COMPOSE_PROJECT_NAME", ""),
		PodmanBin: getenvOr(getenv, "PODMAN", DefaultPodman),
	}
}

func getenvOr(getenv func(string) string, key, fallback string) string {
	if getenv == nil {
		return fallback
	}

	if v := getenv(key); v != "" {
		return v
	}

	return fallback
}

// FindRoot locates the StageFlow repo root by walking up from startDir to the
// nearest git root and confirming it contains the compose files this package
// drives.
func FindRoot(startDir string) (string, error) {
	abs, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path for %q: %w", startDir, err)
	}

	gitRoot, ok, err := projectmode.FindGitRoot(abs)
	if err != nil {
		return "", err
	}

	if !ok {
		return "", errors.New("not inside a git repository; run `stageflow stack` from a stageflow checkout")
	}

	composeFile := filepath.Join(gitRoot, "infra", "compose", "podman-compose.yml")
	if _, statErr := os.Stat(composeFile); statErr != nil {
		return "", fmt.Errorf(
			"%s does not look like a stageflow checkout (missing infra/compose/podman-compose.yml)",
			gitRoot,
		)
	}

	return gitRoot, nil
}

// ComposeFiles resolves the sole Compose file. Both compatibility aliases use
// settings from .env, matching the justfile's dev recipe.
func ComposeFiles(root, env string) ([]string, error) {
	switch env {
	case EnvDev, EnvLocal:
	default:
		return nil, fmt.Errorf("env must be %q or %q (got %q)", EnvDev, EnvLocal, env)
	}

	return []string{
		filepath.Join(root, "infra", "compose", "podman-compose.yml"),
	}, nil
}

// ProtectedHostError reports refusal to run against a configured
// STAGEFLOW_PROTECTED_HOST.
type ProtectedHostError struct {
	Host string
}

func (e *ProtectedHostError) Error() string {
	return fmt.Sprintf(
		"refusing to run the local StageFlow stack on protected host %q; "+
			"set STAGEFLOW_ALLOW_VPS_LOCAL_STACKS=1 to override",
		e.Host,
	)
}

// CheckProtectedHost mirrors the justfile's STAGEFLOW_PROTECTED_HOST guard
// (see `just demo`/`just dev`) so this Go code path can't bypass the same
// production-safety check just because it doesn't go through `just`.
func CheckProtectedHost(getenv func(string) string, hostname func() (string, error)) error {
	protected := getenvOr(getenv, "STAGEFLOW_PROTECTED_HOST", "")
	if protected == "" {
		return nil
	}

	if getenvOr(getenv, "STAGEFLOW_ALLOW_VPS_LOCAL_STACKS", "") == "1" {
		return nil
	}

	host, err := hostname()
	if err != nil {
		//nolint:nilerr // can't confirm we're on the protected host; fail open rather than block on this
		return nil
	}

	if host == protected {
		return &ProtectedHostError{Host: host}
	}

	return nil
}
