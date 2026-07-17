package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"strings"
	"time"
)

func (o *Orchestrator) spawnMonitorContainer(ctx context.Context, containerID, jobID, component string) {
	o.monitorWG.Add(1)

	go func() {
		defer o.monitorWG.Done()

		// Recover so a panic in the background monitor cannot crash the whole
		// orchestrator process; the failure is logged with the job context.
		defer func() {
			if panicValue := recover(); panicValue != nil {
				slog.Error(
					"Recovered panic in container monitor goroutine",
					"component", component,
					"container_id", containerID,
					"job_id", jobID,
					"panic", panicValue,
					"stack", string(debug.Stack()),
				)
			}
		}()

		monitorCtx := ctx

		cancel := func() {}
		if timeout := o.monitorTimeout(component); timeout > 0 {
			monitorCtx, cancel = context.WithTimeout(ctx, timeout)
		}
		defer cancel()

		o.monitorContainer(monitorCtx, containerID, jobID, component)
	}()
}

func (o *Orchestrator) monitorTimeout(component string) time.Duration {
	if o == nil {
		return 0
	}

	switch {
	case strings.HasPrefix(component, "scanner"):
		return o.scanTimeout + time.Minute
	case component == "extraction":
		return o.extractionTimeout + time.Minute
	default:
		return 0
	}
}

func (o *Orchestrator) monitorContainer(ctx context.Context, containerID, jobID, component string) {
	slog.Debug("Monitoring container", "component", component, "container_id", containerID, "job_id", jobID)

	waitResp, err := o.podmanClient.WaitContainer(ctx, containerID)
	if err != nil {
		slog.Error("Error waiting for container", "component", component, "container_id", containerID, "error", err)

		return
	}

	o.recordInternalEvent(ctx, jobID, "orchestrator.container.exited", map[string]any{
		"component":    component,
		"container_id": containerID,
		"exit_code":    waitResp.StatusCode,
	})

	if waitResp.StatusCode != 0 {
		logs, logErr := o.podmanClient.GetContainerLogs(ctx, containerID, true, true)
		logsAvailable := logErr == nil && logs != ""

		if logErr != nil {
			slog.Error(
				"Container exited with error",
				"component",
				component,
				"container_id",
				containerID,
				"exit_code",
				waitResp.StatusCode,
				"log_error",
				logErr,
			)
		} else {
			slog.Error(
				"Container exited with error",
				"component",
				component,
				"container_id",
				containerID,
				"exit_code",
				waitResp.StatusCode,
				"logs_available",
				logsAvailable,
				"logs_bytes",
				len(logs),
			)
		}

		o.recordInternalEvent(ctx, jobID, "orchestrator.container.exit_error", map[string]any{
			"component":      component,
			"container_id":   containerID,
			"exit_code":      waitResp.StatusCode,
			"logs_available": logsAvailable,
		})

		// Fail the job to ensure we don't hang if the worker couldn't publish failure (e.g. NATS down)
		errorMsg := fmt.Sprintf("%s container exited with code %d", component, waitResp.StatusCode)
		o.failJobSafe(ctx, jobID, component, errorMsg)
	} else {
		slog.Debug("Container exited successfully", "component", component, "container_id", containerID)
	}
}

func truncateLogs(logs string, n int) string {
	cleaned := sanitizeLogText(logs)

	if n <= 0 {
		return ""
	}

	runes := []rune(cleaned)
	if len(runes) <= n {
		return cleaned
	}

	return "..." + string(runes[len(runes)-n:])
}

func sanitizeLogText(logs string) string {
	valid := strings.ToValidUTF8(logs, "")

	return strings.Map(func(r rune) rune {
		switch r {
		case '\n', '\r', '\t':
			return r
		case 0:
			return -1
		}

		if r < 0x20 || r == 0x7f {
			return -1
		}

		return r
	}, valid)
}
