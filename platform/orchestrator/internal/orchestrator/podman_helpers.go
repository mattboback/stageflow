package orchestrator

import (
	"context"
	"fmt"
	"log/slog"

	podman "github.com/mattboback/stageflow/platform/orchestrator/internal/adapters/runtime"
)

func (o *Orchestrator) ensureVolume(ctx context.Context, name string) (*podman.VolumeInfo, error) {
	if info, err := o.podmanClient.InspectVolume(ctx, name); err == nil {
		return info, nil
	}

	if err := o.podmanClient.CreateVolume(ctx, name); err != nil {
		return nil, err
	}

	return o.podmanClient.InspectVolume(ctx, name)
}

func (o *Orchestrator) spawnMonitorContainer(ctx context.Context, containerID, jobID, component string) {
	o.monitorWG.Add(1)

	go func() {
		defer o.monitorWG.Done()

		o.monitorContainer(ctx, containerID, jobID, component)
	}()
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

	//nolint:nestif // Error handling with log extraction requires multiple conditional paths
	if waitResp.StatusCode != 0 {
		logs, logErr := o.podmanClient.GetContainerLogs(ctx, containerID, true, true)
		logTail := ""

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
			logTail = truncateLogs(logs, 500)
			slog.Error(
				"Container exited with error",
				"component",
				component,
				"container_id",
				containerID,
				"exit_code",
				waitResp.StatusCode,
				"logs_tail",
				logTail,
			)
		}

		if logTail != "" {
			o.recordInternalEvent(ctx, jobID, "orchestrator.container.exit_error", map[string]any{
				"component":    component,
				"container_id": containerID,
				"exit_code":    waitResp.StatusCode,
				"logs_tail":    logTail,
			})
		} else {
			o.recordInternalEvent(ctx, jobID, "orchestrator.container.exit_error", map[string]any{
				"component":    component,
				"container_id": containerID,
				"exit_code":    waitResp.StatusCode,
			})
		}

		// Fail the job to ensure we don't hang if the worker couldn't publish failure (e.g. NATS down)
		errorMsg := fmt.Sprintf("%s container exited with code %d", component, waitResp.StatusCode)
		if logTail != "" {
			o.failJobSafeWithDetails(ctx, jobID, component, errorMsg, logTail)
		} else {
			o.failJobSafe(ctx, jobID, component, errorMsg)
		}
	} else {
		slog.Debug("Container exited successfully", "component", component, "container_id", containerID)
	}
}

func truncateLogs(logs string, n int) string {
	if len(logs) <= n {
		return logs
	}

	return "..." + logs[len(logs)-n:]
}
