//go:build podmanlive

package podman

import (
	"context"
	"encoding/json"
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
	defer func() { _ = client.RemoveVolume(context.Background(), volName, true) }()

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

	image := pickLiveImage(t, "localhost/stageflow/extractor:latest", "stageflow/extractor:latest", "docker.io/library/alpine:3.20", "docker.io/library/alpine:latest")
	if image == "" {
		t.Skip("no suitable local image found for live contract test; build StageFlow images or pull alpine")
	}

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
	defer func() { _ = client.RemoveContainer(context.Background(), ctrResp.ID, true) }()

	// Inspect via the raw API so we can validate the mount record.
	resp, err := client.doLibpodRequest(ctx, "GET", "/containers/"+ctrResp.ID+"/json", nil)
	if err != nil {
		t.Fatalf("inspect request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var inspect map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&inspect); err != nil {
		t.Fatalf("decode inspect: %v", err)
	}

	// Podman inspect includes Mounts as an array of objects; ensure our bind mount is read-only.
	mountsAny, ok := inspect["Mounts"]
	if !ok {
		t.Fatalf("inspect missing Mounts field")
	}
	mounts, ok := mountsAny.([]any)
	if !ok || len(mounts) == 0 {
		t.Fatalf("inspect Mounts has unexpected type/value: %T", mountsAny)
	}

	found := false
	for _, m := range mounts {
		mm, ok := m.(map[string]any)
		if !ok {
			continue
		}
		if mm["Destination"] != "/workspace" {
			continue
		}
		found = true

		// Most Podman inspect outputs include RW (bool) on mounts.
		if rw, ok := mm["RW"].(bool); ok {
			if rw {
				t.Fatalf("expected /workspace mount to be read-only (RW=false), got RW=true")
			}
			return
		}

		// Fallback: check for "ro" in mount Options if RW isn't present.
		if opts, ok := mm["Options"].([]any); ok {
			for _, o := range opts {
				if s, ok := o.(string); ok && s == "ro" {
					return
				}
			}
			t.Fatalf("expected /workspace mount options to include ro, got %v", opts)
		}

		t.Fatalf("found /workspace mount but could not validate read-only fields: %v", mm)
	}

	if !found {
		t.Fatalf("did not find /workspace mount in inspect Mounts")
	}
}
