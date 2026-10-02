package podman

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
)

func TestCreateContainer(t *testing.T) {
	mock := newMockPodmanServer()
	defer mock.Close()

	mock.handle("POST", "/v5.0.0/libpod/containers/create", func(w http.ResponseWriter, r *http.Request) {
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

	mock.handle("POST", "/v5.0.0/libpod/containers/container-123/start", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	client := mock.newClient()

	err := client.StartContainer(context.Background(), "container-123")
	if err != nil {
		t.Errorf("Failed to start container: %v", err)
	}
}

func TestInspectContainer(t *testing.T) {
	mock := newMockPodmanServer()
	defer mock.Close()

	mock.handle("GET", "/v5.0.0/libpod/containers/container-123/json", func(w http.ResponseWriter, _ *http.Request) {
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

	mock.handle("GET", "/v5.0.0/libpod/containers/scanner-axe-job/json", func(w http.ResponseWriter, _ *http.Request) {
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

func TestContainerInfoReadsOOMKilledFromInspectState(t *testing.T) {
	t.Parallel()

	var inspected ContainerInfo
	if err := json.Unmarshal(
		[]byte(`{"Id":"abc","State":{"Status":"exited","OOMKilled":true}}`),
		&inspected,
	); err != nil {
		t.Fatalf("decode inspect response: %v", err)
	}

	if inspected.State != "exited" || !inspected.OOMKilled {
		t.Fatalf("inspect = %+v, want exited and OOMKilled", inspected)
	}

	var listed ContainerInfo
	if err := json.Unmarshal([]byte(`{"Id":"abc","State":"running"}`), &listed); err != nil {
		t.Fatalf("decode list entry: %v", err)
	}

	if listed.State != "running" || listed.OOMKilled {
		t.Fatalf("list entry = %+v, want running and not OOMKilled", listed)
	}
}
