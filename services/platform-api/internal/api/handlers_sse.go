package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	status "github.com/mattboback/stageflow/internal/jobstore"
	"github.com/mattboback/stageflow/libs/go/httputil"
	"github.com/mattboback/stageflow/libs/go/logging"
	"github.com/mattboback/stageflow/libs/go/models"
)

func jobIDFromJobPath(path, suffix string) (string, bool) {
	path = strings.TrimPrefix(path, "/api/v1/jobs/")
	parts := strings.Split(path, "/")

	if len(parts) != 2 || parts[1] != suffix || parts[0] == "" {
		return "", false
	}

	if !validJobID(parts[0]) {
		return "", false
	}

	return parts[0], true
}

type jobStreamUpdate struct {
	Type  string          `json:"type,omitempty"`
	State models.JobState `json:"state"`
}

func writeSSEEvent(w http.ResponseWriter, flusher http.Flusher, event string, data []byte) error {
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data); err != nil {
		return err
	}

	flusher.Flush()

	return nil
}

func writeSSEKeepalive(w http.ResponseWriter, flusher http.Flusher) error {
	if _, err := fmt.Fprint(w, ":keepalive\n\n"); err != nil {
		return err
	}

	flusher.Flush()

	return nil
}

func terminalDonePayloadFromUpdate(data []byte) (jobStreamUpdate, bool, error) {
	var update jobStreamUpdate
	if err := json.Unmarshal(data, &update); err != nil {
		return jobStreamUpdate{}, false, fmt.Errorf("invalid update payload: %w", err)
	}

	isTerminalType := update.Type == "complete" || update.Type == "failed"
	if !isTerminalType {
		return jobStreamUpdate{}, false, nil
	}

	if update.State == "" {
		switch update.Type {
		case "complete":
			update.State = models.JobStateDone
		case "failed":
			update.State = models.JobStateFailed
		}
	}

	if !update.State.IsTerminal() {
		return jobStreamUpdate{}, false, fmt.Errorf(
			"terminal update has non-terminal state %q",
			update.State,
		)
	}

	return update, true, nil
}

func setSSEHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering
}

func getFlusher(w http.ResponseWriter) (http.Flusher, bool) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		httputil.RespondError(w, http.StatusInternalServerError, "Streaming not supported")

		return nil, false
	}

	return flusher, true
}

func (s *Server) sendInitialStatus(
	ctx context.Context,
	w http.ResponseWriter,
	flusher http.Flusher,
	rec *status.JobRecord,
) bool {
	job, buildErr := s.buildJobStatusResponse(ctx, rec)
	if buildErr != nil {
		logging.Error(ctx, "Failed to build initial status for SSE", "error", buildErr)

		data, marshalErr := json.Marshal(map[string]string{
			"error": "failed to build initial job status",
		})
		if marshalErr != nil {
			logging.Error(ctx, "Failed to marshal initial SSE error", "error", marshalErr)

			return false
		}

		if err := writeSSEEvent(w, flusher, "error", data); err != nil {
			return false
		}

		return false
	}

	data, marshalErr := json.Marshal(job)
	if marshalErr != nil {
		logging.Error(ctx, "Failed to marshal initial status for SSE", "error", marshalErr)
		httputil.RespondError(w, http.StatusInternalServerError, "Internal error")

		return false
	}

	if err := writeSSEEvent(w, flusher, "status", data); err != nil {
		return false
	}

	return true
}

func (s *Server) handleJobStreamUpdate(
	ctx context.Context,
	w http.ResponseWriter,
	flusher http.Flusher,
	data []byte,
) (shouldClose, ok bool) {
	if err := writeSSEEvent(w, flusher, "update", data); err != nil {
		return false, false
	}

	donePayload, isTerminal, parseErr := terminalDonePayloadFromUpdate(data)
	if parseErr != nil {
		logging.Warn(
			ctx,
			"Ignoring invalid terminal SSE update payload",
			"error",
			parseErr,
			"payload",
			string(data),
		)

		return false, true
	}

	if !isTerminal {
		return false, true
	}

	done, marshalErr := json.Marshal(donePayload)
	if marshalErr != nil {
		logging.Error(ctx, "Failed to marshal done payload for SSE", "error", marshalErr)

		return false, false
	}

	if err := writeSSEEvent(w, flusher, "done", done); err != nil {
		return false, false
	}

	return true, true
}

// snapshotUpdates emits scanner outcomes before the terminal job event, even
// when completion and DONE become visible in the same database read.
func snapshotUpdates(previous, next *status.JobRecord, readySent *bool) []map[string]any {
	var updates []map[string]any
	if !*readySent && next.TotalPages > 0 && next.State != models.JobStatePending &&
		next.State != models.JobStateExtracting {
		updates = append(
			updates,
			map[string]any{"type": "status", "state": "READY_TO_SCAN", "totalPages": next.TotalPages},
		)
		*readySent = true
	}

	if previous.State != next.State && !next.State.IsTerminal() {
		updates = append(updates, map[string]any{"type": "status", "state": next.State})
	}

	names := append([]string(nil), next.ExpectedScanners...)
	sort.Strings(names)

	for _, name := range names {
		scanner := next.ScannerStatuses[name]

		old := previous.ScannerStatuses[name]
		if scanner.State == "completed" || scanner.State == "failed" {
			if old.State == scanner.State {
				continue
			}

			payload := scannerCompletePayload(name, scanner, next)
			updates = append(updates, payload)
		} else if scanner.PagesScanned != old.PagesScanned {
			updates = append(updates, map[string]any{
				"type":         "progress",
				"state":        next.State,
				"scanner_type": name,
				"progress":     map[string]int{"currentPage": next.CurrentPage, "totalPages": next.TotalPages},
				"scanner_progress": map[string]int{
					"currentPage": scanner.PagesScanned,
					"totalPages":  scanner.TotalPages,
				},
			})
		}
	}

	if next.State.IsTerminal() && !previous.State.IsTerminal() {
		updates = append(updates, terminalSnapshotPayload(next))
	}

	return updates
}

func (s *Server) handleJobStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.RespondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	id, ok := jobIDFromJobPath(r.URL.Path, "stream")
	if !ok {
		httputil.RespondError(w, http.StatusBadRequest, "Invalid path")
		return
	}

	if s.respondIfJobDeleted(w, r, id) {
		return
	}

	ctx := logging.WithJobID(r.Context(), id)

	rec, err := s.statusReader.GetJob(ctx, id)
	if err != nil {
		if errors.Is(err, status.ErrJobNotFound) {
			httputil.RespondNotFound(w, "Job")
		} else {
			httputil.RespondError(w, http.StatusInternalServerError, "Failed to fetch job status")
		}

		return
	}

	setSSEHeaders(w)

	flusher, ok := getFlusher(w)
	if !ok {
		return
	}

	if !s.sendInitialStatus(ctx, w, flusher, rec) {
		return
	}

	if rec.State.IsTerminal() {
		data, marshalErr := json.Marshal(jobStreamUpdate{State: rec.State})
		if marshalErr != nil {
			return
		}

		if writeErr := writeSSEEvent(w, flusher, "done", data); writeErr != nil {
			return
		}

		return
	}

	s.streamSnapshotUpdates(ctx, w, flusher, id, rec)
}

func scannerCompletePayload(name string, scanner models.ScannerStatus, snapshot *status.JobRecord) map[string]any {
	payload := map[string]any{"type": "scanner_complete", "state": snapshot.State,
		"scanner_type": name, "pages_scanned": scanner.PagesScanned, "violations": snapshot.TotalViolations,
		"scanner_violations": scanner.Violations, "timing": map[string]int64{"total_ms": scanner.TotalMs}}
	if scanner.State == "failed" {
		payload["failed"] = true
		payload["error"] = scanner.Error
	}

	return payload
}

func terminalSnapshotPayload(snapshot *status.JobRecord) map[string]any {
	payload := map[string]any{"type": "complete", "state": snapshot.State}
	if snapshot.State != models.JobStateFailed {
		return payload
	}

	payload["type"], payload["error"] = "failed", snapshot.Error
	if snapshot.LastErrorDetails != "" {
		payload["error_details"] = snapshot.LastErrorDetails
	}

	if snapshot.LastStage != "" {
		payload["stage"] = snapshot.LastStage
	}

	return payload
}

func (s *Server) sendSnapshotUpdates(
	ctx context.Context,
	w http.ResponseWriter,
	flusher http.Flusher,
	updates []map[string]any,
) bool {
	for _, payload := range updates {
		data, marshalErr := json.Marshal(payload)
		if marshalErr != nil {
			return false
		}

		closeStream, streamOK := s.handleJobStreamUpdate(ctx, w, flusher, data)
		if closeStream || !streamOK {
			return false
		}
	}

	return true
}

func (s *Server) streamSnapshotUpdates(
	ctx context.Context,
	w http.ResponseWriter,
	flusher http.Flusher,
	id string,
	rec *status.JobRecord,
) {
	readySent := false

	empty := &status.JobRecord{State: rec.State, ScannerStatuses: rec.ScannerStatuses}
	if !s.sendSnapshotUpdates(ctx, w, flusher, snapshotUpdates(empty, rec, &readySent)) {
		return
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeat.C:
			if writeSSEKeepalive(w, flusher) != nil {
				return
			}
		case <-ticker.C:
			next, readErr := s.statusReader.GetJob(ctx, id)
			if readErr != nil {
				logging.Warn(ctx, "Failed to refresh SSE snapshot", "error", readErr)
				return
			}

			if !s.sendSnapshotUpdates(ctx, w, flusher, snapshotUpdates(rec, next, &readySent)) {
				return
			}

			rec = next
		}
	}
}
