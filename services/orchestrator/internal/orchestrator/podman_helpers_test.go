package orchestrator

import (
	"context"
	"errors"
	"testing"
	"time"

	podman "github.com/mattboback/stageflow/services/orchestrator/internal/adapters/runtime"
)

func TestSpawnMonitorContainerBoundsScannerWait(t *testing.T) {
	t.Parallel()

	deadlineSeen := make(chan time.Duration, 1)
	client := &mockPodmanClient{
		waitContainerFunc: func(ctx context.Context, _ string) (*podman.ContainerWaitResponse, error) {
			deadline, ok := ctx.Deadline()
			if !ok {
				deadlineSeen <- 0
				return nil, errors.New("missing deadline")
			}

			deadlineSeen <- time.Until(deadline)

			return &podman.ContainerWaitResponse{StatusCode: 0}, nil
		},
	}

	orch := NewOrchestrator(&Config{
		PodmanClient: client,
		ScanTimeout:  25 * time.Millisecond,
	})

	orch.spawnMonitorContainer(context.Background(), "container-1", "job-1", "scanner-axe")
	orch.WaitForMonitors()

	select {
	case remaining := <-deadlineSeen:
		if remaining <= 0 {
			t.Fatalf("expected positive scanner monitor deadline, got %v", remaining)
		}
	case <-time.After(time.Second):
		t.Fatal("wait container was not called")
	}
}

func TestSpawnMonitorContainerRecoversPanic(t *testing.T) {
	t.Parallel()

	client := &mockPodmanClient{
		waitContainerFunc: func(context.Context, string) (*podman.ContainerWaitResponse, error) {
			panic("boom in monitor")
		},
	}

	orch := NewOrchestrator(&Config{PodmanClient: client})

	// If the panic were not recovered, the goroutine would crash the test
	// process. Reaching the assertion after WaitForMonitors proves recovery.
	orch.spawnMonitorContainer(context.Background(), "container-1", "job-1", "scanner-axe")
	orch.WaitForMonitors()
}
