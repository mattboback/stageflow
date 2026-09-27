package stack

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func composeArgs(opts Options, files []string, action string, extra ...string) []string {
	args := []string{"compose"}
	if opts.Project != "" {
		args = append(args, "-p", opts.Project)
	}

	for _, f := range files {
		args = append(args, "-f", f)
	}

	envFile := opts.EnvFile
	if envFile != "" {
		if !filepath.IsAbs(envFile) {
			envFile = filepath.Join(opts.Root, envFile)
		}

		args = append(args, "--env-file", envFile)
	} else if _, err := os.Stat(filepath.Join(opts.Root, ".env")); err == nil {
		args = append(args, "--env-file", filepath.Join(opts.Root, ".env"))
	}

	args = append(args, action)
	args = append(args, extra...)

	return args
}

//nolint:gosec // stack management intentionally shells out to the operator-controlled podman binary.
func runPodman(ctx context.Context, opts Options, stdout, stderr io.Writer, args ...string) error {
	cmd := exec.CommandContext(ctx, opts.PodmanBin, args...)
	cmd.Dir = opts.Root
	// Match -p during interpolation too, so the default network belongs to this
	// project. Compose still resolves STAGEFLOW_NETWORK_NAME and IMAGE_TAG from
	// the shell and .env using its own precedence rules.
	if opts.Project != "" {
		cmd.Env = append(os.Environ(), "COMPOSE_PROJECT_NAME="+opts.Project)
	}

	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", opts.PodmanBin, strings.Join(args, " "), err)
	}

	return nil
}

type composeSettings struct {
	project string
	network string
	images  []string
}

// composePrerequisites reads the same rendered configuration that up will use.
// In particular, don't reimplement Compose's .env interpolation for network
// names or the per-job image tags (these images aren't Compose services).
func composePrerequisites(ctx context.Context, opts Options, files []string) (composeSettings, error) {
	var rendered bytes.Buffer
	if err := runPodman(ctx, opts, &rendered, io.Discard, composeArgs(opts, files, "config")...); err != nil {
		return composeSettings{}, fmt.Errorf("render compose configuration: %w", err)
	}

	var config struct {
		Name     string `yaml:"name"`
		Networks map[string]struct {
			Name string `yaml:"name"`
		} `yaml:"networks"`
		Services map[string]struct {
			Environment map[string]string `yaml:"environment"`
		} `yaml:"services"`
	}
	if err := yaml.Unmarshal(rendered.Bytes(), &config); err != nil {
		// Rendered configuration contains credentials; never include it or YAML
		// decoding errors (which can contain values) in the diagnostic.
		return composeSettings{}, errors.New("cannot read rendered compose configuration")
	}

	network := config.Networks["stageflow_net"].Name
	env := config.Services["orchestrator"].Environment
	images := []string{env["EXTRACTION_IMAGE"], env["SCANNER_IMAGE"]}

	if config.Name == "" || network == "" || images[0] == "" || images[1] == "" {
		return composeSettings{}, errors.New("compose configuration is missing the project, network or job images")
	}

	return composeSettings{project: config.Name, network: network, images: images}, nil
}

func ensureNetwork(ctx context.Context, opts Options, network string, stderr io.Writer) error {
	//nolint:gosec // see runPodman
	inspect := exec.CommandContext(ctx, opts.PodmanBin, "network", "inspect", network)
	if err := inspect.Run(); err == nil {
		return nil
	}

	fmt.Fprintf(stderr, "==> Creating %s network...\n", network)

	//nolint:gosec // see runPodman
	create := exec.CommandContext(ctx, opts.PodmanBin, "network", "create", network)
	create.Stderr = stderr

	if err := create.Run(); err != nil {
		return fmt.Errorf("create network %s: %w", network, err)
	}

	return nil
}

func missingJobImages(ctx context.Context, opts Options, images []string) []string {
	var missing []string

	for _, image := range images {
		//nolint:gosec // see runPodman
		cmd := exec.CommandContext(ctx, opts.PodmanBin, "image", "exists", image)
		if err := cmd.Run(); err != nil {
			missing = append(missing, image)
		}
	}

	return missing
}

// Up starts the compose stack (`podman compose up -d`), first ensuring the
// compose network exists and the per-job images (built by `just images`,
// not compose) are present.
func Up(ctx context.Context, opts Options, stdout, stderr io.Writer) error {
	if err := CheckProtectedHost(os.Getenv, os.Hostname); err != nil {
		return err
	}

	files, err := ComposeFiles(opts.Root, opts.Env)
	if err != nil {
		return err
	}

	settings, err := composePrerequisites(ctx, opts, files)
	if err != nil {
		return err
	}

	opts.Project = settings.project

	if err = ensureNetwork(ctx, opts, settings.network, stderr); err != nil {
		return err
	}

	if missing := missingJobImages(ctx, opts, settings.images); len(missing) > 0 {
		return fmt.Errorf(
			"missing required job image(s): %s (run `just images` first)",
			strings.Join(missing, ", "),
		)
	}

	upArgs := append([]string{"-d"}, opts.Services...)

	return runPodman(ctx, opts, stdout, stderr, composeArgs(opts, files, "up", upArgs...)...)
}

// Down stops the compose stack (`podman compose down`).
func Down(ctx context.Context, opts Options, stdout, stderr io.Writer) error {
	if err := CheckProtectedHost(os.Getenv, os.Hostname); err != nil {
		return err
	}

	files, err := ComposeFiles(opts.Root, opts.Env)
	if err != nil {
		return err
	}

	return runPodman(ctx, opts, stdout, stderr, composeArgs(opts, files, "down")...)
}

// Status reports compose service state (`podman compose ps`).
func Status(ctx context.Context, opts Options, stdout, stderr io.Writer, jsonOutput bool) error {
	if err := CheckProtectedHost(os.Getenv, os.Hostname); err != nil {
		return err
	}

	files, err := ComposeFiles(opts.Root, opts.Env)
	if err != nil {
		return err
	}

	extra := []string{}

	if jsonOutput {
		extra = append(extra, "--format", "json")
	}

	return runPodman(ctx, opts, stdout, stderr, composeArgs(opts, files, "ps", extra...)...)
}
