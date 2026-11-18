//go:build podmanlive

package podman

import (
	"context"
	"testing"
	"time"
)

func TestLiveContract_ResourceLimitsMapToInspectHostConfig(t *testing.T) {
	socket := liveSocketPath(t)
	client := liveClient(t, socket)

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	suffix := liveSuffix()

	podResp, err := client.CreatePod(ctx, &PodCreateRequest{
		Name: "stageflow-contract-pod-rl-" + suffix,
		Labels: map[string]string{
			"managed_by": "stageflow-contract-test",
		},
	})
	if err != nil {
		t.Fatalf("CreatePod: %v", err)
	}
	defer func() { _ = client.RemovePod(context.Background(), podResp.ID, true) }()

	image := liveImage(t)

	const memoryMB int64 = 64

	ctrResp, err := client.CreateContainer(ctx, &ContainerCreateRequest{
		Name:    "stageflow-contract-ctr-rl-" + suffix,
		Image:   image,
		Pod:     podResp.ID,
		Command: []string{"sh", "-c", "echo ok"},
		ResourceLimits: &ResourceLimits{
			MemoryLimitMB: memoryMB,
			MemorySwapMB:  memoryMB,
			CPUQuota:      cpuQuota(0.5),
			CPUPeriod:     cpuPeriod,
		},
		Labels: map[string]string{
			"managed_by": "stageflow-contract-test",
		},
	})
	if err != nil {
		t.Fatalf("CreateContainer: %v", err)
	}
	defer removeContainer(ctrResp.ID)

	if startErr := client.StartContainer(ctx, ctrResp.ID); startErr != nil {
		t.Fatalf("StartContainer: %v", startErr)
	}

	if code := waitForExit(t, ctrResp.ID); code != 0 {
		t.Fatalf("expected container exit 0, got %d", code)
	}

	hostConfig := inspectContainer(ctx, t, client, ctrResp.ID).HostConfig
	if hostConfig.Memory != memoryMB*1024*1024 {
		t.Fatalf("expected HostConfig.Memory=%d, got %d", memoryMB*1024*1024, hostConfig.Memory)
	}

	if hostConfig.CpuQuota != cpuQuota(0.5) || hostConfig.CpuPeriod != cpuPeriod {
		t.Fatalf("expected HostConfig.CpuQuota=%d and CpuPeriod=%d, got %d and %d",
			cpuQuota(0.5), cpuPeriod, hostConfig.CpuQuota, hostConfig.CpuPeriod)
	}

	// Podman may omit MemorySwap; when present it must match the request.
	if hostConfig.MemorySwap != 0 && hostConfig.MemorySwap != memoryMB*1024*1024 {
		t.Fatalf("expected HostConfig.MemorySwap=%d (or 0), got %d", memoryMB*1024*1024, hostConfig.MemorySwap)
	}
}
