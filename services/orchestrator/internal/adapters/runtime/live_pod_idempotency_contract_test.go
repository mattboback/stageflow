//go:build podmanlive

package podman

import (
	"context"
	"os/exec"
	"testing"

	"github.com/mattboback/stageflow/libs/go/models"
)

// Real Podman, real 409. The unit tests inject the conflict; this proves the
// recovery works against the daemon's actual behaviour and error shape, which is
// the part a fake cannot vouch for.
func TestLiveContractCreateJobPodIsIdempotent(t *testing.T) {
	socket := liveSocketPath(t)
	runtime := NewJobRuntime(JobRuntimeConfig{Client: liveClient(t, socket)})

	job := &models.Job{ID: "idem-" + liveSuffix()}
	t.Cleanup(func() {
		_ = exec.Command("podman", "pod", "rm", "-f", JobPodName(job.ID)).Run()
	})

	first, err := runtime.CreateJobPod(context.Background(), job)
	if err != nil {
		t.Fatalf("first CreateJobPod: %v", err)
	}

	// The second call takes the recovery path: Podman rejects the duplicate name
	// with a 409, and adoption resolves it to the pod that already exists.
	second, err := runtime.CreateJobPod(context.Background(), job)
	if err != nil {
		t.Fatalf("second CreateJobPod should have adopted the existing pod, got: %v", err)
	}

	if first != second {
		t.Fatalf("pod IDs differ: first=%s second=%s; adoption must return the same pod", first, second)
	}
}

// A pod that happens to carry the same name but belongs to something else must be
// refused, not absorbed into this job.
func TestLiveContractCreateJobPodRefusesAForeignPod(t *testing.T) {
	socket := liveSocketPath(t)
	runtime := NewJobRuntime(JobRuntimeConfig{Client: liveClient(t, socket)})

	job := &models.Job{ID: "foreign-" + liveSuffix()}
	name := JobPodName(job.ID)

	out, err := exec.Command(
		"podman", "pod", "create",
		"--name", name,
		"--label", "managed_by=orchestrator",
		"--label", "job_id=someone-else",
	).CombinedOutput()
	if err != nil {
		t.Skipf("could not create the fixture pod: %v (%s)", err, string(out))
	}

	t.Cleanup(func() { _ = exec.Command("podman", "pod", "rm", "-f", name).Run() })

	if _, err := runtime.CreateJobPod(context.Background(), job); err == nil {
		t.Fatal("expected adoption to be refused for a pod labelled with another job")
	}
}
