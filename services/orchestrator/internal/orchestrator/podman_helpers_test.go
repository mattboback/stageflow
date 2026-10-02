package orchestrator

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/models"
	podman "github.com/mattboback/stageflow/services/orchestrator/internal/adapters/runtime"
)

func TestSpawnMonitorContainerBoundsScannerWait(t *testing.T) {
	t.Parallel()

	deadlineSeen := make(chan time.Duration, 1)
	client := &mockPodmanClient{
		waitContainerFunc: func(ctx context.Context, _ string) (*podman.ContainerWaitResponse, error) {
			deadline, ok := ctx.Deadline()
			if !ok {
				deadlineSeen <- 0
				return nil, errors.New("missing deadline")
			}

			deadlineSeen <- time.Until(deadline)

			return &podman.ContainerWaitResponse{StatusCode: 0}, nil
		},
	}

	orch := NewOrchestrator(&Config{DataDir: t.TempDir(), DirectoryOwner: func(string) error { return nil },
		PodmanClient: client,
		ScanTimeout:  25 * time.Millisecond,
	})

	orch.spawnMonitorContainer(context.Background(), "container-1", "job-1", "scanner-axe")
	orch.WaitForMonitors()

	select {
	case remaining := <-deadlineSeen:
		if remaining <= 0 {
			t.Fatalf("expected positive scanner monitor deadline, got %v", remaining)
		}
	case <-time.After(time.Second):
		t.Fatal("wait container was not called")
	}
}

func TestSpawnMonitorContainerRecoversPanic(t *testing.T) {
	t.Parallel()

	client := &mockPodmanClient{
		waitContainerFunc: func(context.Context, string) (*podman.ContainerWaitResponse, error) {
			panic("boom in monitor")
		},
	}

	orch := NewOrchestrator(
		&Config{DataDir: t.TempDir(), DirectoryOwner: func(string) error { return nil }, PodmanClient: client},
	)

	// If the panic were not recovered, the goroutine would crash the test
	// process. Reaching the assertion after WaitForMonitors proves recovery.
	orch.spawnMonitorContainer(context.Background(), "container-1", "job-1", "scanner-axe")
	orch.WaitForMonitors()
}

// exitingPodmanClient reports the given exit code for every container and,
// when oomKilled is set, an inspect State that says the kernel killed it.
func exitingPodmanClient(exitCode int, oomKilled bool) *mockPodmanClient {
	return &mockPodmanClient{
		waitContainerFunc: func(context.Context, string) (*podman.ContainerWaitResponse, error) {
			return &podman.ContainerWaitResponse{StatusCode: exitCode}, nil
		},
		inspectContainerFunc: func(_ context.Context, id string) (*podman.ContainerInfo, error) {
			return &podman.ContainerInfo{ID: id, OOMKilled: oomKilled}, nil
		},
	}
}

func insertScanningJob(t *testing.T, orch *Orchestrator, jobID string, scanners []string) {
	t.Helper()

	job := &models.Job{
		ID:        jobID,
		State:     models.JobStateScanning,
		InputType: "zip",
		InputPath: "test.zip",
		PodID:     "pod-" + jobID,
		Config:    models.JobConfig{Modules: scanners},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	insertJob(t, orch.database, job)

	if err := orch.database.SetExpectedScanners(context.Background(), jobID, scanners); err != nil {
		t.Fatalf("set expected scanners: %v", err)
	}
}

func completeScanner(t *testing.T, orch *Orchestrator, mem *memoryStorage, jobID, scanner string) {
	t.Helper()

	payload := &events.ScanCompletedPayload{
		JobID:             jobID,
		ScannerType:       scanner,
		ResultsPath:       jobID + "/" + scanner + "/results.json",
		TotalPagesScanned: 1,
		Summary:           events.ScanSummary{BySeverity: map[string]int{}},
	}
	seedScanResults(t, mem, jobID, payload.ResultsPath)

	if err := orch.HandleScanCompleted(context.Background(), payload); err != nil {
		t.Fatalf("HandleScanCompleted(%s): %v", scanner, err)
	}
}

func TestMonitorContainerKilledScannerKeepsOtherResults(t *testing.T) {
	orch, database, publisher, mem := setupTestOrchestratorWithConfig(t, func(c *Config) {
		c.PodmanClient = exitingPodmanClient(137, true)
	})

	insertScanningJob(t, orch, "job-oom", []string{"axe", "seo"})
	completeScanner(t, orch, mem, "job-oom", "seo")

	orch.spawnMonitorContainer(context.Background(), "container-axe", "job-oom", "scanner-axe")
	orch.WaitForMonitors()

	job, err := database.GetJob(context.Background(), "job-oom")
	if err != nil {
		t.Fatalf("get job: %v", err)
	}

	if job.State != models.JobStateDone {
		t.Fatalf("state = %s, want DONE", job.State)
	}

	axe := job.ScannerResults["axe"]
	if axe == nil || axe.Success {
		t.Fatalf("axe result = %+v, want a recorded failure", axe)
	}

	if !strings.HasPrefix(axe.Error, "axe ran out of memory") {
		t.Fatalf("axe error = %q, want the out-of-memory message", axe.Error)
	}

	if seo := job.ScannerResults["seo"]; seo == nil || !seo.Success {
		t.Fatalf("seo result = %+v, want success", seo)
	}

	if publisher.completedCount() != 1 || publisher.failedCount() != 0 {
		t.Fatalf("events: completed=%d failed=%d, want 1 and 0",
			publisher.completedCount(), publisher.failedCount())
	}
}

func TestMonitorContainerAllScannersExitingFailsJob(t *testing.T) {
	orch, database, publisher, _ := setupTestOrchestratorWithConfig(t, func(c *Config) {
		c.PodmanClient = exitingPodmanClient(1, false)
	})

	insertScanningJob(t, orch, "job-all", []string{"axe", "seo"})

	orch.spawnMonitorContainer(context.Background(), "container-axe", "job-all", "scanner-axe")
	orch.WaitForMonitors()

	job, err := database.GetJob(context.Background(), "job-all")
	if err != nil {
		t.Fatalf("get job: %v", err)
	}

	if job.State != models.JobStateScanning {
		t.Fatalf("state after the first exit = %s, want SCANNING", job.State)
	}

	if got := job.ScannerResults["axe"]; got == nil || got.Error != "scanner-axe container exited with code 1" {
		t.Fatalf("axe result = %+v, want the exit-code message", got)
	}

	orch.spawnMonitorContainer(context.Background(), "container-seo", "job-all", "scanner-seo")
	orch.WaitForMonitors()

	job, err = database.GetJob(context.Background(), "job-all")
	if err != nil {
		t.Fatalf("get job: %v", err)
	}

	if job.State != models.JobStateFailed {
		t.Fatalf("state = %s, want FAILED", job.State)
	}

	if publisher.failedCount() != 1 {
		t.Fatalf("job.failed events = %d, want 1", publisher.failedCount())
	}
}

func TestMonitorContainerExitAfterCompletionKeepsSuccess(t *testing.T) {
	orch, database, _, mem := setupTestOrchestratorWithConfig(t, func(c *Config) {
		c.PodmanClient = exitingPodmanClient(1, false)
	})

	insertScanningJob(t, orch, "job-late", []string{"axe", "seo"})
	completeScanner(t, orch, mem, "job-late", "axe")

	orch.spawnMonitorContainer(context.Background(), "container-axe", "job-late", "scanner-axe")
	orch.WaitForMonitors()

	job, err := database.GetJob(context.Background(), "job-late")
	if err != nil {
		t.Fatalf("get job: %v", err)
	}

	if axe := job.ScannerResults["axe"]; axe == nil || !axe.Success {
		t.Fatalf("axe result = %+v, want the recorded success kept", axe)
	}

	if job.State != models.JobStateScanning {
		t.Fatalf("state = %s, want SCANNING while seo is still running", job.State)
	}
}

func TestMonitorContainerExtractionExitFailsJob(t *testing.T) {
	orch, database, publisher, _ := setupTestOrchestratorWithConfig(t, func(c *Config) {
		c.PodmanClient = exitingPodmanClient(1, false)
	})

	job := &models.Job{
		ID:        "job-extract",
		State:     models.JobStateExtracting,
		InputType: "zip",
		InputPath: "test.zip",
		PodID:     "pod-extract",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	insertJob(t, database, job)

	orch.spawnMonitorContainer(context.Background(), "container-extract", "job-extract", "extraction")
	orch.WaitForMonitors()

	got, err := database.GetJob(context.Background(), "job-extract")
	if err != nil {
		t.Fatalf("get job: %v", err)
	}

	if got.State != models.JobStateFailed {
		t.Fatalf("state = %s, want FAILED", got.State)
	}

	if publisher.failedCount() != 1 {
		t.Fatalf("job.failed events = %d, want 1", publisher.failedCount())
	}
}
