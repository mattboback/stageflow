package podman

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"slices"
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

func TestBuildContainerCreateBodyHardensEveryContainer(t *testing.T) {
	for name, limits := range map[string]*ResourceLimits{
		"no limits":    nil,
		"memory limit": {MemoryLimitMB: 512, MemorySwapMB: 512},
	} {
		body := buildContainerCreateBody(&ContainerCreateRequest{Name: "c", Image: "i", ResourceLimits: limits})

		if !reflect.DeepEqual(body["cap_drop"], []string{"ALL"}) || body["no_new_privileges"] != true {
			t.Fatalf("%s: privileges = %v %v", name, body["cap_drop"], body["no_new_privileges"])
		}

		if body["read_only_filesystem"] != true || body["read_write_tmpfs"] != false {
			t.Fatalf("%s: root filesystem = %v, tmpfs defaults = %v",
				name, body["read_only_filesystem"], body["read_write_tmpfs"])
		}

		if _, ok := body["mounts"]; ok {
			t.Fatalf("%s: mounts = %#v, want none requested", name, body["mounts"])
		}

		volumes, _ := body["volumes"].([]map[string]any)
		if len(volumes) != 1 || volumes[0]["Name"] != "" || volumes[0]["Dest"] != "/tmp" {
			t.Fatalf("%s: volumes = %#v, want only an anonymous /tmp", name, body["volumes"])
		}

		if options, _ := volumes[0]["Options"].([]string); !slices.Contains(options, "noexec") {
			t.Fatalf("%s: /tmp options = %v", name, options)
		}

		resources, ok := body["resource_limits"].(map[string]any)
		if !ok || !reflect.DeepEqual(resources["pids"], map[string]int64{"limit": jobContainerPidsLimit}) {
			t.Fatalf("%s: resource_limits = %#v", name, body["resource_limits"])
		}

		if limits != nil && !reflect.DeepEqual(resources["memory"], map[string]int64{
			"limit": 512 * 1024 * 1024,
			"swap":  512 * 1024 * 1024,
		}) {
			t.Fatalf("%s: memory = %#v", name, resources["memory"])
		}
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
