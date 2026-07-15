package projectscan

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
	"github.com/mattboback/stageflow/clients/cli/internal/diffrender"
	"github.com/mattboback/stageflow/clients/cli/internal/exitcode"
	"github.com/mattboback/stageflow/clients/cli/internal/render"
	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
)

const (
	BaselineStatusAvailable = "available"
	BaselineStatusMissing   = "missing"
	BaselineStatusCurrent   = "current"
)

type Envelope struct {
	Schema   string                `json:"schema"`
	Project  ProjectMeta           `json:"project"`
	Decision Decision              `json:"decision"`
	Report   render.ReportEnvelope `json:"report"`
	Diff     *diffrender.Envelope  `json:"diff,omitempty"`
}

type ProjectMeta struct {
	Slug     string   `json:"slug"`
	Baseline Baseline `json:"baseline"`
}

type Baseline struct {
	Status         string `json:"status"`
	Message        string `json:"message,omitempty"`
	PromoteCommand string `json:"promoteCommand,omitempty"`
}

type Decision struct {
	Passed         bool `json:"passed"`
	SeverityFailed bool `json:"severityFailed"`
	Regressed      bool `json:"regressed"`
}

type Options struct {
	APIBaseURL string
	Slug       string
	JobID      string
	Format     render.Format
	Report     render.Options
	Stdout     io.Writer
	Stderr     io.Writer
}

type diffState struct {
	baseline  Baseline
	diff      *diffrender.Envelope
	regressed bool
}

func WriteResult(
	ctx context.Context,
	client *apiclient.Client,
	status apiclient.JobStatus,
	doc report.UnifiedReportV2,
	opts Options,
) error {
	if opts.Format == render.FormatJSON {
		return writeJSON(ctx, client, status, doc, opts)
	}

	severityFailed := false

	err := render.UnifiedReport(opts.Stdout, opts.APIBaseURL, status, doc, opts.Report)
	if err != nil {
		var exitErr exitcode.Error
		if errors.As(err, &exitErr) && exitErr.Code == 1 {
			severityFailed = true
		} else {
			return render.WrapError(err)
		}
	}

	regressed, err := writeDiff(ctx, client, opts)
	if err != nil {
		return err
	}

	if severityFailed || regressed {
		return exitcode.Error{Code: 1}
	}

	return nil
}

func writeJSON(
	ctx context.Context,
	client *apiclient.Client,
	status apiclient.JobStatus,
	doc report.UnifiedReportV2,
	opts Options,
) error {
	selectedIssues, filters, err := render.ValidatedIssueSelection(doc.Issues, opts.Report)
	if err != nil {
		return exitcode.Error{Code: 2, Err: err}
	}

	filteredDoc := doc

	filteredDoc.Issues = selectedIssues
	if opts.Report.SummaryOnly {
		filteredDoc.Issues = nil
	}

	reportPayload, err := render.BuildReportEnvelope(opts.APIBaseURL, status, filteredDoc, filters)
	if err != nil {
		return exitcode.Error{Code: 2, Err: err}
	}

	severityFailed, err := render.ShouldFailForSeverity(selectedIssues, opts.Report.FailSeverity)
	if err != nil {
		return exitcode.Error{Code: 2, Err: err}
	}

	state, err := resolveDiffState(ctx, client, opts.Slug, opts.JobID)
	if err != nil {
		return err
	}

	payload := Envelope{
		Schema:  "stageflow-cli/project-scan@v1",
		Project: ProjectMeta{Slug: opts.Slug, Baseline: state.baseline},
		Decision: Decision{
			Passed:         !severityFailed && !state.regressed,
			SeverityFailed: severityFailed,
			Regressed:      state.regressed,
		},
		Report: reportPayload,
		Diff:   state.diff,
	}

	encoder := json.NewEncoder(opts.Stdout)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)

	if encodeErr := encoder.Encode(payload); encodeErr != nil {
		return exitcode.Error{Code: 2, Err: encodeErr}
	}

	if severityFailed || state.regressed {
		return exitcode.Error{Code: 1}
	}

	return nil
}

func writeDiff(ctx context.Context, client *apiclient.Client, opts Options) (bool, error) {
	state, err := resolveDiffState(ctx, client, opts.Slug, opts.JobID)
	if err != nil {
		return false, err
	}

	fmt.Fprintln(opts.Stdout)

	if state.diff == nil {
		fmt.Fprintln(opts.Stderr, state.baseline.Message)

		if state.baseline.PromoteCommand != "" {
			fmt.Fprintln(opts.Stderr, state.baseline.PromoteCommand)
		}

		return false, nil
	}

	format, err := diffFormat(opts.Format)
	if err != nil {
		return false, exitcode.Error{Code: 2, Err: err}
	}

	if renderErr := diffrender.Render(opts.Stdout, *state.diff, format); renderErr != nil {
		return false, exitcode.Error{Code: 2, Err: renderErr}
	}

	return state.regressed, nil
}

func resolveDiffState(
	ctx context.Context,
	client *apiclient.Client,
	slug, jobID string,
) (diffState, error) {
	diffResult, err := client.FetchJobDiff(ctx, jobID)
	if err == nil {
		d := diffrender.FromResult(diffResult, "", "")

		return diffState{
			baseline:  Baseline{Status: BaselineStatusAvailable},
			diff:      &d,
			regressed: diffrender.IsRegressed(d),
		}, nil
	}

	if state, matched := interpretDiffError(slug, jobID, err); matched {
		return state, nil
	}

	return diffState{}, exitcode.Error{Code: 2, Err: fmt.Errorf("fetch diff: %w", err)}
}

func interpretDiffError(slug, jobID string, err error) (diffState, bool) {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "404"), strings.Contains(msg, "No baseline"):
		return diffState{baseline: Baseline{
			Status:         BaselineStatusMissing,
			Message:        fmt.Sprintf("No baseline set for project %q.", slug),
			PromoteCommand: fmt.Sprintf("Promote this scan: stageflow project promote %s --job-id %s", slug, jobID),
		}}, true
	case strings.Contains(msg, "Cannot diff against self"):
		return diffState{baseline: Baseline{
			Status:  BaselineStatusCurrent,
			Message: "This scan is the current baseline. Run a new scan to see a diff.",
		}}, true
	default:
		return diffState{}, false
	}
}

func diffFormat(format render.Format) (diffrender.Format, error) {
	switch format {
	case render.FormatJSON:
		return diffrender.FormatJSON, nil
	case render.FormatText:
		return diffrender.FormatText, nil
	case render.FormatMarkdown:
		return diffrender.FormatMarkdown, nil
	default:
		return 0, fmt.Errorf("unsupported output format %q", format)
	}
}
