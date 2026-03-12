package orchestrator

import "testing"

func TestNewOrchestrator(t *testing.T) {
	database := newInMemoryDB(t)

	orch := NewOrchestrator(&Config{
		PodmanClient: &mockPodmanClient{},
		Database:     database,
		Publisher:    &mockPublisher{},
	})

	if orch == nil {
		t.Fatal("Expected orchestrator to be non-nil")
	}

	if !orch.canTransition("PENDING", "EXTRACTING") {
		t.Error("Expected domain transition helper to be initialized")
	}
}

func TestNewOrchestratorInitializesStableRuntimeAdapter(t *testing.T) {
	database := newInMemoryDB(t)

	orch := NewOrchestrator(&Config{
		PodmanClient: &mockPodmanClient{},
		Database:     database,
		Publisher:    &mockPublisher{},
	})

	adapter := orch.runtimeAdapter()
	if adapter == nil {
		t.Fatal("expected runtime adapter to be initialized")
	}

	orch.podNetnsMode = podNetnsModeHost

	if got, want := orch.runtimeAdapter(), adapter; got != want {
		t.Fatal("expected runtime adapter to remain stable after construction")
	}

	if got, want := adapter.PodNetnsMode(), podNetnsModeBridge; got != want {
		t.Fatalf("expected runtime adapter pod netns mode %q, got %q", want, got)
	}
}
