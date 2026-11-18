package podman

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// jobNetworkIsolation is the Podman network option value the job network must
// carry. "strict" blocks traffic to every other Podman network; "true" only
// blocks networks that set the option themselves, so it would leave the stack
// network reachable.
const jobNetworkIsolation = "strict"

// NetworkInfo is the part of Podman's network inspect output StageFlow reads.
type NetworkInfo struct {
	Name    string            `json:"name"`
	Driver  string            `json:"driver"`
	Options map[string]string `json:"options,omitempty"`
}

// InspectNetwork fetches a Podman network by name.
func (c *Client) InspectNetwork(ctx context.Context, name string) (*NetworkInfo, error) {
	resp, err := c.doLibpodRequest(ctx, http.MethodGet, "/networks/"+url.PathEscape(name)+"/json", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect network: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	var info NetworkInfo
	if parseErr := parseResponse(resp, &info); parseErr != nil {
		return nil, parseErr
	}

	return &info, nil
}

type networkInspector interface {
	InspectNetwork(ctx context.Context, name string) (*NetworkInfo, error)
}

// VerifyJobNetwork refuses a job network that can reach other Podman networks.
//
// Job pods load untrusted pages. On a network shared with the stack, such a page
// can reach the database and the API by name, and on any plain bridge it can
// reach them by address. Checking at startup turns a misconfigured deployment
// into one clear error instead of scans that quietly run without the boundary.
func VerifyJobNetwork(ctx context.Context, client networkInspector, name string) error {
	network, err := client.InspectNetwork(ctx, name)
	if err != nil {
		if IsNotFound(err) {
			return fmt.Errorf(
				"job network %q does not exist; create it with: podman network create --opt isolate=%s %s",
				name, jobNetworkIsolation, name,
			)
		}

		return fmt.Errorf("inspect job network %q: %w", name, err)
	}

	if isolation := network.Options["isolate"]; isolation != jobNetworkIsolation {
		return fmt.Errorf(
			"job network %q has isolate=%q, want %q; recreate it with: podman network create --opt isolate=%s %s",
			name, isolation, jobNetworkIsolation, jobNetworkIsolation, name,
		)
	}

	return nil
}
