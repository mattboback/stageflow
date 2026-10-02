package podman

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
	// OOMKilled is set from the inspect State object. List responses carry
	// State as a string, so it stays false there.
	OOMKilled bool `json:"-"`
	ExitCode  int  `json:"ExitCode"`
}

func (c *ContainerInfo) UnmarshalJSON(data []byte) error {
	type plain ContainerInfo

	var info plain
	if err := json.Unmarshal(data, &info); err != nil {
		return err
	}

	var raw struct {
		State json.RawMessage `json:"State"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("decode container state: %w", err)
	}

	var state struct {
		OOMKilled bool `json:"OOMKilled"`
		ExitCode  int  `json:"ExitCode"`
	}
	// A string State has no OOMKilled; only the object form is decoded.
	if len(raw.State) > 0 && raw.State[0] == '{' {
		if err := json.Unmarshal(raw.State, &state); err != nil {
			return fmt.Errorf("decode container state: %w", err)
		}
	}

	*c = ContainerInfo(info)

	c.OOMKilled = state.OOMKilled
	if len(raw.State) > 0 && raw.State[0] == '{' {
		c.ExitCode = state.ExitCode
	}

	return nil
}

// ListJobContainers obtains one label-scoped snapshot for each reconciliation.
func (c *Client) ListJobContainers(ctx context.Context) ([]ContainerInfo, error) {
	filter, err := json.Marshal(map[string][]string{"label": {"managed_by=orchestrator"}})
	if err != nil {
		return nil, err
	}

	response, err := c.doLibpodRequest(
		ctx,
		http.MethodGet,
		"/containers/json?all=true&filters="+url.QueryEscape(string(filter)),
		nil,
	)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	var containers []ContainerInfo
	if parseResponseErr := parseResponse(response, &containers); parseResponseErr != nil {
		return nil, parseResponseErr
	}

	return containers, nil
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
