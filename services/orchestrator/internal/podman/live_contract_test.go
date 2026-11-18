//go:build podmanlive

package podman

import (
	"context"
	"testing"
	"time"
)

func TestLiveContract_MountReadOnlyEncodedViaOptions(t *testing.T) {
	socket := liveSocketPath(t)
	client := liveClient(t, socket)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create isolated resources and clean them up.
	suffix := liveSuffix()
	volName := "stageflow-contract-" + suffix
	podName := "stageflow-contract-pod-" + suffix

	if err := client.CreateVolume(ctx, volName); err != nil {
		t.Fatalf("CreateVolume: %v", err)
	}
	defer removeVolume(volName)

	vol, err := client.InspectVolume(ctx, volName)
	if err != nil {
		t.Fatalf("InspectVolume: %v", err)
	}

	podResp, err := client.CreatePod(ctx, &PodCreateRequest{
		Name: podName,
		Labels: map[string]string{
			"managed_by": "stageflow-contract-test",
		},
	})
	if err != nil {
		t.Fatalf("CreatePod: %v", err)
	}
	defer func() { _ = client.RemovePod(context.Background(), podResp.ID, true) }()

	image := liveImage(t)

	ctrResp, err := client.CreateContainer(ctx, &ContainerCreateRequest{
		Name:  "stageflow-contract-ctr-" + suffix,
		Image: image,
		Pod:   podResp.ID,
		Mounts: []VolumeMount{
			{
				Type:        "bind",
				Source:      vol.Mountpoint,
				Destination: "/workspace",
				ReadOnly:    true,
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateContainer: %v", err)
	}
	defer removeContainer(ctrResp.ID)

	for _, mount := range inspectContainer(ctx, t, client, ctrResp.ID).Mounts {
		if mount.Destination != "/workspace" {
			continue
		}

		if mount.RW == nil || *mount.RW {
			t.Fatalf("expected /workspace mount to be read-only, got RW=%v", mount.RW)
		}

		return
	}

	t.Fatalf("did not find /workspace mount in inspect Mounts")
}
