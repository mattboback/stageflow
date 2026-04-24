package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"

	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
)

const (
	projectBaselineStatusAvailable = "available"
	projectBaselineStatusMissing   = "missing"
	projectBaselineStatusCurrent   = "current"
)

type projectScanEnvelope struct {
	Schema   string              `json:"schema"`
	Project  projectScanMeta     `json:"project"`
	Decision projectScanDecision `json:"decision"`
	Report   reportEnvelope      `json:"report"`
	Diff     *diffEnvelope       `json:"diff,omitempty"`
}

type projectScanMeta struct {
	Slug     string              `json:"slug"`
	Baseline projectScanBaseline `json:"baseline"`
}

type projectScanBaseline struct {
	Status         string `json:"status"`
	Message        string `json:"message,omitempty"`
	PromoteCommand string `json:"promoteCommand,omitempty"`
}

type projectScanDecision struct {
	Passed         bool `json:"passed"`
	SeverityFailed bool `json:"severityFailed"`
	Regressed      bool `json:"regressed"`
}

type projectDiffState struct {
	baseline  projectScanBaseline
	diff      *diffEnvelope
	regressed bool
}

func waitForCompletedJobReport(
	ctx context.Context,
	client *Client,
	jobID string,
	progressOut io.Writer,
	noStream bool,
) (JobStatus, report.UnifiedReportV2, error) {
	err := waitJobState(ctx, client, jobID, progressOut, noStream)
	if err != nil {
		return JobStatus{}, report.UnifiedReportV2{}, fmt.Errorf("wait for completion: %w", err)
	}

	status, err := fetchJobStatus(ctx, client, jobID)
	if err != nil {
		return JobStatus{}, report.UnifiedReportV2{}, fmt.Errorf("fetch job status: %w", err)
	}

	if status.State != jobStateDone {
		if status.State == jobStateFailed {
			return JobStatus{}, report.UnifiedReportV2{}, fmt.Errorf("job failed: %s", status.Error)
		}

		return JobStatus{}, report.UnifiedReportV2{}, fmt.Errorf("job finished with non-DONE state: %s", status.State)
	}

	doc, err := fetchReport(ctx, client, jobID)
	if err != nil {
		return JobStatus{}, report.UnifiedReportV2{}, fmt.Errorf("fetch report: %w", err)
	}

	return status, doc, nil
}

func resolveProjectDiffState(
	ctx context.Context,
	client *Client,
	slug, jobID string,
) (projectDiffState, error) {
	diffResult, err := client.FetchJobDiff(ctx, jobID)
	if err == nil {
		d := diffFromResult(diffResult, "", "")

		return projectDiffState{
			baseline: projectScanBaseline{
				Status: projectBaselineStatusAvailable,
			},
			diff:      &d,
			regressed: isDiffRegressed(d),
		}, nil
	}

	if state, matched := interpretProjectDiffError(slug, jobID, err); matched {
		return state, nil
	}

	return projectDiffState{}, exitCodeError{Code: 2, Err: fmt.Errorf("fetch diff: %w", err)}
}

func interpretProjectDiffError(slug, jobID string, err error) (projectDiffState, bool) {
	msg := err.Error()

	switch {
	case strings.Contains(msg, "404"), strings.Contains(msg, "No baseline"):
		return projectDiffState{
			baseline: projectScanBaseline{
				Status:         projectBaselineStatusMissing,
				Message:        fmt.Sprintf("No baseline set for project %q.", slug),
				PromoteCommand: fmt.Sprintf("Promote this scan: stageflow project promote %s --job-id %s", slug, jobID),
			},
		}, true
	case strings.Contains(msg, "Cannot diff against self"):
		return projectDiffState{
			baseline: projectScanBaseline{
				Status:  projectBaselineStatusCurrent,
				Message: "This scan is the current baseline. Run a new scan to see a diff.",
			},
		}, true
	default:
		return projectDiffState{}, false
	}
}

func runScanJob(
	ctx context.Context,
	client *Client,
	req SubmitJobRequest,
	timeout time.Duration,
	progressOut io.Writer,
	noStream bool,
) (JobStatus, report.UnifiedReportV2, error) {
	opCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var resp SubmitJobResponse

	if err := client.PostJSON(opCtx, "/api/v1/jobs/urls", req, &resp); err != nil {
		return JobStatus{}, report.UnifiedReportV2{}, fmt.Errorf("submit job: %w", enhanceSubmitJobError(err, req))
	}

	jobID := resp.JobID
	if jobID == "" {
		return JobStatus{}, report.UnifiedReportV2{}, errors.New("submit job: missing job_id in response")
	}

	fmt.Fprintf(progressOut, "Job submitted: %s\nWaiting for completion...\n", jobID)

	return waitForCompletedJobReport(opCtx, client, jobID, progressOut, noStream)
}

func enhanceSubmitJobError(err error, req SubmitJobRequest) error {
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

func runRemoteProjectScan(
	cmd *cobra.Command,
	root *rootOptions,
	slug string,
	timeout time.Duration,
	noStream bool,
	reportOpts reportCommandOptions,
) error {
	client := NewClient(root.apiURL, root.apiKey, nil)

	opCtx, cancel := context.WithTimeout(cmd.Context(), timeout)
	defer cancel()

	resp, err := client.ProjectScan(opCtx, slug)
	if err != nil {
		return exitCodeError{Code: 2, Err: fmt.Errorf("submit project scan: %w", err)}
	}

	jobID := resp.JobID
	if jobID == "" {
		return exitCodeError{Code: 2, Err: errors.New("submit project scan: missing job_id in response")}
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "Project scan submitted: %s (job %s)\nWaiting for completion...\n", slug, jobID)

	status, doc, err := waitForCompletedJobReport(opCtx, client, jobID, cmd.ErrOrStderr(), noStream)
	if err != nil {
		return exitCodeError{Code: 2, Err: err}
	}

	format, err := root.outputFormat()
	if err != nil {
		return exitCodeError{Code: 2, Err: err}
	}

	if format == outputFormatJSON {
		return runRemoteProjectScanJSON(cmd, root.apiURL, slug, status, doc, client, jobID, reportOpts)
	}

	severityFailed := false

	err = renderUnifiedReport(cmd.OutOrStdout(), root.apiURL, status, doc, reportOpts.renderOptions(format))
	if err != nil {
		var exitErr exitCodeError
		if errors.As(err, &exitErr) && exitErr.Code == 1 {
			severityFailed = true
		} else {
			return wrapRenderError(err)
		}
	}

	regressed, err := renderProjectDiff(opCtx, cmd, client, slug, jobID, format)
	if err != nil {
		return err
	}

	if severityFailed || regressed {
		return exitCodeError{Code: 1}
	}

	return nil
}
