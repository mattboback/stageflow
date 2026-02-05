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

	if orch.stateMachine == nil {
		t.Error("Expected state machine to be initialized")
	}
}
