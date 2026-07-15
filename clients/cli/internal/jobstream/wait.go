package jobstream

import (
	"context"
	"fmt"
	"io"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
)

// WaitJobState waits for the job to reach a terminal state. If noStream is
// true, it polls the API instead of using Server-Sent Events.
func WaitJobState(ctx context.Context, c *apiclient.Client, jobID string, out io.Writer, noStream bool) error {
	if noStream {
		return pollJobState(ctx, c, jobID, out)
	}

	err := sseJobState(ctx, c, jobID, out)
	if err == nil || ctx.Err() != nil {
		return err
	}

	fmt.Fprintln(out, "stream lost, falling back to polling...")

	return pollJobState(ctx, c, jobID, out)
}
