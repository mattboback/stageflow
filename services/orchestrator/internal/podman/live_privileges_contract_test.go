//go:build podmanlive

package podman

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"
)

// A job container must have no capabilities, must not gain privileges through
// setuid binaries, must run under the process cap, and may write only to its
// mounts and /tmp, an anonymous volume that goes with the pod. The checks read
// the kernel's own view from inside the container.
func TestLiveContract_JobContainersRunWithoutPrivileges(t *testing.T) {
	socket := liveSocketPath(t)
	client := liveClient(t, socket)

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	suffix := liveSuffix()

	pod, err := client.CreatePod(ctx, &PodCreateRequest{
		Name:   "stageflow-contract-pod-priv-" + suffix,
		Labels: map[string]string{"managed_by": "stageflow-contract-test"},
	})
	if err != nil {
		t.Fatalf("CreatePod: %v", err)
	}

	podRemoved := false
	defer func() {
		if !podRemoved {
			_ = client.RemovePod(context.Background(), pod.ID, true)
		}
	}()

	container, err := client.CreateContainer(ctx, &ContainerCreateRequest{
		Name:  "stageflow-contract-ctr-priv-" + suffix,
		Image: liveImage(t),
		Pod:   pod.ID,
		Command: []string{"sh", "-c", strings.Join([]string{
			"grep -E '^(CapEff|CapBnd|NoNewPrivs):' /proc/self/status",
			"cat /sys/fs/cgroup/pids.max",
			"if touch /rootfs-write 2>/dev/null; then echo rootfs-writable; else echo rootfs-read-only; fi",
			"touch /tmp/probe && echo tmp-writable",
			"grep ' /tmp ' /proc/self/mounts",
		}, "; ")},
		Labels: map[string]string{"managed_by": "stageflow-contract-test"},
	})
	if err != nil {
		t.Fatalf("CreateContainer: %v", err)
	}

	if startErr := client.StartContainer(ctx, container.ID); startErr != nil {
		t.Fatalf("StartContainer: %v", startErr)
	}

	if code := waitForExit(t, container.ID); code != 0 {
		t.Fatalf("probe exited %d: %s", code, containerLogs(t, container.ID))
	}

	logs := containerLogs(t, container.ID)
	for _, want := range []string{
		"CapEff:\t0000000000000000",
		"CapBnd:\t0000000000000000",
		"NoNewPrivs:\t1",
		strconv.Itoa(jobContainerPidsLimit),
		"rootfs-read-only",
		"tmp-writable",
		"noexec",
	} {
		if !strings.Contains(logs, want) {
			t.Errorf("container status lacks %q:\n%s", want, logs)
		}
	}

	if strings.Contains(logs, "tmpfs /tmp") {
		t.Errorf("/tmp is a tmpfs, which counts against the memory limit:\n%s", logs)
	}

	// The orchestrator removes only pods. That must take the /tmp volume too.
	volume := scratchVolumeName(ctx, t, client, container.ID)
	if removeErr := client.RemovePod(ctx, pod.ID, true); removeErr != nil {
		t.Fatalf("RemovePod: %v", removeErr)
	}

	podRemoved = true

	if volumeExists(volume) {
		t.Errorf("removing the pod left its /tmp volume %s behind", volume)
		removeVolume(volume)
	}
}
