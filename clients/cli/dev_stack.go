package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type runningProcess struct {
	cmd    *exec.Cmd
	waitCh chan error
}

func runCommandSteps(ctx context.Context, projectRoot string, steps [][]string, stderr io.Writer) error {
	for i, step := range steps {
		if len(step) == 0 {
			return fmt.Errorf("dev.up[%d] is empty", i)
		}

		fmt.Fprintf(stderr, "[dev] running: %s\n", strings.Join(step, " "))

		//nolint:gosec // dev scan executes repo-provided commands; config is trusted input.
		cmd := exec.CommandContext(ctx, step[0], step[1:]...)
		cmd.Dir = projectRoot
		cmd.Env = os.Environ()
		cmd.Stdout = stderr
		cmd.Stderr = stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("dev command failed (%s): %w", strings.Join(step, " "), err)
		}
	}

	return nil
}

func startDevServer(
	ctx context.Context,
	projectRoot string,
	cfg projectDevStartCfg,
	stderr io.Writer,
) (*runningProcess, error) {
	if len(cfg.Cmd) == 0 {
		return nil, errors.New("dev.start.cmd is empty")
	}

	dir := projectRoot
	if strings.TrimSpace(cfg.Cwd) != "" {
		dir = cfg.Cwd
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(projectRoot, dir)
		}
	}

	//nolint:gosec // dev scan executes repo-provided commands; config is trusted input.
	cmd := exec.CommandContext(ctx, cfg.Cmd[0], cfg.Cmd[1:]...)
	cmd.Dir = dir
	cmd.Env = mergeEnv(os.Environ(), cfg.Env)
	cmd.Stdout = stderr
	cmd.Stderr = stderr

	setProcessGroup(cmd)

	fmt.Fprintf(stderr, "[dev] starting: %s\n", strings.Join(cfg.Cmd, " "))

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start dev server: %w", err)
	}

	waitCh := make(chan error, 1)

	go func() {
		waitCh <- cmd.Wait()
	}()

	return &runningProcess{
		cmd:    cmd,
		waitCh: waitCh,
	}, nil
}

//nolint:gocognit,gocyclo // Readiness checks involve timeouts, polling, and process lifecycle handling.
func waitForReady(ctx context.Context, proc *runningProcess, cfg projectDevReadyCfg, stderr io.Writer) error {
	readyURL := strings.TrimSpace(cfg.URL)
	if readyURL == "" {
		return errors.New("dev.ready.url is empty")
	}

	timeout := 60 * time.Second

	if d, ok, err := configDuration(cfg.Timeout); err != nil {
		return fmt.Errorf("dev.ready.timeout: %w", err)
	} else if ok {
		timeout = d
	}

	interval := 500 * time.Millisecond

	if d, ok, err := configDuration(cfg.Interval); err != nil {
		return fmt.Errorf("dev.ready.interval: %w", err)
	} else if ok {
		interval = d
	}

	readyCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	checkOnce := func() (bool, error) {
		req, err := http.NewRequestWithContext(readyCtx, http.MethodGet, readyURL, http.NoBody)
		if err != nil {
			return false, fmt.Errorf("invalid ready URL %q: %w", readyURL, err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return false, nil
		}

		defer func() { _ = resp.Body.Close() }()

		return resp.StatusCode >= 200 && resp.StatusCode < 400, nil
	}

	fmt.Fprintf(stderr, "[dev] waiting for readiness: %s\n", readyURL)

	ready, err := checkOnce()
	if err != nil {
		return err
	}

	if ready {
		return nil
	}

	for {
		select {
		case <-readyCtx.Done():
			return fmt.Errorf("dev server not ready within %v", timeout)
		case procErr := <-proc.waitCh:
			return fmt.Errorf("dev server exited before ready: %w", procErr)
		case <-ticker.C:
			ready, err = checkOnce()
			if err != nil {
				return err
			}

			if ready {
				return nil
			}
		}
	}
}

func stopDevServer(proc *runningProcess, cfg projectDevStopCfg, stderr io.Writer) {
	if proc == nil || proc.cmd == nil || proc.cmd.Process == nil {
		return
	}

	sig := syscall.SIGINT

	if s := strings.TrimSpace(cfg.Signal); s != "" {
		parsed, err := parseSignal(s)
		if err != nil {
			fmt.Fprintf(stderr, "[dev] invalid stop.signal %q: %v (defaulting to SIGINT)\n", s, err)
		} else {
			sig = parsed
		}
	}

	timeout := 10 * time.Second

	if d, ok, err := configDuration(cfg.Timeout); err != nil {
		fmt.Fprintf(stderr, "[dev] invalid stop.timeout %q: %v (defaulting to %v)\n", cfg.Timeout, err, timeout)
	} else if ok {
		timeout = d
	}

	fmt.Fprintf(stderr, "[dev] stopping dev server (%v)...\n", sig)
	sendSignal(proc.cmd.Process, sig)

	select {
	case <-proc.waitCh:
		return
	case <-time.After(timeout):
		fmt.Fprintln(stderr, "[dev] dev server did not stop in time; killing")
		sendSignal(proc.cmd.Process, syscall.SIGKILL)
		<-proc.waitCh
	}
}

func parseSignal(raw string) (syscall.Signal, error) {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "SIGINT", "INT":
		return syscall.SIGINT, nil
	case "SIGTERM", "TERM":
		return syscall.SIGTERM, nil
	case "SIGKILL", "KILL":
		return syscall.SIGKILL, nil
	default:
		return 0, fmt.Errorf("unsupported signal %q (use SIGINT, SIGTERM, or SIGKILL)", raw)
	}
}

func sendSignal(p *os.Process, sig syscall.Signal) {
	if p == nil {
		return
	}

	// Best-effort: signal the whole process group so child processes also stop.
	if signalProcessGroup(p, sig) {
		return
	}

	if err := p.Signal(sig); err != nil {
		return
	}
}

func mergeEnv(base []string, overlay map[string]string) []string {
	if len(overlay) == 0 {
		return base
	}

	seen := map[string]bool{}
	for k := range overlay {
		seen[k] = true
	}

	out := make([]string, 0, len(base)+len(overlay))
	for _, item := range base {
		key, _, ok := strings.Cut(item, "=")
		if ok && seen[key] {
			continue
		}

		out = append(out, item)
	}

	for k, v := range overlay {
		out = append(out, k+"="+v)
	}

	return out
}
