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

func TestNewOrchestratorInitializesStableRuntime(t *testing.T) {
	database := newInMemoryDB(t)

	orch := NewOrchestrator(&Config{
		PodmanClient: &mockPodmanClient{},
		Database:     database,
		Publisher:    &mockPublisher{},
	})

	runtime := orch.jobRuntime
	if runtime == nil {
		t.Fatal("expected runtime to be initialized")
	}

	orch.podNetnsMode = podNetnsModeHost

	if got, want := orch.jobRuntime, runtime; got != want {
		t.Fatal("expected runtime to remain stable after construction")
	}

	if got, want := orch.PodNetnsMode(), podNetnsModeBridge; got != want {
		t.Fatalf("expected runtime pod netns mode %q, got %q", want, got)
	}
}
