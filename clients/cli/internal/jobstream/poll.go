package jobstream

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
)

func pollJobState(ctx context.Context, c *apiclient.Client, jobID string, out io.Writer) error {
	apiPath := fmt.Sprintf("/api/v1/jobs/%s", url.PathEscape(jobID))

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	state := newStreamState()
	isDone, err := checkPolledJobStatus(ctx, c, apiPath, out, state)
	if err != nil || isDone {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			isDone, err = checkPolledJobStatus(ctx, c, apiPath, out, state)
			if err != nil || isDone {
				return err
			}
		}
	}
}

func checkPolledJobStatus(
	ctx context.Context,
	c *apiclient.Client,
	apiPath string,
	out io.Writer,
	state *streamState,
) (bool, error) {
	var status apiclient.JobStatus
	if err := c.GetJSON(ctx, apiPath, &status); err != nil {
		return false, fmt.Errorf("poll failed: %w", err)
	}

	return emitStatusSnapshot(out, state, &status)
}
