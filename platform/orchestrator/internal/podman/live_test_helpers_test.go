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

	if _, err := os.Stat(socket); err != nil {
		t.Skipf("podman socket not available at %s: %v", socket, err)
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

func pickLiveImage(t *testing.T, candidates ...string) string {
	t.Helper()
	for _, c := range candidates {
		if c != "" && podmanImageExists(c) {
			return c
		}
	}
	return ""
}

func createPodmanNetwork(t *testing.T, name string) {
	t.Helper()
	cmd := exec.Command("podman", "network", "create", name)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("podman network create %s: %v (%s)", name, err, string(out))
	}
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
