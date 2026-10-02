//go:build podmanlive

package podman

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"testing"
	"time"
)

func liveSocketPath(t *testing.T) string {
	t.Helper()

	socket := os.Getenv("PODMAN_SOCKET")
	if socket == "" {
		socket = "/run/user/1000/podman/podman.sock"
	}

	// The podmanlive tag asks for these tests by name. A skip would read as a
	// pass in a gate that exists to prove the runtime contract.
	if _, err := os.Stat(socket); err != nil {
		t.Fatalf("podman socket not available at %s: %v (set PODMAN_SOCKET)", socket, err)
	}

	return socket
}

func liveClient(t *testing.T, socket string) *Client {
	t.Helper()

	client, err := NewClient(&Config{SocketPath: socket})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client
}

func liveSuffix() string {
	return time.Now().UTC().Format("20060102t150405.000000000")
}

func podmanImageExists(image string) bool {
	cmd := exec.Command("podman", "image", "exists", image)
	return cmd.Run() == nil
}

// liveImage returns a small local image with a shell and busybox tools. The
// tests never pull: a missing image fails with the command that fixes it.
func liveImage(t *testing.T) string {
	t.Helper()
	candidates := []string{
		os.Getenv("STAGEFLOW_LIVE_IMAGE"),
		"docker.io/library/busybox:latest",
		"docker.io/library/alpine:3.22",
		"docker.io/library/alpine:3.20",
		"docker.io/library/alpine:latest",
	}
	for _, c := range candidates {
		if c != "" && podmanImageExists(c) {
			return c
		}
	}
	t.Fatalf("no local image for the live contract tests; run `podman pull docker.io/library/alpine:3.22` " +
		"or set STAGEFLOW_LIVE_IMAGE")
	return ""
}

func createPodmanNetwork(t *testing.T, name string, options ...string) {
	t.Helper()
	runPodman(t, append(append([]string{"network", "create"}, options...), name)...)
}

func runPodman(t *testing.T, args ...string) string {
	t.Helper()
	out, err := exec.Command("podman", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("podman %v: %v (%s)", args, err, string(out))
	}
	return string(out)
}

func removePodmanNetwork(t *testing.T, name string) {
	t.Helper()
	cmd := exec.Command("podman", "network", "rm", name)
	_ = cmd.Run()
}

func decodeJSONBody(t *testing.T, respBody any, out any) {
	t.Helper()
	data, err := json.Marshal(respBody)
	if err != nil {
		t.Fatalf("marshal JSON: %v", err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		t.Fatalf("unmarshal JSON: %v", err)
	}
}

func getLibpodJSON(ctx context.Context, t *testing.T, client *Client, suffix string, out any) {
	t.Helper()
	resp, err := client.doLibpodRequest(ctx, "GET", suffix, nil)
	if err != nil {
		t.Fatalf("GET %s: %v", suffix, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if err := parseResponse(resp, out); err != nil {
		t.Fatalf("parse %s: %v", suffix, err)
	}
}
