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

	image := pickLiveImage(t, "docker.io/library/busybox:latest", "docker.io/library/alpine:3.20", "docker.io/library/alpine:latest")
	if image == "" {
		t.Skip("no suitable local image found for resource limits contract test; pull busybox or alpine to run")
	}

	const memoryMB int64 = 64
	expectedBytes := float64(memoryMB * 1024 * 1024)

	ctrResp, err := client.CreateContainer(ctx, &ContainerCreateRequest{
		Name:    "stageflow-contract-ctr-rl-" + suffix,
		Image:   image,
		Pod:     podResp.ID,
		Command: []string{"sh", "-c", "echo ok"},
		ResourceLimits: &ResourceLimits{
			MemoryLimitMB: memoryMB,
			MemorySwapMB:  memoryMB,
		},
		Labels: map[string]string{
			"managed_by": "stageflow-contract-test",
		},
	})
	if err != nil {
		t.Fatalf("CreateContainer: %v", err)
	}
	defer func() { _ = client.RemoveContainer(context.Background(), ctrResp.ID, true) }()

	if err := client.StartContainer(ctx, ctrResp.ID); err != nil {
		t.Fatalf("StartContainer: %v", err)
	}

	waitResp, err := client.WaitContainer(ctx, ctrResp.ID)
	if err != nil {
		t.Fatalf("WaitContainer: %v", err)
	}
	if waitResp.StatusCode != 0 {
		t.Fatalf("expected container exit 0, got %d", waitResp.StatusCode)
	}

	var ctrInspect map[string]any
	getLibpodJSON(ctx, t, client, "/containers/"+ctrResp.ID+"/json", &ctrInspect)

	hostCfgAny, ok := ctrInspect["HostConfig"]
	if !ok {
		t.Fatalf("container inspect missing HostConfig")
	}
	hostCfg, ok := hostCfgAny.(map[string]any)
	if !ok {
		t.Fatalf("container inspect HostConfig has unexpected type: %T", hostCfgAny)
	}

	memAny, ok := hostCfg["Memory"]
	if !ok {
		t.Fatalf("container inspect HostConfig missing Memory")
	}
	mem, ok := memAny.(float64)
	if !ok {
		t.Fatalf("container inspect HostConfig.Memory has unexpected type: %T", memAny)
	}
	if mem != expectedBytes {
		t.Fatalf("expected HostConfig.Memory=%v, got %v", expectedBytes, mem)
	}

	// MemorySwap isn't always reported consistently, but when present it should match our request.
	if swapAny, ok := hostCfg["MemorySwap"]; ok {
		if swap, ok := swapAny.(float64); ok && swap != 0 && swap != expectedBytes {
			t.Fatalf("expected HostConfig.MemorySwap=%v (or 0), got %v", expectedBytes, swap)
		}
	}
}
