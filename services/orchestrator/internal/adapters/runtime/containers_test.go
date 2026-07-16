package podman

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
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

func TestInspectContainerDecodesLibpodStateAndConfigLabels(t *testing.T) {
	mock := newMockPodmanServer()
	defer mock.Close()

	mock.handle("GET", "/v4.0.0/libpod/containers/scanner-axe-job/json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"Id":"container-123",
			"Name":"scanner-axe-job",
			"State":{"Status":"running","Running":true},
			"Config":{"Labels":{"managed_by":"orchestrator","job_id":"job"}}
		}`))
	})

	result, err := mock.newClient().InspectContainer(t.Context(), "scanner-axe-job")
	if err != nil {
		t.Fatalf("InspectContainer() error = %v", err)
	}

	if result.State != ContainerState("running") {
		t.Fatalf("state = %q, want running", result.State)
	}

	if result.labels()["job_id"] != "job" {
		t.Fatalf("labels = %#v", result.labels())
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

func TestWaitContainer_UsesLongPollClient(t *testing.T) {
	mock := newMockPodmanServer()
	defer mock.Close()

	mock.handle("POST", "/v4.0.0/libpod/containers/container-123/wait", func(w http.ResponseWriter, _ *http.Request) {
		resp := ContainerWaitResponse{StatusCode: 0}

		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("Failed to encode response: %v", err)
		}
	})

	client := &Client{
		httpClient: &http.Client{
			Transport: roundTripperFunc(func(_ *http.Request) (*http.Response, error) {
				return nil, errors.New("regular client should not be used for wait")
			}),
		},
		longPollHTTPClient: &http.Client{Transport: mock},
		baseURL:            mock.URL,
		apiPrefix:          libpodV4Prefix,
	}

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

func TestGetContainerLogsReturnsBoundedTail(t *testing.T) {
	mock := newMockPodmanServer()
	defer mock.Close()

	mock.handle("GET", "/v4.0.0/libpod/containers/container-123/logs", func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write([]byte(strings.Repeat("a", maxPodmanDiagnosticBytes+32) + "tail")); err != nil {
			t.Fatalf("Failed to write logs: %v", err)
		}
	})

	client := mock.newClient()

	logs, err := client.GetContainerLogs(context.Background(), "container-123", true, true)
	if err != nil {
		t.Fatalf("Failed to get container logs: %v", err)
	}

	if !strings.HasPrefix(logs, "[truncated] ") {
		t.Fatalf("expected truncated marker, got %q", logs[:min(len(logs), 20)])
	}

	if !strings.HasSuffix(logs, "tail") {
		t.Fatal("expected tail to be retained")
	}

	if len(logs) > maxPodmanDiagnosticBytes+len("[truncated] ") {
		t.Fatalf("logs length = %d, want <= %d", len(logs), maxPodmanDiagnosticBytes+len("[truncated] "))
	}
}

func TestGetContainerLogsBoundsErrorBody(t *testing.T) {
	mock := newMockPodmanServer()
	defer mock.Close()

	mock.handle("GET", "/v4.0.0/libpod/containers/container-123/logs", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)

		if _, err := w.Write([]byte(strings.Repeat("x", maxPodmanDiagnosticBytes+16) + "errtail")); err != nil {
			t.Fatalf("Failed to write logs: %v", err)
		}
	})

	client := mock.newClient()

	_, err := client.GetContainerLogs(context.Background(), "container-123", true, true)
	if err == nil {
		t.Fatal("expected API error")
	}

	if !strings.Contains(err.Error(), "[truncated] ") {
		t.Fatalf("expected truncated marker in error, got %v", err)
	}

	if !strings.Contains(err.Error(), "errtail") {
		t.Fatalf("expected tail in error, got %v", err)
	}
}

func TestBuildContainerCreateBodyMountOptions(t *testing.T) {
	req := &ContainerCreateRequest{
		Name:  "extraction-worker",
		Image: "extractor:latest",
		Mounts: []VolumeMount{
			{Type: "bind", Source: "/volumes/ws", Destination: "/workspace", ChownToContainerUser: true},
			{Type: "bind", Source: "/volumes/results", Destination: "/results", ReadOnly: true},
		},
	}

	body := buildContainerCreateBody(req)

	mounts, ok := body["mounts"].([]map[string]any)
	if !ok || len(mounts) != 2 {
		t.Fatalf("mounts = %#v, want 2 entries", body["mounts"])
	}

	wsOpts, _ := mounts[0]["options"].([]string)
	if !reflect.DeepEqual(wsOpts, []string{"rbind", "U"}) {
		t.Fatalf("workspace mount options = %v, want [rbind U]", wsOpts)
	}

	resultOpts, _ := mounts[1]["options"].([]string)
	if !reflect.DeepEqual(resultOpts, []string{"rbind", "ro"}) {
		t.Fatalf("results mount options = %v, want [rbind ro]", resultOpts)
	}
}
