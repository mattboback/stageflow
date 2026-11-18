package podman

import (
	"context"
	"fmt"
	"net/http"
)

// ImageExists reports whether an image is in Podman's local store. Job images
// are built on the host and never pulled, so a missing one fails every job
// that would use it.
func (c *Client) ImageExists(ctx context.Context, name string) (bool, error) {
	resp, err := c.doLibpodRequest(ctx, http.MethodGet, "/images/"+name+"/exists", nil)
	if err != nil {
		return false, fmt.Errorf("check image %s: %w", name, err)
	}

	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusNoContent:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, parseResponse(resp, nil)
	}
}
