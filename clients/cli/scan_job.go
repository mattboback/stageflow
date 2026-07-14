package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
	"github.com/mattboback/stageflow/clients/cli/internal/exitcode"
	"github.com/mattboback/stageflow/clients/cli/internal/render"
	"github.com/mattboback/stageflow/clients/cli/internal/scanflow"
)

const (
	projectBaselineStatusAvailable = "available"
	projectBaselineStatusMissing   = "missing"
	projectBaselineStatusCurrent   = "current"
)

type projectScanEnvelope struct {
	Schema   string                `json:"schema"`
	Project  projectScanMeta       `json:"project"`
	Decision projectScanDecision   `json:"decision"`
	Report   render.ReportEnvelope `json:"report"`
	Diff     *diffEnvelope         `json:"diff,omitempty"`
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

func resolveProjectDiffState(
	ctx context.Context,
	client *apiclient.Client,
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

	return projectDiffState{}, exitcode.Error{Code: 2, Err: fmt.Errorf("fetch diff: %w", err)}
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

func runRemoteProjectScan(
	cmd *cobra.Command,
	root *rootOptions,
	slug string,
	timeout time.Duration,
	noStream bool,
	reportOpts render.ReportFlags,
) error {
	client := apiclient.NewClient(root.apiURL, root.apiKey, nil)

	opCtx, cancel := context.WithTimeout(cmd.Context(), timeout)
	defer cancel()

	resp, err := client.ProjectScan(opCtx, slug)
	if err != nil {
		return exitcode.Error{Code: 2, Err: fmt.Errorf("submit project scan: %w", err)}
	}

	jobID, err := scanflow.RequireJobID(resp)
	if err != nil {
		return exitcode.Error{Code: 2, Err: fmt.Errorf("submit project scan: %w", err)}
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "Project scan submitted: %s (job %s)\nWaiting for completion...\n", slug, jobID)

	result, err := scanflow.WaitForReport(
		opCtx,
		client,
		jobID,
		scanflow.WaitOptions{Progress: cmd.ErrOrStderr(), NoStream: noStream},
	)
	if err != nil {
		return exitcode.Error{Code: 2, Err: err}
	}

	format, err := root.renderFormat()
	if err != nil {
		return exitcode.Error{Code: 2, Err: err}
	}

	if format == render.FormatJSON {
		return runRemoteProjectScanJSON(
			cmd, root.apiURL, slug, result.Status, result.Report, client, jobID, reportOpts,
		)
	}

	severityFailed := false

	err = render.UnifiedReport(
		cmd.OutOrStdout(), root.apiURL, result.Status, result.Report, reportOpts.RenderOptions(format),
	)
	if err != nil {
		var exitErr exitcode.Error
		if errors.As(err, &exitErr) && exitErr.Code == 1 {
			severityFailed = true
		} else {
			return render.WrapError(err)
		}
	}

	regressed, err := renderProjectDiff(opCtx, cmd, client, slug, jobID, format)
	if err != nil {
		return err
	}

	if severityFailed || regressed {
		return exitcode.Error{Code: 1}
	}

	return nil
}
