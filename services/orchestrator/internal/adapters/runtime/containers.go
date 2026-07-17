package podman

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// ResourceLimits configures container resource constraints.
type ResourceLimits struct {
	MemoryLimitMB int64 `json:"memory_limit_mb,omitempty"` // Memory limit in MB
	MemorySwapMB  int64 `json:"memory_swap_mb,omitempty"`  // Memory + swap limit (same as memory to disable swap)
	CPUQuota      int64 `json:"cpu_quota,omitempty"`       // CPU quota in microseconds per period
	CPUPeriod     int64 `json:"cpu_period,omitempty"`      // CPU period (default 100000)
	CPUShares     int64 `json:"cpu_shares,omitempty"`      // Relative CPU weight
}

// ContainerCreateRequest models the subset of the Podman Libpod create API StageFlow needs.
type ContainerCreateRequest struct {
	Name           string            `json:"name"`
	Image          string            `json:"image"`
	Pod            string            `json:"pod,omitempty"`
	User           string            `json:"user,omitempty"`
	Env            map[string]string `json:"env,omitempty"`
	Mounts         []VolumeMount     `json:"mounts,omitempty"`
	Labels         map[string]string `json:"labels,omitempty"`
	Command        []string          `json:"command,omitempty"`
	ResourceLimits *ResourceLimits   `json:"resource_limits,omitempty"`
}

// VolumeMount describes a bind/volume mount in Podman requests.
type VolumeMount struct {
	Type        string `json:"type,omitempty"`
	Source      string `json:"source,omitempty"`
	Destination string `json:"destination"`
	ReadOnly    bool   `json:"read_only,omitempty"`
	// ChownToContainerUser adds Podman's "U" mount option, which recursively
	// chowns the mount source to the container's runtime user at start. Needed
	// when a non-root container user must write to a freshly created volume,
	// whose mountpoint is otherwise owned by container-root.
	ChownToContainerUser bool `json:"chown_to_container_user,omitempty"`
}

// ContainerCreateResponse captures the Podman create result (container ID plus warnings).
type ContainerCreateResponse struct {
	ID       string   `json:"Id"`
	Warnings []string `json:"Warnings,omitempty"`
}

// ContainerInfo is a reduced Podman inspect view used for status and labels.
type ContainerState string

func (s *ContainerState) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err == nil {
		*s = ContainerState(value)

		return nil
	}

	var object struct {
		Status string `json:"Status"`
	}
	if err := json.Unmarshal(data, &object); err != nil {
		return fmt.Errorf("decode container state: %w", err)
	}

	*s = ContainerState(object.Status)

	return nil
}

func (s ContainerState) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(s))
}

type ContainerInfo struct {
	ID     string            `json:"Id"`
	Name   string            `json:"Name"`
	State  ContainerState    `json:"State"`
	Status string            `json:"Status"`
	Labels map[string]string `json:"Labels,omitempty"`
	Config struct {
		Labels map[string]string `json:"Labels,omitempty"`
	} `json:"Config,omitempty"`
}

func (c *ContainerInfo) labels() map[string]string {
	if c == nil {
		return nil
	}

	if len(c.Labels) > 0 {
		return c.Labels
	}

	return c.Config.Labels
}

// ContainerWaitResponse mirrors Podman's wait output for exit status checks.
type ContainerWaitResponse struct {
	StatusCode int    `json:"StatusCode"`
	Error      string `json:"Error,omitempty"`
}

// CreateContainer calls the Podman containers/create endpoint.
func (c *Client) CreateContainer(ctx context.Context, req *ContainerCreateRequest) (*ContainerCreateResponse, error) {
	// Build the Podman API request body with properly nested resource_limits
	body := buildContainerCreateBody(req)

	resp, err := c.doLibpodRequest(ctx, http.MethodPost, "/containers/create", body)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	var result ContainerCreateResponse
	if parseErr := parseResponse(resp, &result); parseErr != nil {
		return nil, parseErr
	}

	return &result, nil
}

// buildContainerCreateBody constructs the Podman Libpod API request body with properly nested resource_limits.
//
//nolint:gocognit,gocyclo // Container configuration requires multiple optional field checks
func buildContainerCreateBody(req *ContainerCreateRequest) map[string]any {
	body := map[string]any{
		"name":  req.Name,
		"image": req.Image,
	}

	if req.Pod != "" {
		body["pod"] = req.Pod
	}

	if req.User != "" {
		body["user"] = req.User
	}

	if len(req.Env) > 0 {
		// Podman Libpod expects env as a map of "KEY": "VALUE".
		body["env"] = req.Env
	}

	//nolint:nestif // Mount configuration requires multiple optional field handling
	if len(req.Mounts) > 0 {
		// Podman expects OCI runtime-spec mounts (`specs-go.Mount`) where read-only is
		// expressed via mount options (e.g., "ro"), not a separate boolean field.
		mounts := make([]map[string]any, 0, len(req.Mounts))
		for _, m := range req.Mounts {
			mount := map[string]any{
				"destination": m.Destination,
			}
			if m.Type != "" {
				mount["type"] = m.Type
			}

			if m.Source != "" {
				mount["source"] = m.Source
			}

			opts := make([]string, 0, 2)
			// Bind mounts are typically recursive in Podman; be explicit.
			if m.Type == "bind" {
				opts = append(opts, "rbind")
			}

			if m.ReadOnly {
				opts = append(opts, "ro")
			}

			if m.ChownToContainerUser {
				opts = append(opts, "U")
			}

			if len(opts) > 0 {
				mount["options"] = opts
			}

			mounts = append(mounts, mount)
		}

		body["mounts"] = mounts
	}

	if len(req.Labels) > 0 {
		body["labels"] = req.Labels
	}

	if len(req.Command) > 0 {
		body["command"] = req.Command
	}

	// Build nested resource_limits structure for Podman API
	//nolint:nestif // Resource limits require multiple nested optional fields
	if req.ResourceLimits != nil {
		resourceLimits := make(map[string]any)

		if req.ResourceLimits.MemoryLimitMB > 0 {
			memoryLimits := map[string]int64{
				"limit": req.ResourceLimits.MemoryLimitMB * 1024 * 1024, // Convert MB to bytes
			}
			if req.ResourceLimits.MemorySwapMB > 0 {
				memoryLimits["swap"] = req.ResourceLimits.MemorySwapMB * 1024 * 1024
			}

			resourceLimits["memory"] = memoryLimits
		}

		if req.ResourceLimits.CPUQuota > 0 || req.ResourceLimits.CPUShares > 0 {
			cpuLimits := make(map[string]int64)
			if req.ResourceLimits.CPUQuota > 0 {
				cpuLimits["quota"] = req.ResourceLimits.CPUQuota
			}

			if req.ResourceLimits.CPUPeriod > 0 {
				cpuLimits["period"] = req.ResourceLimits.CPUPeriod
			}

			if req.ResourceLimits.CPUShares > 0 {
				cpuLimits["shares"] = req.ResourceLimits.CPUShares
			}

			resourceLimits["cpu"] = cpuLimits
		}

		if len(resourceLimits) > 0 {
			body["resource_limits"] = resourceLimits
		}
	}

	return body
}

// StartContainer requests Podman to start an existing container.
func (c *Client) StartContainer(ctx context.Context, containerID string) error {
	path := fmt.Sprintf("/containers/%s/start", containerID)

	resp, err := c.doLibpodRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	return parseResponse(resp, nil)
}

// StopContainer requests Podman to stop a container with a timeout in seconds.
func (c *Client) StopContainer(ctx context.Context, containerID string, timeout int) error {
	path := fmt.Sprintf("/containers/%s/stop?timeout=%d", containerID, timeout)

	resp, err := c.doLibpodRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	return parseResponse(resp, nil)
}

// RemoveContainer deletes a container, optionally forcing removal.
func (c *Client) RemoveContainer(ctx context.Context, containerID string, force bool) error {
	path := fmt.Sprintf("/containers/%s?force=%t", containerID, force)

	resp, err := c.doLibpodRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	return parseResponse(resp, nil)
}

// InspectContainer fetches Podman's container JSON for status endpoints.
func (c *Client) InspectContainer(ctx context.Context, containerID string) (*ContainerInfo, error) {
	path := fmt.Sprintf("/containers/%s/json", containerID)

	resp, err := c.doLibpodRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	var result ContainerInfo
	if parseErr := parseResponse(resp, &result); parseErr != nil {
		return nil, parseErr
	}

	return &result, nil
}

// WaitContainer blocks until a container exits and returns Podman's status code.
func (c *Client) WaitContainer(ctx context.Context, containerID string) (*ContainerWaitResponse, error) {
	path := fmt.Sprintf("/containers/%s/wait", containerID)

	resp, err := c.doLibpodLongPollRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for container: %w", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	body, truncated, err := readBoundedPodmanBody(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read wait response: %w", err)
	}

	if truncated {
		return nil, fmt.Errorf("failed to read wait response: response exceeded %d bytes", maxPodmanDiagnosticBytes)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Podman may return either a JSON object or a bare status code (number).
	var result ContainerWaitResponse
	if unmarshalErr := json.Unmarshal(body, &result); unmarshalErr == nil {
		// Podman may return {"StatusCode":0} when the container exits cleanly.
		return &result, nil
	}

	// Fallback: try to parse as a bare integer
	codeStr := strings.TrimSpace(string(body))
	if codeStr != "" {
		code, convErr := strconv.Atoi(codeStr)
		if convErr == nil {
			return &ContainerWaitResponse{StatusCode: code}, nil
		}
	}

	return nil, fmt.Errorf("failed to decode wait response: %s", string(body))
}

// GetContainerLogs streams stdout/stderr logs from Podman.
func (c *Client) GetContainerLogs(ctx context.Context, containerID string, stdout, stderr bool) (string, error) {
	path := fmt.Sprintf("/containers/%s/logs?stdout=%t&stderr=%t", containerID, stdout, stderr)

	resp, err := c.doLibpodRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get container logs: %w", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	logs, truncated, err := readPodmanTail(resp.Body, maxPodmanDiagnosticBytes)
	if err != nil {
		return "", fmt.Errorf("failed to read logs: %w", err)
	}

	if truncated {
		logs = "[truncated] " + logs
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, logs)
	}

	return logs, nil
}
