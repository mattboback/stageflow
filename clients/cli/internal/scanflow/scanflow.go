package scanflow

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
	"github.com/mattboback/stageflow/clients/cli/internal/jobstream"
	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
)

type Result struct {
	Status apiclient.JobStatus
	Report report.UnifiedReportV2
}

type WaitOptions struct {
	Progress io.Writer
	NoStream bool
}

func SubmitURLsAndWait(
	ctx context.Context,
	client *apiclient.Client,
	req apiclient.SubmitJobRequest,
	timeout time.Duration,
	opts WaitOptions,
) (Result, error) {
	opCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var resp apiclient.SubmitJobResponse
	if err := client.PostJSON(opCtx, "/api/v1/jobs/urls", req, &resp); err != nil {
		return Result{}, fmt.Errorf("submit job: %w", enhanceSubmitJobError(err, req))
	}

	jobID, err := RequireJobID(resp)
	if err != nil {
		return Result{}, fmt.Errorf("submit job: %w", err)
	}

	fmt.Fprintf(opts.Progress, "Job submitted: %s\nWaiting for completion...\n", jobID)

	return WaitForReport(opCtx, client, jobID, opts)
}

func RequireJobID(resp apiclient.SubmitJobResponse) (string, error) {
	if resp.JobID == "" {
		return "", errors.New("missing job_id in response")
	}

	return resp.JobID, nil
}

func WaitForReport(
	ctx context.Context,
	client *apiclient.Client,
	jobID string,
	opts WaitOptions,
) (Result, error) {
	if err := jobstream.WaitJobState(ctx, client, jobID, opts.Progress, opts.NoStream); err != nil {
		return Result{}, fmt.Errorf("wait for completion: %w", err)
	}

	status, err := client.FetchJobStatus(ctx, jobID)
	if err != nil {
		return Result{}, fmt.Errorf("fetch job status: %w", err)
	}

	if status.State != apiclient.JobStateDone {
		if status.State == apiclient.JobStateFailed {
			return Result{}, fmt.Errorf("job failed: %s", status.Error)
		}

		return Result{}, fmt.Errorf("job finished with non-DONE state: %s", status.State)
	}

	doc, err := client.FetchReport(ctx, jobID)
	if err != nil {
		return Result{}, fmt.Errorf("fetch report: %w", err)
	}

	return Result{Status: status, Report: doc}, nil
}

func enhanceSubmitJobError(err error, req apiclient.SubmitJobRequest) error {
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "allow_private_targets") &&
		!strings.Contains(msg, "disallowed address") &&
		!strings.Contains(msg, "disallowed") {
		return err
	}

	if req.AllowPrivateTargets {
		return fmt.Errorf(
			"%w; this API instance may not permit private target scans. For local development, run `just dev up local` and `just dev init local`",
			err,
		)
	}

	return fmt.Errorf(
		"%w; local/private targets require allow_private_targets. Re-run with --allow-private-targets or use a localhost/private URL so the CLI can auto-enable it",
		err,
	)
}
