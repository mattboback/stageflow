//go:build podmanlive

package podman

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestLiveContract_PodNetworkAndHostAddPropagateToContainer(t *testing.T) {
	socket := liveSocketPath(t)
	client := liveClient(t, socket)

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	suffix := liveSuffix()
	networkName := "stageflow-contract-net-" + suffix

	createPodmanNetwork(t, networkName)
	defer removePodmanNetwork(t, networkName)

	hostName := "stageflow-contract-host-" + suffix
	hostIP := "169.254.1.2"
	hostEntry := hostName + ":" + hostIP

	podResp, err := client.CreatePod(ctx, &PodCreateRequest{
		Name: "stageflow-contract-pod-" + suffix,
		Labels: map[string]string{
			"managed_by": "stageflow-contract-test",
		},
		Networks: map[string]PerNetworkOptions{
			networkName: {},
		},
		Netns:   PodNetns{Nsmode: "bridge"},
		HostAdd: []string{hostEntry},
	})
	if err != nil {
		t.Fatalf("CreatePod: %v", err)
	}
	defer func() { _ = client.RemovePod(context.Background(), podResp.ID, true) }()

	infra := inspectPod(ctx, t, client, podResp.ID).InfraConfig
	if !slices.Contains(infra.HostAdd, hostEntry) {
		t.Fatalf("expected InfraConfig.HostAdd to include %q, got %v", hostEntry, infra.HostAdd)
	}

	if !slices.Contains(infra.Networks, networkName) {
		t.Fatalf("expected pod to include network %q, got %v", networkName, infra.Networks)
	}

	image := liveImage(t)

	ctrResp, err := client.CreateContainer(ctx, &ContainerCreateRequest{
		Name:    "stageflow-contract-ctr-" + suffix,
		Image:   image,
		Pod:     podResp.ID,
		Command: []string{"sh", "-c", "cat /etc/hosts"},
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

	logs := containerLogs(t, ctrResp.ID)

	if !strings.Contains(logs, hostName) || !strings.Contains(logs, hostIP) {
		t.Fatalf("expected logs to contain host mapping %s -> %s; logs=%q", hostName, hostIP, logs)
	}

	networks := inspectContainer(ctx, t, client, ctrResp.ID).NetworkSettings.Networks
	if _, attached := networks[networkName]; !attached {
		t.Fatalf("expected container to be attached to network %q, got %v", networkName, networks)
	}
}
