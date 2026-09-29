package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"strings"
	"time"

	"github.com/mattboback/stageflow/libs/go/events"
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

		o.handleContainerFailure(ctx, containerID, jobID, component, waitResp.StatusCode)
	} else {
		slog.Debug("Container exited successfully", "component", component, "container_id", containerID)
	}
}

const scannerComponentPrefix = "scanner-"

// handleContainerFailure covers a container that exited before it could
// publish its own failure (killed, crashed, or NATS down). A scanner fails
// only itself, so the job keeps the other scanners' results. Anything else
// fails the job.
func (o *Orchestrator) handleContainerFailure(
	ctx context.Context,
	containerID, jobID, component string,
	exitCode int,
) {
	errorMsg := fmt.Sprintf("%s container exited with code %d", component, exitCode)

	scannerType, isScanner := strings.CutPrefix(component, scannerComponentPrefix)
	if !isScanner || scannerType == "" {
		o.failJobSafe(ctx, jobID, component, errorMsg)

		return
	}

	if o.containerOOMKilled(ctx, containerID) {
		errorMsg = o.outOfMemoryMessage(scannerType)
	}

	payload := &events.ScanFailedPayload{
		JobID:       jobID,
		ScannerType: scannerType,
		Error:       errorMsg,
	}

	if err := o.newService().RecordScannerFailure(ctx, payload); err != nil {
		slog.Error(
			"Failed to record scanner failure; failing the job",
			"component", component,
			"job_id", jobID,
			"error", err,
		)
		o.failJobSafe(ctx, jobID, component, errorMsg)
	}
}

func (o *Orchestrator) containerOOMKilled(ctx context.Context, containerID string) bool {
	info, err := o.podmanClient.InspectContainer(ctx, containerID)
	if err != nil {
		slog.Debug("Could not inspect exited container", "container_id", containerID, "error", err)

		return false
	}

	return info != nil && info.OOMKilled
}

func (o *Orchestrator) outOfMemoryMessage(scannerType string) string {
	if o.scannerRegistry != nil {
		if def, ok := o.scannerRegistry.Get(scannerType); ok && def.Requirements.MaxMemoryMB > 0 {
			return fmt.Sprintf("%s ran out of memory (limit %d MB)", scannerType, def.Requirements.MaxMemoryMB)
		}
	}

	return scannerType + " ran out of memory"
}
