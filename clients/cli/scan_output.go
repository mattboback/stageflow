package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
)

func runRemoteProjectScanJSON(
	cmd *cobra.Command,
	apiBaseURL, slug string,
	status JobStatus,
	doc report.UnifiedReportV2,
	client *Client,
	jobID string,
	reportOpts reportCommandOptions,
) error {
	renderOpts := reportOpts.renderOptions(outputFormatJSON)

	selectedIssues, filters, err := validatedIssueSelection(doc.Issues, renderOpts)
	if err != nil {
		return exitCodeError{Code: 2, Err: err}
	}

	filteredDoc := doc
	filteredDoc.Issues = selectedIssues

	if renderOpts.SummaryOnly {
		filteredDoc.Issues = nil
	}

	reportPayload, err := buildReportEnvelope(apiBaseURL, status, filteredDoc, filters)
	if err != nil {
		return exitCodeError{Code: 2, Err: err}
	}

	severityFailed, err := shouldFailForSeverity(selectedIssues, renderOpts.FailSeverity)
	if err != nil {
		return exitCodeError{Code: 2, Err: err}
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
		return exitCodeError{Code: 2, Err: encodeErr}
	}

	if severityFailed || diffState.regressed {
		return exitCodeError{Code: 1}
	}

	return nil
}

func renderProjectDiff(
	ctx context.Context,
	cmd *cobra.Command,
	client *Client,
	slug, jobID string,
	format outputFormat,
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
		return false, exitCodeError{Code: 2, Err: err}
	}

	return diffState.regressed, nil
}
