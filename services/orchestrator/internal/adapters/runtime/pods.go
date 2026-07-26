package podman

import (
	"context"
	"fmt"
	"net/http"
)

// PodCreateRequest models the subset of Podman pod create input StageFlow uses.
type PodCreateRequest struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels,omitempty"`
	// Networks attaches the pod to existing CNI networks (name -> options).
	Networks map[string]PerNetworkOptions `json:"networks,omitempty"`
	// Netns sets the network namespace mode (e.g., "bridge", "host", "none").
	// Must be "bridge" when using custom Networks.
	Netns PodNetns `json:"netns,omitempty"`
	// HostAdd adds custom host-to-IP mappings to /etc/hosts in all pod containers.
	// Format: "hostname:ip" (e.g., "mysite.com:169.254.1.2").
	HostAdd []string `json:"hostadd,omitempty"`
}

// PodNetns configures the network namespace mode for a pod.
type PodNetns struct {
	Nsmode string `json:"nsmode"`
}

// PerNetworkOptions holds per-network configuration (aliases only for now).
type PerNetworkOptions struct {
	Aliases []string `json:"aliases,omitempty"`
}

// PodCreateResponse captures Podman's create response (pod ID).
type PodCreateResponse struct {
	ID string `json:"Id"`
}

// PodInfo is a reduced view of Podman pod list/inspect output for API responses.
//
// Labels carries the pod's own labels, which Podman returns at the top level of
// the inspect report (unlike containers, which nest theirs under Config). It is
// what lets pod adoption verify ownership before reusing a pod it did not
// observe being created — see adoptPod in job_runtime.go.
type PodInfo struct {
	ID     string            `json:"Id"`
	Name   string            `json:"Name"`
	Status string            `json:"Status"`
	Labels map[string]string `json:"Labels,omitempty"`
}

// CreatePod calls the Podman pods/create endpoint.
func (c *Client) CreatePod(ctx context.Context, req *PodCreateRequest) (*PodCreateResponse, error) {
	resp, err := c.doLibpodRequest(ctx, http.MethodPost, "/pods/create", req)
	if err != nil {
		return nil, fmt.Errorf("failed to create pod: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	var result PodCreateResponse
	if parseErr := parseResponse(resp, &result); parseErr != nil {
		return nil, parseErr
	}

	return &result, nil
}

// InspectPod fetches Podman's pod JSON for status endpoints.
func (c *Client) InspectPod(ctx context.Context, podID string) (*PodInfo, error) {
	path := fmt.Sprintf("/pods/%s/json", podID)

	resp, err := c.doLibpodRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect pod: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	var result PodInfo
	if parseErr := parseResponse(resp, &result); parseErr != nil {
		return nil, parseErr
	}

	return &result, nil
}

// StartPod requests Podman to start a pod.
func (c *Client) StartPod(ctx context.Context, podID string) error {
	path := fmt.Sprintf("/pods/%s/start", podID)

	resp, err := c.doLibpodRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return fmt.Errorf("failed to start pod: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	return parseResponse(resp, nil)
}

// StopPod requests Podman to stop a pod.
func (c *Client) StopPod(ctx context.Context, podID string) error {
	path := fmt.Sprintf("/pods/%s/stop", podID)

	resp, err := c.doLibpodRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return fmt.Errorf("failed to stop pod: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	return parseResponse(resp, nil)
}

// RemovePod deletes a pod, optionally forcing removal.
func (c *Client) RemovePod(ctx context.Context, podID string, force bool) error {
	path := fmt.Sprintf("/pods/%s?force=%t", podID, force)

	resp, err := c.doLibpodRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("failed to remove pod: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	return parseResponse(resp, nil)
}

// ListPods retrieves all Podman pods visible to the service.
func (c *Client) ListPods(ctx context.Context) ([]PodInfo, error) {
	resp, err := c.doLibpodRequest(ctx, http.MethodGet, "/pods/json", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	var result []PodInfo
	if parseErr := parseResponse(resp, &result); parseErr != nil {
		return nil, parseErr
	}

	return result, nil
}
