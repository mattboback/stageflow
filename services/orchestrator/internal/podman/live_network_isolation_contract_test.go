//go:build podmanlive

package podman

import (
	"context"
	"net"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestLiveContract_JobNetworkCannotReachOtherNetworks proves the boundary the
// orchestrator relies on: a pod on an isolate=strict network reaches the
// internet and nothing on the host's other Podman networks. The plain-network
// control shows the probe would catch a missing isolation option.
func TestLiveContract_JobNetworkCannotReachOtherNetworks(t *testing.T) {
	socket := liveSocketPath(t)
	client := liveClient(t, socket)

	image := liveImage(t)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	suffix := liveSuffix()
	stackNetwork := "stageflow-contract-stack-" + suffix
	plainNetwork := "stageflow-contract-plain-" + suffix
	jobNetwork := "stageflow-contract-jobs-" + suffix

	createPodmanNetwork(t, stackNetwork)
	defer removePodmanNetwork(t, stackNetwork)
	createPodmanNetwork(t, plainNetwork)
	defer removePodmanNetwork(t, plainNetwork)
	createPodmanNetwork(t, jobNetwork, "--opt", "isolate=strict")
	defer removePodmanNetwork(t, jobNetwork)

	// The startup check must agree with real Podman about which network is safe.
	if err := VerifyJobNetwork(ctx, client, jobNetwork); err != nil {
		t.Fatalf("VerifyJobNetwork(isolated) error = %v, want nil", err)
	}

	if err := VerifyJobNetwork(ctx, client, plainNetwork); err == nil {
		t.Fatal("VerifyJobNetwork(plain) error = nil, want a refusal")
	}

	if err := VerifyJobNetwork(ctx, client, "stageflow-contract-absent-"+suffix); err == nil ||
		!strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("VerifyJobNetwork(missing) error = %v, want a does-not-exist error", err)
	}

	// A stand-in for the stack: a listener on another network, with an alias.
	serviceName := "stageflow-contract-svc-" + suffix
	runPodman(t, "run", "-d", "--name", serviceName, "--network", stackNetwork, "--network-alias", "stack-service",
		image, "nc", "-lk", "-p", "8080", "-e", "/bin/cat")

	defer func() { _ = exec.Command("podman", "rm", "-f", serviceName).Run() }()

	serviceIP := strings.TrimSpace(runPodman(t, "inspect", serviceName, "--format",
		`{{(index .NetworkSettings.Networks "`+stackNetwork+`").IPAddress}}`))
	if net.ParseIP(serviceIP) == nil {
		t.Fatalf("service address = %q, want an IP", serviceIP)
	}

	probe := func(network, name, command string) int {
		t.Helper()

		return runProbePod(ctx, t, client, image, network, name+"-"+suffix, command)
	}

	if code := probe(stackNetwork, "stageflow-contract-same", "nc -z -w 5 stack-service 8080"); code != 0 {
		t.Fatalf("same-network probe exited %d, want 0: the listener is not reachable at all", code)
	}

	if code := probe(plainNetwork, "stageflow-contract-plain", "nc -z -w 5 "+serviceIP+" 8080"); code != 0 {
		t.Fatalf(
			"plain-network probe exited %d, want 0: without isolation another bridge is reachable by address",
			code,
		)
	}

	if code := probe(jobNetwork, "stageflow-contract-ip", "nc -z -w 5 "+serviceIP+" 8080"); code == 0 {
		t.Fatal("isolated pod reached a container on another network by address")
	}

	if code := probe(jobNetwork, "stageflow-contract-name", "nc -z -w 5 stack-service 8080"); code == 0 {
		t.Fatal("isolated pod reached a container on another network by name")
	}

	// Scans need the internet. Only assert it when the host itself has it.
	dialer := net.Dialer{Timeout: 5 * time.Second}

	conn, dialErr := dialer.DialContext(ctx, "tcp", "1.1.1.1:443")
	if dialErr != nil {
		t.Logf("host has no outbound route to 1.1.1.1:443 (%v); skipping the internet assertion", dialErr)

		return
	}

	_ = conn.Close()

	if code := probe(jobNetwork, "stageflow-contract-internet", "nc -z -w 8 1.1.1.1 443"); code != 0 {
		t.Fatalf("isolated pod could not reach the internet (exit %d)", code)
	}
}

// runProbePod runs one shell command in a pod on the given network, the same
// way the orchestrator creates job pods, and returns its exit code.
func runProbePod(
	ctx context.Context,
	t *testing.T,
	client *Client,
	image, network, name, command string,
) int {
	t.Helper()

	labels := map[string]string{"managed_by": "stageflow-contract-test"}

	pod, err := client.CreatePod(ctx, &PodCreateRequest{
		Name:     name,
		Labels:   labels,
		Networks: map[string]PerNetworkOptions{network: {}},
		Netns:    PodNetns{Nsmode: PodNetnsModeBridge},
	})
	if err != nil {
		t.Fatalf("CreatePod(%s): %v", name, err)
	}

	defer func() { _ = client.RemovePod(context.Background(), pod.ID, true) }()

	container, err := client.CreateContainer(ctx, &ContainerCreateRequest{
		Name:    name + "-probe",
		Image:   image,
		Pod:     pod.ID,
		Command: []string{"sh", "-c", command},
		Labels:  labels,
	})
	if err != nil {
		t.Fatalf("CreateContainer(%s): %v", name, err)
	}

	if err := client.StartContainer(ctx, container.ID); err != nil {
		t.Fatalf("StartContainer(%s): %v", name, err)
	}

	return waitForExit(t, container.ID)
}
