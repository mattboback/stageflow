package podman

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestCreateContainer(t *testing.T) {
	mock := newMockPodmanServer()
	defer mock.Close()

	mock.handle("POST", "/v4.0.0/libpod/containers/create", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name  string            `json:"name"`
			Image string            `json:"image"`
			Pod   string            `json:"pod,omitempty"`
			Env   map[string]string `json:"env,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		if req.Image == "" {
			http.Error(w, "image required", http.StatusBadRequest)
			return
		}

		if len(req.Env) != 1 || req.Env["FOO"] != "bar" {
			t.Errorf("Expected env to contain FOO=bar, got %v", req.Env)
		}

		resp := ContainerCreateResponse{
			ID: "container-12345",
		}

		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("Failed to encode response: %v", err)
		}
	})

	client := mock.newClient()

	req := &ContainerCreateRequest{
		Name:  "test-container",
		Image: "alpine:latest",
		Pod:   "pod-123",
		Env:   map[string]string{"FOO": "bar"},
	}

	result, err := client.CreateContainer(context.Background(), req)
	if err != nil {
		t.Fatalf("Failed to create container: %v", err)
	}

	if result.ID != "container-12345" {
		t.Errorf("Expected container ID container-12345, got %s", result.ID)
	}
}

func TestStartContainer(t *testing.T) {
	mock := newMockPodmanServer()
	defer mock.Close()

	mock.handle("POST", "/v4.0.0/libpod/containers/container-123/start", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	client := mock.newClient()

	err := client.StartContainer(context.Background(), "container-123")
	if err != nil {
		t.Errorf("Failed to start container: %v", err)
	}
}

func TestStopContainer(t *testing.T) {
	mock := newMockPodmanServer()
	defer mock.Close()

	mock.handle("POST", "/v4.0.0/libpod/containers/container-123/stop", func(w http.ResponseWriter, r *http.Request) {
		timeout := r.URL.Query().Get("timeout")
		if timeout != "10" {
			t.Errorf("Expected timeout=10, got %s", timeout)
		}

		w.WriteHeader(http.StatusNoContent)
	})

	client := mock.newClient()

	err := client.StopContainer(context.Background(), "container-123", 10)
	if err != nil {
		t.Errorf("Failed to stop container: %v", err)
	}
}

func TestRemoveContainer(t *testing.T) {
	mock := newMockPodmanServer()
	defer mock.Close()

	mock.handle("DELETE", "/v4.0.0/libpod/containers/container-123", func(w http.ResponseWriter, r *http.Request) {
		force := r.URL.Query().Get("force")
		if force != "true" {
			t.Error("Expected force=true")
		}

		w.WriteHeader(http.StatusNoContent)
	})

	client := mock.newClient()

	err := client.RemoveContainer(context.Background(), "container-123", true)
	if err != nil {
		t.Errorf("Failed to remove container: %v", err)
	}
}

func TestInspectContainer(t *testing.T) {
	mock := newMockPodmanServer()
	defer mock.Close()

	mock.handle("GET", "/v4.0.0/libpod/containers/container-123/json", func(w http.ResponseWriter, _ *http.Request) {
		resp := ContainerInfo{
			ID:     "container-123",
			Name:   "test-container",
			State:  "running",
			Status: "Up 5 minutes",
		}

		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("Failed to encode response: %v", err)
		}
	})

	client := mock.newClient()

	result, err := client.InspectContainer(context.Background(), "container-123")
	if err != nil {
		t.Fatalf("Failed to inspect container: %v", err)
	}

	if result.ID != "container-123" {
		t.Errorf("Expected container ID container-123, got %s", result.ID)
	}

	if result.State != "running" {
		t.Errorf("Expected state running, got %s", result.State)
	}
}

func TestWaitContainer(t *testing.T) {
	mock := newMockPodmanServer()
	defer mock.Close()

	mock.handle("POST", "/v4.0.0/libpod/containers/container-123/wait", func(w http.ResponseWriter, _ *http.Request) {
		resp := ContainerWaitResponse{
			StatusCode: 0,
		}

		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("Failed to encode response: %v", err)
		}
	})

	client := mock.newClient()

	result, err := client.WaitContainer(context.Background(), "container-123")
	if err != nil {
		t.Fatalf("Failed to wait for container: %v", err)
	}

	if result.StatusCode != 0 {
		t.Errorf("Expected status code 0, got %d", result.StatusCode)
	}
}

func TestGetContainerLogs(t *testing.T) {
	mock := newMockPodmanServer()
	defer mock.Close()

	mock.handle("GET", "/v4.0.0/libpod/containers/container-123/logs", func(w http.ResponseWriter, r *http.Request) {
		stdout := r.URL.Query().Get("stdout")
		stderr := r.URL.Query().Get("stderr")

		if stdout != "true" || stderr != "true" {
			t.Error("Expected stdout=true and stderr=true")
		}

		if _, err := w.Write([]byte("container log output\n")); err != nil {
			t.Fatalf("Failed to write logs: %v", err)
		}
	})

	client := mock.newClient()

	logs, err := client.GetContainerLogs(context.Background(), "container-123", true, true)
	if err != nil {
		t.Fatalf("Failed to get container logs: %v", err)
	}

	if logs != "container log output\n" {
		t.Errorf("Expected log output, got: %s", logs)
	}
}
