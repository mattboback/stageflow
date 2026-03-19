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
	"github.com/mattboback/stageflow/services/platform-api/internal/status"
)

func jobIDFromJobStreamPath(path string) (string, bool) {
	path = strings.TrimPrefix(path, "/api/v1/jobs/")
	parts := strings.Split(path, "/")

	if len(parts) < 2 || parts[1] != "stream" || parts[0] == "" {
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

func (s *Server) streamJobEvents(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	flusher http.Flusher,
	clientEvents <-chan []byte,
) {
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			logging.Debug(ctx, "SSE stream closed by client disconnect")

			return
		case <-heartbeat.C:
			if err := writeSSEKeepalive(w, flusher); err != nil {
				logging.Debug(ctx, "SSE keepalive write failed; closing stream", "error", err)

				return
			}
		case data, ok := <-clientEvents:
			if !ok {
				logging.Debug(ctx, "SSE hub closed client channel; closing stream")

				return
			}

			shouldClose, ok := s.handleJobStreamUpdate(ctx, w, flusher, data)
			if !ok {
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

	jobID, ok := jobIDFromJobStreamPath(r.URL.Path)
	if !ok {
		http.Error(w, "Invalid path", http.StatusBadRequest)

		return
	}

	ctx := logging.WithJobID(r.Context(), jobID)

	rec, err := s.loadJobRecord(ctx, jobID)
	if err != nil {
		if errors.Is(err, status.ErrJobNotFound) {
			http.Error(w, "Job not found", http.StatusNotFound)
		} else {
			logging.Error(ctx, "Failed to fetch job status for SSE", "error", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}

		return
	}

	setSSEHeaders(w)

	flusher, ok := getFlusher(w)
	if !ok {
		return
	}

	client := s.sseHub.Subscribe(jobID)
	defer s.sseHub.Unsubscribe(client)

	if !s.sendInitialStatus(ctx, w, flusher, rec) {
		return
	}

	if rec.State.IsTerminal() {
		done, marshalErr := json.Marshal(jobStreamUpdate{State: rec.State})
		if marshalErr != nil {
			logging.Error(ctx, "Failed to marshal done payload for SSE", "error", marshalErr)

			return
		}

		if writeErr := writeSSEEvent(w, flusher, "done", done); writeErr != nil {
			return
		}

		return
	}

	s.streamJobEvents(ctx, w, r, flusher, client.Events)
}
