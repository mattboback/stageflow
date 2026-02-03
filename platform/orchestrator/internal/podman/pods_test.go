package podman

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestCreatePod(t *testing.T) {
	mock := newMockPodmanServer()
	defer mock.Close()

	mock.handle("POST", "/v4.0.0/libpod/pods/create", func(w http.ResponseWriter, r *http.Request) {
		var req PodCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		if req.Name == "" {
			http.Error(w, "name required", http.StatusBadRequest)
			return
		}

		resp := PodCreateResponse{
			ID: "pod-12345",
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("Failed to encode response: %v", err)
		}
	})

	client := mock.newClient()

	req := &PodCreateRequest{
		Name: "test-pod",
		Labels: map[string]string{
			"managed_by": "orchestrator",
		},
	}

	result, err := client.CreatePod(context.Background(), req)
	if err != nil {
		t.Fatalf("Failed to create pod: %v", err)
	}

	if result.ID != "pod-12345" {
		t.Errorf("Expected pod ID pod-12345, got %s", result.ID)
	}
}

func TestInspectPod(t *testing.T) {
	mock := newMockPodmanServer()
	defer mock.Close()

	mock.handle("GET", "/v4.0.0/libpod/pods/pod-123/json", func(w http.ResponseWriter, _ *http.Request) {
		resp := PodInfo{
			ID:     "pod-123",
			Name:   "test-pod",
			Status: "Running",
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("Failed to encode response: %v", err)
		}
	})

	client := mock.newClient()

	result, err := client.InspectPod(context.Background(), "pod-123")
	if err != nil {
		t.Fatalf("Failed to inspect pod: %v", err)
	}

	if result.ID != "pod-123" {
		t.Errorf("Expected pod ID pod-123, got %s", result.ID)
	}
	if result.Status != "Running" {
		t.Errorf("Expected status Running, got %s", result.Status)
	}
}

func TestStartPod(t *testing.T) {
	mock := newMockPodmanServer()
	defer mock.Close()

	mock.handle("POST", "/v4.0.0/libpod/pods/pod-123/start", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	client := mock.newClient()

	err := client.StartPod(context.Background(), "pod-123")
	if err != nil {
		t.Errorf("Failed to start pod: %v", err)
	}
}

func TestStopPod(t *testing.T) {
	mock := newMockPodmanServer()
	defer mock.Close()

	mock.handle("POST", "/v4.0.0/libpod/pods/pod-123/stop", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	client := mock.newClient()

	err := client.StopPod(context.Background(), "pod-123")
	if err != nil {
		t.Errorf("Failed to stop pod: %v", err)
	}
}

func TestRemovePod(t *testing.T) {
	mock := newMockPodmanServer()
	defer mock.Close()

	mock.handle("DELETE", "/v4.0.0/libpod/pods/pod-123", func(w http.ResponseWriter, r *http.Request) {
		force := r.URL.Query().Get("force")
		if force != "true" {
			t.Error("Expected force=true")
		}
		w.WriteHeader(http.StatusNoContent)
	})

	client := mock.newClient()

	err := client.RemovePod(context.Background(), "pod-123", true)
	if err != nil {
		t.Errorf("Failed to remove pod: %v", err)
	}
}

func TestListPods(t *testing.T) {
	mock := newMockPodmanServer()
	defer mock.Close()

	mock.handle("GET", "/v4.0.0/libpod/pods/json", func(w http.ResponseWriter, _ *http.Request) {
		resp := []PodInfo{
			{ID: "pod-1", Name: "pod-one", Status: "Running"},
			{ID: "pod-2", Name: "pod-two", Status: "Stopped"},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("Failed to encode response: %v", err)
		}
	})

	client := mock.newClient()

	result, err := client.ListPods(context.Background())
	if err != nil {
		t.Fatalf("Failed to list pods: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 pods, got %d", len(result))
	}
	if result[0].ID != "pod-1" {
		t.Errorf("Expected first pod ID pod-1, got %s", result[0].ID)
	}
}
