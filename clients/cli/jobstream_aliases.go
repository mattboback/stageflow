package main

import (
	"context"
	"io"

	"github.com/mattboback/stageflow/clients/cli/internal/jobstream"
)

func waitJobState(ctx context.Context, c *Client, jobID string, out io.Writer, noStream bool) error {
	return jobstream.WaitJobState(ctx, c, jobID, out, noStream)
}
