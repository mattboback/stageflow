package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/mattboback/stageflow/packages/shared-go/models"
	podman "github.com/mattboback/stageflow/platform/orchestrator/internal/adapters/runtime"
)

func TestStartExtractionWorker_HostNetnsUsesLoopbackServiceEndpoints(t *testing.T) {
	var gotReq *podman.ContainerCreateRequest

	mockClient := &mockPodmanClient{
		createVolumeFunc: func(_ context.Context, _ string) error { return nil },
		inspectVolumeFunc: func(_ context.Context, name string) (*podman.VolumeInfo, error) {
			return &podman.VolumeInfo{Name: name, Mountpoint: "/volumes/" + name}, nil
		},
		createContainerFunc: func(_ context.Context, req *podman.ContainerCreateRequest) (*podman.ContainerCreateResponse, error) {
			gotReq = req
			return &podman.ContainerCreateResponse{ID: "c1"}, nil
		},
		startContainerFunc: func(_ context.Context, _ string) error { return nil },
		waitContainerFunc: func(_ context.Context, _ string) (*podman.ContainerWaitResponse, error) {
			return &podman.ContainerWaitResponse{StatusCode: 0}, nil
		},
	}

	orch, database, _, _ := setupTestOrchestratorWithConfig(t, func(config *Config) {
		config.PodmanClient = mockClient
		config.PodNetnsMode = podNetnsModeHost
	})
	defer orch.WaitForMonitors()

	job := &models.Job{
		ID:        "job-extraction-hostnetns",
		State:     models.JobStateExtracting,
		InputType: models.JobInputTypeZip,
		InputPath: "test.zip",
		PodID:     "pod-1",
		Config:    models.JobConfig{Modules: []string{"axe"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	insertJob(t, database, job)

	if err := orch.startExtractionWorker(context.Background(), job, "pod-1"); err != nil {
		t.Fatalf("startExtractionWorker error: %v", err)
	}

	if gotReq == nil {
		t.Fatalf("expected container create request")
	}

	if got, want := gotReq.Env["NATS_URL"], hostNetnsNATSURL; got != want {
		t.Fatalf("expected NATS_URL %q, got %q", want, got)
	}

	if got, want := gotReq.Env["MINIO_ENDPOINT"], hostNetnsMinioEndpoint; got != want {
		t.Fatalf("expected MINIO_ENDPOINT %q, got %q", want, got)
	}
}
