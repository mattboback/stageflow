//go:build podmanlive

package podman

import (
	"context"
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

	var podInspect map[string]any
	getLibpodJSON(ctx, t, client, "/pods/"+podResp.ID+"/json", &podInspect)

	infraAny, ok := podInspect["InfraConfig"]
	if !ok {
		t.Fatalf("pod inspect missing InfraConfig")
	}
	infra, ok := infraAny.(map[string]any)
	if !ok {
		t.Fatalf("pod inspect InfraConfig has unexpected type: %T", infraAny)
	}

	if hostAddAny, ok := infra["HostAdd"]; ok {
		hostAdd, ok := hostAddAny.([]any)
		if !ok {
			t.Fatalf("InfraConfig.HostAdd has unexpected type: %T", hostAddAny)
		}
		found := false
		for _, v := range hostAdd {
			if s, ok := v.(string); ok && s == hostEntry {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected InfraConfig.HostAdd to include %q, got %v", hostEntry, hostAddAny)
		}
	} else {
		t.Fatalf("pod inspect InfraConfig missing HostAdd")
	}

	if networksAny, ok := infra["Networks"]; ok {
		found := false
		switch v := networksAny.(type) {
		case map[string]any:
			_, found = v[networkName]
		case []any:
			for _, item := range v {
				if s, ok := item.(string); ok && s == networkName {
					found = true
					break
				}
				if m, ok := item.(map[string]any); ok {
					if name, ok := m["Name"].(string); ok && name == networkName {
						found = true
						break
					}
					if name, ok := m["name"].(string); ok && name == networkName {
						found = true
						break
					}
				}
			}
		default:
			t.Fatalf("InfraConfig.Networks has unexpected type: %T", networksAny)
		}

		if !found {
			t.Fatalf("expected pod to include network %q, got %v", networkName, networksAny)
		}
	} else {
		t.Fatalf("pod inspect InfraConfig missing Networks")
	}

	image := pickLiveImage(t, "docker.io/library/busybox:latest", "docker.io/library/alpine:3.20", "docker.io/library/alpine:latest")
	if image == "" {
		t.Skip("no suitable local image found for host/network contract test; pull busybox or alpine to run")
	}

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

	logs, err := client.GetContainerLogs(ctx, ctrResp.ID, true, true)
	if err != nil {
		t.Fatalf("GetContainerLogs: %v", err)
	}

	if !strings.Contains(logs, hostName) || !strings.Contains(logs, hostIP) {
		t.Fatalf("expected logs to contain host mapping %s -> %s; logs=%q", hostName, hostIP, logs)
	}

	var ctrInspect map[string]any
	getLibpodJSON(ctx, t, client, "/containers/"+ctrResp.ID+"/json", &ctrInspect)

	netSettingsAny, ok := ctrInspect["NetworkSettings"]
	if !ok {
		t.Fatalf("container inspect missing NetworkSettings")
	}
	netSettings, ok := netSettingsAny.(map[string]any)
	if !ok {
		t.Fatalf("container inspect NetworkSettings has unexpected type: %T", netSettingsAny)
	}

	networksAny, ok := netSettings["Networks"]
	if !ok {
		t.Fatalf("container inspect missing NetworkSettings.Networks")
	}
	networks, ok := networksAny.(map[string]any)
	if !ok {
		t.Fatalf("container inspect NetworkSettings.Networks has unexpected type: %T", networksAny)
	}
	if _, ok := networks[networkName]; !ok {
		t.Fatalf("expected container to be attached to network %q, got %v", networkName, networks)
	}
}
