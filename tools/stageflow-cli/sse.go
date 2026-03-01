package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type jobStreamUpdate struct {
	Type  string `json:"type"`
	State string `json:"state"`
	Event string `json:"event,omitempty"` // For printing progress
}

// waitJobState waits for the job to reach a terminal state.
// If noStream is true, it polls the API instead of using Server-Sent Events.
func waitJobState(ctx context.Context, c *Client, jobID string, out io.Writer, noStream bool) error {
	if noStream {
		return pollJobState(ctx, c, jobID, out)
	}

	return sseJobState(ctx, c, jobID, out)
}

func pollJobState(ctx context.Context, c *Client, jobID string, out io.Writer) error {
	apiPath := fmt.Sprintf("/api/v1/jobs/%s", jobID)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	checkStatus := func() (bool, error) {
		var status JobStatus
		if err := c.getJSON(ctx, apiPath, &status); err != nil {
			return false, fmt.Errorf("poll failed: %w", err)
		}

		fmt.Fprintf(out, "polling state: %s\n", status.State)

		return isTerminalState(status.State), nil
	}

	isDone, err := checkStatus()
	if err != nil {
		return err
	}

	if isDone {
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			isDone, err = checkStatus()
			if err != nil {
				return err
			}

			if isDone {
				return nil
			}
		}
	}
}

func sseJobState(ctx context.Context, c *Client, jobID string, out io.Writer) error {
	apiPath := fmt.Sprintf("/api/v1/jobs/%s/stream", jobID)

	reqURL, err := c.buildURL(apiPath, nil)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), http.NoBody)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("stream failed: %w", err)
	}

	defer func() {
		_, copyErr := io.Copy(io.Discard, resp.Body)
		if copyErr != nil {
			return
		}

		closeErr := resp.Body.Close()
		if closeErr != nil {
			return
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("stream failed (%d): %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	reader := bufio.NewReaderSize(resp.Body, 1024*1024)

	return consumeSSEUpdates(ctx, reader, out)
}

func readNextSSEEvent(r *bufio.Reader) (eventType, data string, err error) {
	var dataLines []string

	for {
		line, readErr := r.ReadString('\n')
		if readErr != nil {
			return "", "", readErr
		}

		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			return eventType, strings.Join(dataLines, "\n"), nil
		}

		if strings.HasPrefix(line, ":") { // SSE comment / keepalive
			continue
		}

		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}

		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
			continue
		}
	}
}

func isTerminalState(state string) bool {
	return state == jobStateDone || state == jobStateFailed
}

func truncateLog(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}

	return s
}

func consumeSSEUpdates(ctx context.Context, reader *bufio.Reader, out io.Writer) error {
	for {
		eventType, data, readErr := readNextSSEEvent(reader)
		if readErr != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			// Connection drop during SSE could force a polling fallback in a highly robust client,
			// but for this MVP, we return the error.
			return fmt.Errorf("read stream error: %w", readErr)
		}

		done, err := handleSSEEvent(eventType, data, out)
		if err != nil {
			return err
		}

		if done {
			return nil
		}
	}
}

func handleSSEEvent(eventType, data string, out io.Writer) (bool, error) {
	switch eventType {
	case "done":
		return true, nil
	case "update":
		update, ok := parseJobStreamUpdate(data)
		if !ok {
			return false, nil
		}

		if _, err := fmt.Fprintf(out, "update: %s %s\n", update.State, truncateLog(update.Event, 50)); err != nil {
			return false, err
		}

		return isTerminalState(update.State) || isTerminalEvent(update.Type), nil
	default:
		return false, nil
	}
}

func isTerminalEvent(eventType string) bool {
	return eventType == "complete" || eventType == "failed"
}

func parseJobStreamUpdate(data string) (jobStreamUpdate, bool) {
	var update jobStreamUpdate

	if err := json.Unmarshal([]byte(data), &update); err != nil {
		return jobStreamUpdate{}, false
	}

	return update, true
}
