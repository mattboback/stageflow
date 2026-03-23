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

type scanTiming struct {
	TotalMs            int64 `json:"total_ms"`
	PageIterationMs    int64 `json:"page_iteration_ms"`
	WriteResultsMs     int64 `json:"write_results_ms"`
	UploadArtifactsMs  int64 `json:"upload_artifacts_ms"`
	PublishCompletedMs int64 `json:"publish_completed_ms"`
	FinalizationMs     int64 `json:"finalization_ms"`
}

type jobStreamUpdate struct {
	Type         string      `json:"type"`
	State        string      `json:"state"`
	Event        string      `json:"event,omitempty"` // For printing progress
	ScannerType  string      `json:"scanner_type,omitempty"`
	PagesScanned *int        `json:"pages_scanned,omitempty"`
	Violations   *int        `json:"violations,omitempty"`
	Timing       *scanTiming `json:"timing,omitempty"`
}

type streamState struct {
	lastState         string
	seenCompleted     map[string]struct{}
	remainingScanners []string
}

func newStreamState() *streamState {
	return &streamState{
		seenCompleted: make(map[string]struct{}),
	}
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

	state := newStreamState()

	isDone, err := checkPolledJobStatus(ctx, c, apiPath, out, state)
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
			isDone, err = checkPolledJobStatus(ctx, c, apiPath, out, state)
			if err != nil {
				return err
			}

			if isDone {
				return nil
			}
		}
	}
}

func checkPolledJobStatus(
	ctx context.Context,
	c *Client,
	apiPath string,
	out io.Writer,
	state *streamState,
) (bool, error) {
	var status JobStatus

	if err := c.getJSON(ctx, apiPath, &status); err != nil {
		return false, fmt.Errorf("poll failed: %w", err)
	}

	return emitStatusSnapshot(out, state, &status)
}

func sseJobState(ctx context.Context, c *Client, jobID string, out io.Writer) error {
	apiPath := fmt.Sprintf("/api/v1/jobs/%s/stream", jobID)

	reqURL, err := c.buildURL(apiPath)
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

func consumeSSEUpdates(ctx context.Context, reader *bufio.Reader, out io.Writer) error {
	state := newStreamState()

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

		done, err := handleSSEEvent(eventType, data, out, state)
		if err != nil {
			return err
		}

		if done {
			return nil
		}
	}
}

func handleSSEEvent(eventType, data string, out io.Writer, state *streamState) (bool, error) {
	switch eventType {
	case "status":
		return handleStatusEvent(data, out, state)
	case "done":
		return true, nil
	case "update":
		return handleUpdateEvent(data, out, state)
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

func parseJobStatus(data string) (JobStatus, bool) {
	var status JobStatus

	if err := json.Unmarshal([]byte(data), &status); err != nil {
		return JobStatus{}, false
	}

	return status, true
}

func handleStatusEvent(data string, out io.Writer, state *streamState) (bool, error) {
	status, ok := parseJobStatus(data)
	if !ok {
		return false, nil
	}

	return emitStatusSnapshot(out, state, &status)
}

func handleUpdateEvent(data string, out io.Writer, state *streamState) (bool, error) {
	update, ok := parseJobStreamUpdate(data)
	if !ok {
		return false, nil
	}

	if err := emitUpdateEvent(out, state, update); err != nil {
		return false, err
	}

	return isTerminalState(update.State) || isTerminalEvent(update.Type), nil
}

func emitStatusSnapshot(out io.Writer, state *streamState, status *JobStatus) (bool, error) {
	if err := emitScannerCompletionsFromStatus(out, state, status); err != nil {
		return false, err
	}

	if err := emitStateTransition(out, state, status.State); err != nil {
		return false, err
	}

	return isTerminalState(status.State), nil
}

func emitUpdateEvent(out io.Writer, state *streamState, update jobStreamUpdate) error {
	if update.Type == "scanner_complete" && update.ScannerType != "" {
		remaining := state.markScannerComplete(update.ScannerType)

		if _, err := fmt.Fprintln(out, formatScannerCompletion(update, remaining)); err != nil {
			return err
		}
	}

	return emitStateTransition(out, state, update.State)
}

func emitStateTransition(out io.Writer, state *streamState, nextState string) error {
	if nextState == "" || nextState == state.lastState {
		return nil
	}

	state.lastState = nextState

	_, err := fmt.Fprintf(out, "state: %s\n", nextState)

	return err
}

func emitScannerCompletionsFromStatus(out io.Writer, state *streamState, status *JobStatus) error {
	if status == nil {
		return nil
	}

	state.remainingScanners = deriveRemainingScanners(
		status.ExpectedScanners,
		status.CompletedScanners,
		status.RemainingScanners,
	)

	for _, scannerType := range status.CompletedScanners {
		if scannerType == "" {
			continue
		}

		if _, seen := state.seenCompleted[scannerType]; seen {
			continue
		}

		state.seenCompleted[scannerType] = struct{}{}

		if _, err := fmt.Fprintln(
			out,
			formatStatusScannerCompletion(scannerType, state.remainingScanners),
		); err != nil {
			return err
		}
	}

	return nil
}

func (s *streamState) markScannerComplete(scannerType string) []string {
	if s == nil || scannerType == "" {
		return nil
	}

	s.seenCompleted[scannerType] = struct{}{}
	if len(s.remainingScanners) == 0 {
		return nil
	}

	next := make([]string, 0, len(s.remainingScanners))
	for _, candidate := range s.remainingScanners {
		if candidate == scannerType || candidate == "" {
			continue
		}

		next = append(next, candidate)
	}

	s.remainingScanners = next

	return cloneStrings(next)
}

func deriveRemainingScanners(expected, completed, provided []string) []string {
	if len(provided) > 0 {
		return cloneStrings(provided)
	}

	if len(expected) == 0 {
		return nil
	}

	completedSet := make(map[string]struct{}, len(completed))
	for _, scannerType := range completed {
		if scannerType == "" {
			continue
		}

		completedSet[scannerType] = struct{}{}
	}

	remaining := make([]string, 0, len(expected))
	for _, scannerType := range expected {
		if scannerType == "" {
			continue
		}

		if _, done := completedSet[scannerType]; done {
			continue
		}

		remaining = append(remaining, scannerType)
	}

	return remaining
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	cloned := make([]string, len(values))
	copy(cloned, values)

	return cloned
}

func formatStatusScannerCompletion(scannerType string, remaining []string) string {
	return formatScannerCompletion(jobStreamUpdate{ScannerType: scannerType}, remaining)
}

func formatScannerCompletion(update jobStreamUpdate, remaining []string) string {
	details := make([]string, 0, 3)
	if update.Timing != nil && update.Timing.TotalMs > 0 {
		details = append(details, formatDurationMs(update.Timing.TotalMs))
	}

	if update.PagesScanned != nil {
		details = append(details, fmt.Sprintf("%d page%s", *update.PagesScanned, pluralize(*update.PagesScanned)))
	}

	if update.Violations != nil {
		details = append(details, fmt.Sprintf("%d issue%s", *update.Violations, pluralize(*update.Violations)))
	}

	message := fmt.Sprintf("scanner: %s complete", update.ScannerType)
	if len(details) > 0 {
		message = fmt.Sprintf("%s (%s)", message, strings.Join(details, ", "))
	}

	switch {
	case len(remaining) == 0:
		return message + "; remaining: none"
	case len(remaining) == 1:
		return fmt.Sprintf("%s; remaining: %s", message, remaining[0])
	default:
		return fmt.Sprintf("%s; remaining: %s", message, strings.Join(remaining, ", "))
	}
}

func formatDurationMs(durationMs int64) string {
	return fmt.Sprintf("%.1fs", float64(durationMs)/1000)
}

func pluralize(value int) string {
	if value == 1 {
		return ""
	}

	return "s"
}
