package jobstream

import (
	"context"
	"fmt"
	"io"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
)

// WaitJobState waits for a terminal state using SSE, falling back to polling
// when the stream fails.
func WaitJobState(ctx context.Context, c *apiclient.Client, jobID string, out io.Writer) error {
	err := sseJobState(ctx, c, jobID, out)
	if err == nil || ctx.Err() != nil {
		return err
	}

	fmt.Fprintln(out, "stream lost, falling back to polling...")

	return pollJobState(ctx, c, jobID, out)
}
