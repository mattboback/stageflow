package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
	"github.com/mattboback/stageflow/clients/cli/internal/exitcode"
	"github.com/mattboback/stageflow/clients/cli/internal/render"
	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
)

func runRemoteProjectScanJSON(
	cmd *cobra.Command,
	apiBaseURL, slug string,
	status apiclient.JobStatus,
	doc report.UnifiedReportV2,
	client *apiclient.Client,
	jobID string,
	reportOpts render.ReportFlags,
) error {
	renderOpts := reportOpts.RenderOptions(render.FormatJSON)

	selectedIssues, filters, err := render.ValidatedIssueSelection(doc.Issues, renderOpts)
	if err != nil {
		return exitcode.Error{Code: 2, Err: err}
	}

	filteredDoc := doc
	filteredDoc.Issues = selectedIssues

	if renderOpts.SummaryOnly {
		filteredDoc.Issues = nil
	}

	reportPayload, err := render.BuildReportEnvelope(apiBaseURL, status, filteredDoc, filters)
	if err != nil {
		return exitcode.Error{Code: 2, Err: err}
	}

	severityFailed, err := render.ShouldFailForSeverity(selectedIssues, renderOpts.FailSeverity)
	if err != nil {
		return exitcode.Error{Code: 2, Err: err}
	}

	diffState, err := resolveProjectDiffState(cmd.Context(), client, slug, jobID)
	if err != nil {
		return err
	}

	payload := projectScanEnvelope{
		Schema: "stageflow-cli/project-scan@v1",
		Project: projectScanMeta{
			Slug:     slug,
			Baseline: diffState.baseline,
		},
		Decision: projectScanDecision{
			Passed:         !severityFailed && !diffState.regressed,
			SeverityFailed: severityFailed,
			Regressed:      diffState.regressed,
		},
		Report: reportPayload,
		Diff:   diffState.diff,
	}

	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)

	if encodeErr := encoder.Encode(payload); encodeErr != nil {
		return exitcode.Error{Code: 2, Err: encodeErr}
	}

	if severityFailed || diffState.regressed {
		return exitcode.Error{Code: 1}
	}

	return nil
}

func renderProjectDiff(
	ctx context.Context,
	cmd *cobra.Command,
	client *apiclient.Client,
	slug, jobID string,
	format render.Format,
) (bool, error) {
	diffState, err := resolveProjectDiffState(ctx, client, slug, jobID)
	if err != nil {
		return false, err
	}

	fmt.Fprintln(cmd.OutOrStdout())

	if diffState.diff == nil {
		fmt.Fprintln(cmd.ErrOrStderr(), diffState.baseline.Message)

		if diffState.baseline.PromoteCommand != "" {
			fmt.Fprintln(cmd.ErrOrStderr(), diffState.baseline.PromoteCommand)
		}

		return false, nil
	}

	err = renderDiff(cmd.OutOrStdout(), *diffState.diff, format)
	if err != nil {
		return false, exitcode.Error{Code: 2, Err: err}
	}

	return diffState.regressed, nil
}
