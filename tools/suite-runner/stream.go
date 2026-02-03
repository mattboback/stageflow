package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type jobStreamUpdate struct {
	Type  string `json:"type"`
	State string `json:"state"`
}

func waitForTerminalWithRetry(ctx context.Context, client *http.Client, apiURL, jobID string, retryDelay time.Duration) error {
	for {
		err := waitForJobTerminalEvent(ctx, client, apiURL, jobID)
		if err == nil {
			return nil
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}

		if retryDelay <= 0 {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(retryDelay):
			continue
		}
	}
}

func outcomeFromTerminalStatus(ctx context.Context, client *http.Client, apiURL, domain, jobID string, th Thresholds) (jobOutcome, error) {
	status, err := fetchStatus(ctx, client, apiURL, jobID)
	if err != nil {
		return jobOutcome{}, err
	}

	switch status.State {
	case "DONE":
		summary, err := fetchResults(ctx, client, status.Artifacts)
		if err != nil {
			return jobOutcome{}, err
		}

		outcome := jobOutcome{
			Domain:          domain,
			JobID:           jobID,
			State:           status.State,
			TotalViolations: summary.Summary.TotalViolations,
			Critical:        summary.Summary.ByImpact["critical"],
			Serious:         summary.Summary.ByImpact["serious"],
		}
		outcome.Passed = evaluate(outcome, th)

		return outcome, nil
	case "FAILED":
		return jobOutcome{
			Domain: domain,
			JobID:  jobID,
			State:  status.State,
			Error:  status.Error,
			Passed: false,
		}, nil
	default:
		return jobOutcome{}, fmt.Errorf("stream ended but job not terminal (state=%s)", status.State)
	}
}

func streamJob(
	ctx context.Context,
	client *http.Client,
	apiURL string,
	domain string,
	jobID string,
	retryDelay time.Duration,
	th Thresholds,
) jobOutcome {
	streamClient := &http.Client{}
	if err := waitForTerminalWithRetry(ctx, streamClient, apiURL, jobID, retryDelay); err != nil {
		if ctx.Err() != nil {
			return jobOutcome{
				Domain: domain,
				JobID:  jobID,
				State:  "TIMEOUT",
				Error:  err.Error(),
				Passed: false,
			}
		}

		return jobOutcome{
			Domain: domain,
			JobID:  jobID,
			State:  "FAILED",
			Error:  err.Error(),
			Passed: false,
		}
	}

	outcome, err := outcomeFromTerminalStatus(ctx, client, apiURL, domain, jobID, th)
	if err != nil {
		return jobOutcome{
			Domain: domain,
			JobID:  jobID,
			State:  "FAILED",
			Error:  err.Error(),
			Passed: false,
		}
	}

	return outcome
}

func waitForJobTerminalEvent(ctx context.Context, client *http.Client, apiURL, jobID string) error {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/api/v1/jobs/%s/stream", apiURL, jobID),
		http.NoBody,
	)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "text/event-stream")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		_, copyErr := io.Copy(io.Discard, resp.Body)
		closeErr := resp.Body.Close()

		if copyErr != nil {
			_ = copyErr
		}

		if closeErr != nil {
			_ = closeErr
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("stream failed (%d): %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	reader := bufio.NewReaderSize(resp.Body, 1024*1024)
	for {
		eventType, data, err := readNextSSEEvent(reader)
		if err != nil {
			return err
		}

		switch eventType {
		case "done":
			return nil
		case "update":
			var update jobStreamUpdate
			if err := json.Unmarshal([]byte(data), &update); err != nil {
				continue
			}

			if update.Type == "complete" || update.Type == "failed" || update.State == "DONE" || update.State == "FAILED" {
				return nil
			}
		}
	}
}
