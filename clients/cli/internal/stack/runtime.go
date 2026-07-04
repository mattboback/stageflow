package stack

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func composeArgs(opts Options, files []string, action string, extra ...string) []string {
	args := []string{"compose", "-p", opts.Project}

	for _, f := range files {
		args = append(args, "-f", f)
	}

	envFile := filepath.Join(opts.Root, ".env")
	if _, err := os.Stat(envFile); err == nil {
		args = append(args, "--env-file", envFile)
	}

	args = append(args, action)
	args = append(args, extra...)

	return args
}

//nolint:gosec // stack management intentionally shells out to the operator-controlled podman binary.
func runPodman(ctx context.Context, opts Options, stdout, stderr io.Writer, args ...string) error {
	cmd := exec.CommandContext(ctx, opts.PodmanBin, args...)
	cmd.Dir = opts.Root
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", opts.PodmanBin, strings.Join(args, " "), err)
	}

	return nil
}

func ensureNetwork(ctx context.Context, opts Options, stderr io.Writer) error {
	//nolint:gosec // see runPodman
	inspect := exec.CommandContext(ctx, opts.PodmanBin, "network", "inspect", opts.networkName())
	if err := inspect.Run(); err == nil {
		return nil
	}

	fmt.Fprintf(stderr, "==> Creating %s network...\n", opts.networkName())

	//nolint:gosec // see runPodman
	create := exec.CommandContext(ctx, opts.PodmanBin, "network", "create", opts.networkName())
	create.Stderr = stderr

	if err := create.Run(); err != nil {
		return fmt.Errorf("create network %s: %w", opts.networkName(), err)
	}

	return nil
}

func missingJobImages(ctx context.Context, opts Options) []string {
	var missing []string

	for _, image := range RequiredJobImages {
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

	if err = ensureNetwork(ctx, opts, stderr); err != nil {
		return err
	}

	if missing := missingJobImages(ctx, opts); len(missing) > 0 {
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
