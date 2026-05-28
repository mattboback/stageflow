// Authenticated-scan capture surfaces.
//
// `stageflow auth capture <url>` launches a non-headless Chromium via
// `npx playwright open --save-storage=<path> <url>`, lets the developer log in
// interactively, and writes the resulting Playwright storage-state JSON to a
// developer-specified path with file mode 0600.
//
// This is the only flow that ever sees a real password. It runs locally on the
// developer's machine and never reaches the platform-api or the orchestrator;
// only the captured storage-state file does (via `stageflow scan --auth-state`).
//
// See docs/architecture/system.md#authenticated-scanning for the trust boundaries.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// authCaptureRunner is the indirection that lets tests replace the actual
// `npx playwright open ...` invocation with a stub. The default implementation
// shells out via os/exec.
type authCaptureRunner func(cmd *cobra.Command, url, output string, extraArgs []string) error

func newAuthCmd(_ *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication artifacts for authenticated scans",
		Long: "Subcommands that capture browser session state for use with `stageflow scan --auth-state`.\n\n" +
			"See docs/architecture/system.md#authenticated-scanning for the design and trust boundaries.",
	}

	cmd.AddCommand(newAuthCaptureCmd(defaultAuthCaptureRunner))

	return cmd
}

func newAuthCaptureCmd(runner authCaptureRunner) *cobra.Command {
	var (
		output    string
		extraArgs []string
	)

	cmd := &cobra.Command{
		Use:   "capture <url>",
		Short: "Launch a non-headless Chromium and write Playwright storage state on exit",
		Long: "Opens a non-headless Chromium pointed at <url> using `npx playwright open --save-storage`. " +
			"Log in by hand, then close the browser; the resulting storage-state JSON is written to --output " +
			"with file mode 0600.\n\n" +
			"Pass that file to `stageflow scan --auth-state <path>` to attach the captured session to a job.\n\n" +
			"Requires Node.js and Playwright to be installed locally (npm install -g playwright, or run from " +
			"a project that already has playwright as a dependency). The CLI never sees the password; only the " +
			"resulting cookies + localStorage do.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := strings.TrimSpace(args[0])
			if url == "" {
				return exitCodeError{Code: 2, Err: errors.New("login URL must not be empty")}
			}

			if strings.TrimSpace(output) == "" {
				return exitCodeError{Code: 2, Err: errors.New("--output is required")}
			}

			absOut, err := filepath.Abs(output)
			if err != nil {
				return exitCodeError{Code: 2, Err: fmt.Errorf("resolve --output path: %w", err)}
			}

			if dir := filepath.Dir(absOut); dir != "" {
				// 0o700 so a captured session never lives in a world-readable
				// directory we created ourselves. Existing parent dirs are left
				// alone.
				if mkErr := os.MkdirAll(dir, 0o700); mkErr != nil {
					return exitCodeError{Code: 2, Err: fmt.Errorf("create output directory: %w", mkErr)}
				}
			}

			if runErr := runner(cmd, url, absOut, extraArgs); runErr != nil {
				return runErr
			}

			info, statErr := os.Stat(absOut)
			if statErr != nil {
				return exitCodeError{
					Code: 2,
					Err: fmt.Errorf(
						"playwright open exited but did not write %s: %w",
						absOut,
						statErr,
					),
				}
			}

			if info.Size() == 0 {
				return exitCodeError{
					Code: 2,
					Err: fmt.Errorf(
						"playwright open wrote an empty storage-state file at %s; "+
							"did the browser close before login completed?",
						absOut,
					),
				}
			}

			if chmodErr := os.Chmod(absOut, 0o600); chmodErr != nil {
				return exitCodeError{
					Code: 2,
					Err:  fmt.Errorf("set 0600 permissions on %s: %w", absOut, chmodErr),
				}
			}

			fmt.Fprintf(
				cmd.ErrOrStderr(),
				"Captured Playwright storage state to %s (mode 0600, %d bytes).\n"+
					"Use it with: stageflow scan ... --auth-state %s\n",
				absOut,
				info.Size(),
				absOut,
			)

			return nil
		},
	}

	cmd.Flags().StringVarP(
		&output,
		"output",
		"o",
		"",
		"Path to write the storage-state JSON (required). Created with mode 0600.",
	)
	cmd.Flags().StringSliceVar(
		&extraArgs,
		"playwright-arg",
		nil,
		"Extra argument to forward to `playwright open` (repeatable, e.g. --playwright-arg=--browser=chromium)",
	)

	cobra.CheckErr(cmd.MarkFlagRequired("output"))

	return cmd
}

// defaultAuthCaptureRunner shells out to `npx playwright open --save-storage=<path> <url>`.
// The user-facing process inherits stdin/stdout/stderr so the developer can interact with
// the browser as if they had run npx directly.
func defaultAuthCaptureRunner(cmd *cobra.Command, url, output string, extraArgs []string) error {
	if _, err := exec.LookPath("npx"); err != nil {
		return exitCodeError{
			Code: 2,
			Err: errors.New(
				"npx not found on PATH. Install Node.js (>= 22) and Playwright " +
					"(npm install -g playwright, or run from a project that already depends on playwright) " +
					"to use `stageflow auth capture`",
			),
		}
	}

	args := []string{"--yes", "playwright", "open", "--save-storage=" + output}
	args = append(args, extraArgs...)
	args = append(args, url)

	fmt.Fprintf(
		cmd.ErrOrStderr(),
		"Launching Chromium for interactive login...\n"+
			"Log in by hand, then close the browser window. Storage state will be written to %s.\n",
		output,
	)

	sub := exec.CommandContext(cmd.Context(), "npx", args...)
	sub.Stdin = os.Stdin
	sub.Stdout = cmd.OutOrStdout()
	sub.Stderr = cmd.ErrOrStderr()

	if err := sub.Run(); err != nil {
		return exitCodeError{Code: 2, Err: fmt.Errorf("`npx playwright open` exited with error: %w", err)}
	}

	return nil
}
