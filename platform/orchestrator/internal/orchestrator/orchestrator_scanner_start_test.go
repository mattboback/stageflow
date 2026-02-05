package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/mattboback/stageflow/packages/shared-go/models"
	"github.com/mattboback/stageflow/platform/orchestrator/internal/podman"
)

func TestStartScannerReusesExistingWorkspaceVolume(t *testing.T) {
	orch, database, _, _ := setupTestOrchestrator(t)

	createCalls := 0
	inspectCounts := map[string]int{}

	mockClient := &mockPodmanClient{
		inspectVolumeFunc: func(_ context.Context, name string) (*podman.VolumeInfo, error) {
			inspectCounts[name]++
			if strings.HasPrefix(name, "workspace-") {
				return &podman.VolumeInfo{Name: name, Mountpoint: "/volumes/" + name}, nil
			}

			if strings.HasPrefix(name, "results-") {
				if inspectCounts[name] == 1 {
					return nil, errors.New("not found")
				}

				return &podman.VolumeInfo{Name: name, Mountpoint: "/volumes/" + name}, nil
			}

			return nil, fmt.Errorf("unexpected volume %s", name)
		},
		createVolumeFunc: func(_ context.Context, _ string) error {
			createCalls++
			return nil
		},
		createContainerFunc: func(_ context.Context, _ *podman.ContainerCreateRequest) (*podman.ContainerCreateResponse, error) {
			return &podman.ContainerCreateResponse{ID: "c1"}, nil
		},
		startContainerFunc: func(_ context.Context, _ string) error { return nil },
		waitContainerFunc: func(_ context.Context, _ string) (*podman.ContainerWaitResponse, error) {
			return &podman.ContainerWaitResponse{StatusCode: 0}, nil
		},
	}

	orch.podmanClient = mockClient

	job := &models.Job{
		ID:        "job-999",
		State:     models.JobStateReady,
		InputType: "zip",
		InputPath: "staging/job-999.zip",
		PodID:     "pod-1",
		Config:    models.JobConfig{Modules: []string{"axe"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := database.CreateJob(context.Background(), job); err != nil {
		t.Fatalf("failed to seed job: %v", err)
	}

	// Test single scanner start (the internal method used for each scanner type).
	if err := orch.startSingleScanner(context.Background(), job, "axe"); err != nil {
		t.Fatalf("startSingleScanner error: %v", err)
	}

	if createCalls != 1 { // only results volume should require creation
		t.Fatalf("expected 1 createVolume call, got %d", createCalls)
	}
}

func TestStartSingleScanner_URLJobUsesServiceNetworking(t *testing.T) {
	orch, database, _, _ := setupTestOrchestrator(t)

	var gotReq *podman.ContainerCreateRequest

	mockClient := &mockPodmanClient{
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

	orch.podmanClient = mockClient

	job := &models.Job{
		ID:        "job-urls",
		State:     models.JobStateReady,
		InputType: inputTypeURLs,
		URLs:      []string{"https://example.com"},
		PodID:     "pod-1",
		Config:    models.JobConfig{Modules: []string{"axe"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := database.CreateJob(context.Background(), job); err != nil {
		t.Fatalf("failed to seed job: %v", err)
	}

	if err := orch.startSingleScanner(context.Background(), job, "axe"); err != nil {
		t.Fatalf("startSingleScanner error: %v", err)
	}

	if gotReq == nil {
		t.Fatalf("expected container create request")
	}

	if got, want := gotReq.Env["NATS_URL"], "nats://nats:4222"; got != want {
		t.Fatalf("expected NATS_URL %q, got %q", want, got)
	}

	if got, want := gotReq.Env["MINIO_ENDPOINT"], "minio:9000"; got != want {
		t.Fatalf("expected MINIO_ENDPOINT %q, got %q", want, got)
	}
}
