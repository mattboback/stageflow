package apiclient

import (
	"context"
	"time"
)

// DiscoveryResult keeps the original fields and adds explicit coverage warnings.
type DiscoveryResult struct {
	URLs        []string `json:"urls"`
	Source      string   `json:"source"`
	Truncated   bool     `json:"truncated"`
	Sources     []string `json:"sources"`
	Complete    bool     `json:"complete"`
	StopReasons []string `json:"stopReasons"`
	Warnings    []string `json:"warnings"`
}

func (c *Client) DiscoverPages(ctx context.Context, seed string) (DiscoveryResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 55*time.Second)
	defer cancel()

	var result DiscoveryResult

	err := c.PostJSON(ctx, "/api/v1/discover", map[string]string{"url": seed}, &result)

	return result, err
}
