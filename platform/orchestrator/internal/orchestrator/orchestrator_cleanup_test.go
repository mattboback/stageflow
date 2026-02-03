package orchestrator

import (
	"context"
	"testing"
)

func TestCleanupPod(t *testing.T) {
	stopCalled := false
	removeCalled := false

	mockClient := &mockPodmanClient{
		stopPodFunc: func(_ context.Context, podID string) error {
			stopCalled = true
			if podID != "pod-123" {
				t.Errorf("Expected pod ID pod-123, got %s", podID)
			}
			return nil
		},
		removePodFunc: func(_ context.Context, podID string, force bool) error {
			removeCalled = true
			if podID != "pod-123" {
				t.Errorf("Expected pod ID pod-123, got %s", podID)
			}
			if !force {
				t.Error("Expected force=true")
			}
			return nil
		},
	}

	database := newInMemoryDB(t)

	orch := NewOrchestrator(&Config{
		PodmanClient: mockClient,
		Database:     database,
		Publisher:    &mockPublisher{},
	})

	err := orch.cleanupPod(context.Background(), "pod-123")
	if err != nil {
		t.Fatalf("cleanupPod failed: %v", err)
	}

	if !stopCalled {
		t.Error("Expected StopPod to be called")
	}
	if !removeCalled {
		t.Error("Expected RemovePod to be called")
	}
}

func TestCleanupVolumes(t *testing.T) {
	var removed []string
	mockClient := &mockPodmanClient{
		removeVolumeFunc: func(_ context.Context, name string, force bool) error {
			removed = append(removed, name)
			if !force {
				t.Error("expected force=true")
			}
			return nil
		},
	}

	database := newInMemoryDB(t)

	orch := NewOrchestrator(&Config{
		PodmanClient: mockClient,
		Database:     database,
		Publisher:    &mockPublisher{},
	})

	orch.cleanupVolumes(context.Background(), "job-xyz")

	expected := []string{"workspace-job-xyz", "results-job-xyz"}
	if len(removed) != len(expected) {
		t.Fatalf("expected %d volumes removed, got %d", len(expected), len(removed))
	}
	for i, name := range expected {
		if removed[i] != name {
			t.Errorf("expected %s, got %s", name, removed[i])
		}
	}
}
