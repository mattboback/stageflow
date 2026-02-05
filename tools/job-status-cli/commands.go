package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"text/tabwriter"
	"time"
)

type jobsOptions struct {
	state  string
	limit  int
	offset int
}

type jobEventsOptions struct {
	limit       int
	offset      int
	showPayload bool
}

func listJobs(ctx context.Context, client *http.Client, out io.Writer, apiURL string, opts jobsOptions) error {
	query := url.Values{
		"limit":  {strconv.Itoa(opts.limit)},
		"offset": {strconv.Itoa(opts.offset)},
	}
	if opts.state != "" {
		query.Set("state", opts.state)
	}

	requestURL, err := buildAPIURL(apiURL, "/api/v1/jobs", query)
	if err != nil {
		return err
	}

	var response ListJobsResponse
	if decodeErr := decodeOKJSON(ctx, client, requestURL, &response); decodeErr != nil {
		return decodeErr
	}

	if _, writeErr := fmt.Fprintf(
		out, "Jobs (showing %d of %d total)\n\n", len(response.Jobs), response.Total,
	); writeErr != nil {
		return writeErr
	}

	if len(response.Jobs) == 0 {
		if _, writeErr := fmt.Fprintln(out, "No jobs found"); writeErr != nil {
			return writeErr
		}

		return nil
	}

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	if _, writeErr := fmt.Fprintf(
		w,
		"JOB ID\tSTATE\tINPUT TYPE\tCREATED\tCOMPLETED\tERROR\n------\t-----\t----------\t-------\t---------\t-----\n",
	); writeErr != nil {
		return writeErr
	}

	now := time.Now()

	for _, job := range response.Jobs {
		completedStr := "-"
		if job.CompletedAt != nil {
			completedStr = formatDuration(now.Sub(*job.CompletedAt))
		}

		errorStr := ""
		if job.Error != "" {
			errorStr = truncateWithEllipsis(job.Error, 30)
		}

		if _, writeErr := fmt.Fprintf(
			w,
			"%s\t%s\t%s\t%s\t%s\t%s\n",
			abbreviate(job.ID, 8),
			job.State,
			job.InputType,
			formatDuration(now.Sub(job.CreatedAt)),
			completedStr,
			errorStr,
		); writeErr != nil {
			return writeErr
		}
	}

	return w.Flush()
}

func showJobEvents(
	ctx context.Context,
	client *http.Client,
	out io.Writer,
	apiURL, jobID string,
	opts jobEventsOptions,
) error {
	if jobID == "" {
		return errors.New("job id is required")
	}

	opts = normalizeJobEventsOptions(opts)

	requestURL, err := buildJobEventsURL(apiURL, jobID, opts)
	if err != nil {
		return err
	}

	var response ListJobEventsResponse
	if decodeErr := decodeOKJSON(ctx, client, requestURL, &response); decodeErr != nil {
		return decodeErr
	}

	if headingErr := writeJobEventsHeading(out, jobID); headingErr != nil {
		return headingErr
	}

	if len(response.Events) == 0 {
		return writeNoJobEventsMessage(out)
	}

	if tableErr := writeJobEventsTable(out, response.Events); tableErr != nil {
		return tableErr
	}

	if !opts.showPayload {
		return nil
	}

	return writeJobEventsPayloads(out, response.Events)
}

func normalizeJobEventsOptions(opts jobEventsOptions) jobEventsOptions {
	if opts.limit <= 0 {
		opts.limit = 500
	}

	return opts
}

func buildJobEventsURL(apiURL, jobID string, opts jobEventsOptions) (*url.URL, error) {
	query := url.Values{
		"limit":  {strconv.Itoa(opts.limit)},
		"offset": {strconv.Itoa(opts.offset)},
	}

	return buildAPIURL(apiURL, "/api/v1/jobs/"+url.PathEscape(jobID)+"/events", query)
}

func writeJobEventsHeading(out io.Writer, jobID string) error {
	if _, err := fmt.Fprintf(out, "Job Events (%s)\n\n", abbreviate(jobID, 8)); err != nil {
		return err
	}

	return nil
}

func writeNoJobEventsMessage(out io.Writer) error {
	if _, err := fmt.Fprintln(out, "No events found"); err != nil {
		return err
	}

	return nil
}

func writeJobEventsTable(out io.Writer, events []JobEvent) error {
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintf(
		w,
		"WHEN\tEVENT\tSTATUS\tDELIVERIES\tSTREAM_SEQ\tPRODUCER\tERROR\n----\t-----\t------\t----------\t----------\t--------\t-----\n",
	); err != nil {
		return err
	}

	now := time.Now()

	for _, ev := range events {
		when := formatDuration(now.Sub(ev.Timestamp))
		status := stringOrDash(ev.HandlerStatus)
		deliveries := int64OrDash(ev.NATSDeliveries)
		streamSeq := int64OrDash(ev.NATSStreamSeq)
		producer := stringOrDash(ev.Producer)
		errStr := truncateOrEmpty(ev.HandlerError, 60)

		if _, err := fmt.Fprintf(
			w,
			"%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			when,
			ev.Event,
			status,
			deliveries,
			streamSeq,
			producer,
			errStr,
		); err != nil {
			return err
		}
	}

	return w.Flush()
}

func writeJobEventsPayloads(out io.Writer, events []JobEvent) error {
	if _, err := fmt.Fprintln(out, "\nPayloads\n--------"); err != nil {
		return err
	}

	for _, ev := range events {
		if ev.Payload == "" {
			continue
		}

		if _, err := fmt.Fprintf(
			out,
			"\n%s %s\n%s\n",
			ev.Timestamp.Format(time.RFC3339),
			ev.Event,
			ev.Payload,
		); err != nil {
			return err
		}
	}

	return nil
}

func stringOrDash(value string) string {
	if value == "" {
		return "-"
	}

	return value
}

func int64OrDash(value int64) string {
	if value <= 0 {
		return "-"
	}

	return strconv.FormatInt(value, 10)
}

func truncateOrEmpty(value string, maxLen int) string {
	if value == "" {
		return ""
	}

	return truncateWithEllipsis(value, maxLen)
}

func listPods(ctx context.Context, client *http.Client, out io.Writer, apiURL string) error {
	requestURL, err := buildAPIURL(apiURL, "/api/v1/pods", nil)
	if err != nil {
		return err
	}

	var response ListPodsResponse
	if decodeErr := decodeOKJSON(ctx, client, requestURL, &response); decodeErr != nil {
		return decodeErr
	}

	if _, writeErr := fmt.Fprintf(out, "Pods (total: %d)\n\n", response.Total); writeErr != nil {
		return writeErr
	}

	if len(response.Pods) == 0 {
		if _, writeErr := fmt.Fprintln(out, "No pods found"); writeErr != nil {
			return writeErr
		}

		return nil
	}

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	if _, writeErr := fmt.Fprintf(
		w,
		"POD ID\tNAME\tSTATUS\tJOB ID\tJOB STATE\n------\t----\t------\t------\t---------\n",
	); writeErr != nil {
		return writeErr
	}

	for _, pod := range response.Pods {
		if _, writeErr := fmt.Fprintf(
			w,
			"%s\t%s\t%s\t%s\t%s\n",
			truncate(pod.ID, 12),
			pod.Name,
			pod.Status,
			formatJobID(pod.JobID),
			formatJobState(pod.JobState),
		); writeErr != nil {
			return writeErr
		}
	}

	return w.Flush()
}

func showSystemStatus(ctx context.Context, client *http.Client, out io.Writer, apiURL string) error {
	requestURL, err := buildAPIURL(apiURL, "/api/v1/status", nil)
	if err != nil {
		return err
	}

	var response SystemStatusResponse
	if decodeErr := decodeOKJSON(ctx, client, requestURL, &response); decodeErr != nil {
		return decodeErr
	}

	if _, writeErr := fmt.Fprint(out, "System Status\n=============\n\n"); writeErr != nil {
		return writeErr
	}

	if metricsErr := printJobMetrics(out, response.Jobs); metricsErr != nil {
		return metricsErr
	}

	return printPodMetrics(out, response.Pods)
}

func printJobMetrics(out io.Writer, jobs JobsStatus) error {
	if _, err := fmt.Fprintf(
		out,
		"Jobs:\n  Total:     %d\n  Active:    %d\n  Completed: %d\n  Failed:    %d\n",
		jobs.Total,
		jobs.Active,
		jobs.Completed,
		jobs.Failed,
	); err != nil {
		return err
	}

	if len(jobs.ByState) == 0 {
		return nil
	}

	if _, err := fmt.Fprintln(out, "\n  By State:"); err != nil {
		return err
	}

	states := make([]string, 0, len(jobs.ByState))
	for state := range jobs.ByState {
		states = append(states, state)
	}

	sort.Strings(states)

	for _, state := range states {
		count := jobs.ByState[state]
		if count <= 0 {
			continue
		}

		if _, err := fmt.Fprintf(out, "    %-12s %d\n", state+":", count); err != nil {
			return err
		}
	}

	return nil
}

func printPodMetrics(out io.Writer, pods PodsStatus) error {
	if _, err := fmt.Fprintf(out, "\nPods:\n  Total: %d\n", pods.Total); err != nil {
		return err
	}

	if len(pods.ByStatus) == 0 {
		return nil
	}

	if _, err := fmt.Fprintln(out, "\n  By Status:"); err != nil {
		return err
	}

	statuses := make([]string, 0, len(pods.ByStatus))
	for status := range pods.ByStatus {
		statuses = append(statuses, status)
	}

	sort.Strings(statuses)

	for _, status := range statuses {
		count := pods.ByStatus[status]
		if _, err := fmt.Fprintf(out, "    %-12s %d\n", status+":", count); err != nil {
			return err
		}
	}

	return nil
}
