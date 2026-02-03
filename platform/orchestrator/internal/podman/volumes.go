package podman

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// VolumeInfo is the Podman volume view used to resolve mountpoints for job workspaces.
type VolumeInfo struct {
	Name       string `json:"Name"`
	Mountpoint string `json:"Mountpoint"`
}

// CreateVolume creates a named volume (idempotent).
func (c *Client) CreateVolume(ctx context.Context, name string) error {
	// Tell Podman to treat this as idempotent if the volume already exists.
	body := map[string]any{"Name": name, "IgnoreIfExists": true}

	resp, err := c.doLibpodRequest(ctx, http.MethodPost, "/volumes/create", body)
	if err != nil {
		return fmt.Errorf("failed to create volume: %w", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= 400 {
		data, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("API error (status %d): failed to read response body: %w", resp.StatusCode, readErr)
		}

		if strings.Contains(strings.ToLower(string(data)), "already exists") {
			// Idempotent: treat existing volume as success
			return nil
		}

		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(data))
	}

	if resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(&VolumeInfo{}); err != nil {
			return fmt.Errorf("failed to decode volume response: %w", err)
		}
	}

	return nil
}

// InspectVolume fetches Podman volume JSON to resolve the mountpoint.
func (c *Client) InspectVolume(ctx context.Context, name string) (*VolumeInfo, error) {
	path := fmt.Sprintf("/volumes/%s/json", name)

	resp, err := c.doLibpodRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect volume: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	var info VolumeInfo
	if err := parseResponse(resp, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

// RemoveVolume deletes a volume, optionally forcing cleanup.
func (c *Client) RemoveVolume(ctx context.Context, name string, force bool) error {
	path := fmt.Sprintf("/volumes/%s?force=%t", name, force)

	resp, err := c.doLibpodRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("failed to remove volume: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	return parseResponse(resp, nil)
}
