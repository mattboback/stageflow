package projectmode

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

//nolint:gocognit,gocyclo // Readiness checks involve timeouts, polling, and process lifecycle handling.
func WaitForReady(
	ctx context.Context,
	httpClient *http.Client,
	proc *RunningProcess,
	cfg DevReadyConfig,
	stderr io.Writer,
) error {
	readyURL := strings.TrimSpace(cfg.URL)
	if readyURL == "" {
		return errors.New("dev.ready.url is empty")
	}

	timeout := 60 * time.Second

	if d, ok, err := ConfigDuration(cfg.Timeout); err != nil {
		return fmt.Errorf("dev.ready.timeout: %w", err)
	} else if ok {
		timeout = d
	}

	interval := 500 * time.Millisecond

	if d, ok, err := ConfigDuration(cfg.Interval); err != nil {
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

		resp, err := httpClient.Do(req)
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
