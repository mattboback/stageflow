package test

import (
	"context"
	"fmt"
	"sync"

	podman "github.com/mattboback/stageflow/services/orchestrator/internal/adapters/runtime"
)

// mockPodmanClient implements orchestrator.PodmanClient interface for testing.
type mockPodmanClient struct {
	mu         sync.RWMutex
	pods       map[string]*mockPod
	containers map[string]*mockContainer
	volumes    map[string]*mockVolume
}

type mockPod struct {
	ID     string
	Name   string
	State  string
	Labels map[string]string
}

type mockContainer struct {
	ID     string
	Name   string
	PodID  string
	State  string
	Labels map[string]string
}

type mockVolume struct {
	Name       string
	Mountpoint string
}

func newMockPodmanClient() *mockPodmanClient {
	return &mockPodmanClient{
		pods:       make(map[string]*mockPod),
		containers: make(map[string]*mockContainer),
		volumes:    make(map[string]*mockVolume),
	}
}

// InspectPod resolves by ID or by name. CreatePod keys pods as "pod-"+Name, while
// the runtime's idempotent recovery path inspects by the bare pod name, so both
// have to work or adoption silently never matches.
func (m *mockPodmanClient) InspectPod(_ context.Context, podID string) (*podman.PodInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pod, ok := m.pods[podID]
	if !ok {
		for _, candidate := range m.pods {
			if candidate.Name == podID {
				pod = candidate
				ok = true

				break
			}
		}
	}

	if !ok {
		return nil, &podman.APIError{StatusCode: 404, Body: "pod not found"}
	}

	return &podman.PodInfo{
		ID:     pod.ID,
		Name:   pod.Name,
		Status: pod.State,
		Labels: pod.Labels,
	}, nil
}

func (m *mockPodmanClient) CreatePod(
	_ context.Context,
	req *podman.PodCreateRequest,
) (*podman.PodCreateResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	podID := "pod-" + req.Name
	m.pods[podID] = &mockPod{
		ID:     podID,
		Name:   req.Name,
		State:  "running",
		Labels: req.Labels,
	}

	return &podman.PodCreateResponse{ID: podID}, nil
}

func (m *mockPodmanClient) StopPod(_ context.Context, podID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if pod, exists := m.pods[podID]; exists {
		pod.State = "stopped"
	}

	return nil
}

func (m *mockPodmanClient) RemovePod(_ context.Context, podID string, _ bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.pods, podID)

	return nil
}

func (m *mockPodmanClient) CreateVolume(_ context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.volumes[name]; !exists {
		m.volumes[name] = &mockVolume{
			Name:       name,
			Mountpoint: fmt.Sprintf("/volumes/%s/_data", name),
		}
	}

	return nil
}

func (m *mockPodmanClient) InspectVolume(_ context.Context, name string) (*podman.VolumeInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if volume, ok := m.volumes[name]; ok {
		return &podman.VolumeInfo{
			Name:       volume.Name,
			Mountpoint: volume.Mountpoint,
		}, nil
	}

	return nil, fmt.Errorf("volume %s not found", name)
}

func (m *mockPodmanClient) RemoveVolume(_ context.Context, name string, _ bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.volumes, name)

	return nil
}

func (m *mockPodmanClient) CreateContainer(
	_ context.Context,
	req *podman.ContainerCreateRequest,
) (*podman.ContainerCreateResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := "container-" + req.Name
	m.containers[id] = &mockContainer{
		ID:     id,
		Name:   req.Name,
		PodID:  req.Pod,
		State:  "created",
		Labels: req.Labels,
	}

	return &podman.ContainerCreateResponse{ID: id}, nil
}

func (m *mockPodmanClient) StartContainer(_ context.Context, containerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if c, ok := m.containers[containerID]; ok {
		c.State = "running"
	}

	return nil
}

func (m *mockPodmanClient) InspectContainer(
	_ context.Context,
	containerID string,
) (*podman.ContainerInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, container := range m.containers {
		if container.ID == containerID || container.Name == containerID {
			return &podman.ContainerInfo{
				ID:     container.ID,
				Name:   container.Name,
				State:  podman.ContainerState(container.State),
				Labels: container.Labels,
			}, nil
		}
	}

	return nil, &podman.APIError{StatusCode: 404, Body: "container not found"}
}

func (m *mockPodmanClient) WaitContainer(_ context.Context, containerID string) (*podman.ContainerWaitResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if c, ok := m.containers[containerID]; ok {
		c.State = "exited"
	}

	return &podman.ContainerWaitResponse{StatusCode: 0}, nil
}

func (m *mockPodmanClient) RemoveContainer(_ context.Context, containerID string, _ bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.containers, containerID)

	return nil
}

func (m *mockPodmanClient) GetContainerLogs(_ context.Context, _ string, _, _ bool) (string, error) {
	return "mock container logs", nil
}
