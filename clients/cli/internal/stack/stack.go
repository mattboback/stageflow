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
	// EnvDev pairs the base compose file with podman-compose.test.yml.
	EnvDev = "dev"
	// EnvLocal pairs the base compose file with podman-compose.local.yml.
	EnvLocal = "local"

	DefaultProject = "stageflow_dev"
	DefaultPodman  = "podman"
)

// RequiredJobImages are built by `just images` (not by `compose up --build`)
// because they run as per-job pods launched by the orchestrator, not as
// long-lived compose services.
var RequiredJobImages = []string{
	"localhost/stageflow/extractor:latest",
	"localhost/stageflow/scanner-runner:latest",
}

// Options configures how stack commands locate and drive the compose stack.
type Options struct {
	Root      string
	Env       string
	Project   string
	PodmanBin string
	Services  []string
}

func (o Options) networkName() string {
	return o.Project + "_net"
}

// NewOptions builds Options from environment defaults, matching the
// justfile's COMPOSE_PROJECT_NAME/PODMAN variable fallbacks.
func NewOptions(getenv func(string) string) Options {
	return Options{
		Env:       EnvDev,
		Project:   getenvOr(getenv, "COMPOSE_PROJECT_NAME", DefaultProject),
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

// ComposeFiles resolves the base + overlay compose files for env, matching
// the justfile's `dev` recipe (dev -> podman-compose.test.yml, local ->
// podman-compose.local.yml).
func ComposeFiles(root, env string) ([]string, error) {
	var overlay string

	switch env {
	case EnvDev:
		overlay = "podman-compose.test.yml"
	case EnvLocal:
		overlay = "podman-compose.local.yml"
	default:
		return nil, fmt.Errorf("env must be %q or %q (got %q)", EnvDev, EnvLocal, env)
	}

	return []string{
		filepath.Join(root, "infra", "compose", "podman-compose.yml"),
		filepath.Join(root, "infra", "compose", overlay),
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
