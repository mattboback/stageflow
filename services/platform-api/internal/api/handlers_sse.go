package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mattboback/stageflow/libs/go/logging"
	"github.com/mattboback/stageflow/libs/go/models"
	"github.com/mattboback/stageflow/services/platform-api/internal/jobstatus"
	"github.com/mattboback/stageflow/services/platform-api/internal/status"
)

func jobIDFromJobPath(path, suffix string) (string, bool) {
	path = strings.TrimPrefix(path, "/api/v1/jobs/")
	parts := strings.Split(path, "/")

	if len(parts) < 2 || parts[1] != suffix || parts[0] == "" {
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
	if !isTerminalType && !update.State.IsTerminal() {
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

func statusPayload(snapshot *status.JobRecord) map[string]any {
	return map[string]any{
		"type":  "status",
		"state": snapshot.State,
	}
}

func mapChangeToSSEPayload(change jobstatus.Change) map[string]any {
	snapshot := change.Snapshot
	if snapshot == nil {
		return map[string]any{}
	}

	switch change.Signal.Kind {
	case jobstatus.SignalJobCreated:
		return statusPayload(snapshot)
	case jobstatus.SignalScanPageCompleted:
		payload := map[string]any{
			"type":  "progress",
			"state": snapshot.State,
			"progress": map[string]int{
				"currentPage": snapshot.CurrentPage,
				"totalPages":  snapshot.TotalPages,
			},
		}

		if change.Signal.ScanPageCompleted != nil && change.Signal.ScanPageCompleted.ScannerType != "" {
			payload["scanner_type"] = change.Signal.ScanPageCompleted.ScannerType
		}

		return payload
	case jobstatus.SignalScanCompleted:
		payload := map[string]any{
			"type":          "scanner_complete",
			"state":         snapshot.State,
			"pages_scanned": snapshot.CurrentPage,
			"violations":    snapshot.TotalViolations,
		}

		if change.Signal.ScanCompleted != nil {
			payload["scanner_violations"] = change.Signal.ScanCompleted.Summary.TotalViolations

			if change.Signal.ScanCompleted.ScannerType != "" {
				payload["scanner_type"] = change.Signal.ScanCompleted.ScannerType
			}

			if change.Signal.ScanCompleted.Timing != nil {
				payload["timing"] = map[string]int64{
					"total_ms":             change.Signal.ScanCompleted.Timing.TotalMs,
					"page_iteration_ms":    change.Signal.ScanCompleted.Timing.PageIterationMs,
					"write_results_ms":     change.Signal.ScanCompleted.Timing.WriteResultsMs,
					"upload_artifacts_ms":  change.Signal.ScanCompleted.Timing.UploadArtifactsMs,
					"publish_completed_ms": change.Signal.ScanCompleted.Timing.PublishCompletedMs,
					"finalization_ms":      change.Signal.ScanCompleted.Timing.FinalizationMs,
				}
			}
		}

		return payload
	case jobstatus.SignalExtractionReady:
		payload := map[string]any{
			"type":  "status",
			"state": "READY_TO_SCAN",
		}

		if snapshot.TotalPages > 0 {
			payload["totalPages"] = snapshot.TotalPages
		}

		return payload
	case jobstatus.SignalExtractionFailed:
		return map[string]any{
			"type":          "failed",
			"state":         snapshot.State,
			"error":         snapshot.Error,
			"error_details": snapshot.LastErrorDetails,
			"stage":         snapshot.LastStage,
		}
	case jobstatus.SignalScanFailed, jobstatus.SignalJobFailed:
		return map[string]any{
			"type":  "failed",
			"state": snapshot.State,
			"error": snapshot.Error,
		}
	case jobstatus.SignalJobCompleted:
		return map[string]any{
			"type":  "complete",
			"state": snapshot.State,
		}
	default:
		return statusPayload(snapshot)
	}
}

func getFlusher(w http.ResponseWriter) (http.Flusher, bool) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)

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
		return true
	}

	data, marshalErr := json.Marshal(job)
	if marshalErr != nil {
		logging.Error(ctx, "Failed to marshal initial status for SSE", "error", marshalErr)
		http.Error(w, "Internal error", http.StatusInternalServerError)

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

func (s *Server) watchJobStream(
	ctx context.Context,
	w http.ResponseWriter,
	jobID string,
) (*status.JobRecord, jobstatus.Subscription, bool) {
	rec, sub, err := s.jobStatus.Watch(ctx, jobID, jobstatus.WatchOptions{})
	if err != nil {
		if errors.Is(err, status.ErrJobNotFound) {
			http.Error(w, "Job not found", http.StatusNotFound)
		} else {
			logging.Error(ctx, "Failed to fetch job status for SSE", "error", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}

		return nil, nil, false
	}

	return rec, sub, true
}

func (s *Server) writeTerminalStatus(
	ctx context.Context,
	w http.ResponseWriter,
	flusher http.Flusher,
	rec *status.JobRecord,
) bool {
	if !rec.State.IsTerminal() {
		return false
	}

	done, marshalErr := json.Marshal(jobStreamUpdate{State: rec.State})
	if marshalErr != nil {
		logging.Error(ctx, "Failed to marshal done payload for SSE", "error", marshalErr)

		return true
	}

	if writeErr := writeSSEEvent(w, flusher, "done", done); writeErr != nil {
		return true
	}

	return true
}

func (s *Server) handleJobStreamChange(
	ctx context.Context,
	w http.ResponseWriter,
	flusher http.Flusher,
	change jobstatus.Change,
) (shouldClose, ok bool) {
	payload, marshalErr := json.Marshal(mapChangeToSSEPayload(change))
	if marshalErr != nil {
		logging.Error(ctx, "Failed to marshal SSE update payload", "error", marshalErr)

		return false, false
	}

	return s.handleJobStreamUpdate(ctx, w, flusher, payload)
}

func (s *Server) streamJobUpdates(
	ctx context.Context,
	requestCtx context.Context,
	w http.ResponseWriter,
	flusher http.Flusher,
	sub jobstatus.Subscription,
) {
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-requestCtx.Done():
			logging.Debug(ctx, "SSE stream closed by client disconnect")

			return
		case <-heartbeat.C:
			if keepaliveErr := writeSSEKeepalive(w, flusher); keepaliveErr != nil {
				logging.Debug(ctx, "SSE keepalive write failed; closing stream", "error", keepaliveErr)

				return
			}
		case change, updatesOpen := <-sub.Updates():
			if !updatesOpen {
				logging.Debug(ctx, "status pipeline closed watcher; closing stream")

				return
			}

			shouldClose, streamOK := s.handleJobStreamChange(ctx, w, flusher, change)
			if !streamOK {
				logging.Debug(ctx, "SSE update write failed; closing stream")

				return
			}

			if shouldClose {
				logging.Debug(ctx, "SSE stream closing after terminal update")

				return
			}
		}
	}
}

// handleJobStream provides Server-Sent Events for real-time job status updates.
func (s *Server) handleJobStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

		return
	}

	jobID, ok := jobIDFromJobPath(r.URL.Path, "stream")
	if !ok {
		http.Error(w, "Invalid path", http.StatusBadRequest)

		return
	}

	ctx := logging.WithJobID(r.Context(), jobID)

	rec, sub, ok := s.watchJobStream(ctx, w, jobID)
	if !ok {
		return
	}
	defer sub.Close()

	setSSEHeaders(w)

	flusher, ok := getFlusher(w)
	if !ok {
		return
	}

	if !s.sendInitialStatus(ctx, w, flusher, rec) {
		return
	}

	if s.writeTerminalStatus(ctx, w, flusher, rec) {
		return
	}

	s.streamJobUpdates(ctx, r.Context(), w, flusher, sub)
}
