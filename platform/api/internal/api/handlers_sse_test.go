package api

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mattboback/stageflow/packages/shared-go/events"
	"github.com/mattboback/stageflow/packages/shared-go/models"
)

type sseEvent struct {
	Event string
	Data  string
	Err   error
}

const sseTestJobID = "job-123"

type pipeResponseWriter struct {
	header http.Header
	pw     *io.PipeWriter
	status int
}

func newPipeResponseWriter(pw *io.PipeWriter) *pipeResponseWriter {
	return &pipeResponseWriter{
		header: make(http.Header),
		pw:     pw,
		status: http.StatusOK,
	}
}

func (w *pipeResponseWriter) Header() http.Header {
	return w.header
}

func (w *pipeResponseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
}

func (w *pipeResponseWriter) Write(p []byte) (int, error) {
	return w.pw.Write(p)
}

func (w *pipeResponseWriter) Flush() {}

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

		switch {
		case strings.HasPrefix(line, ":"):
			// SSE comment / keepalive
		case strings.HasPrefix(line, "event:"):
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
}

func TestHandleJobStream_SendsDoneAndClosesOnTerminalUpdate(t *testing.T) {
	server, _, store, _ := newTestServer(t)

	if err := store.HandleJobCreated(context.Background(), &events.JobCreatedPayload{
		JobID:     sseTestJobID,
		InputType: events.InputTypeURLs,
		URLs:      []string{"https://example.com"},
		Config: models.JobConfig{
			Modules: []string{scannerTypeAxe},
		},
	}); err != nil {
		t.Fatalf("create job: %v", err)
	}

	eventsCh, done := startJobStream(t, server, sseTestJobID)

	waitForSSEEvent(t, eventsCh, "status", 2*time.Second)

	server.sseHub.Broadcast(sseTestJobID, map[string]any{
		"type":  "complete",
		"state": "DONE",
	})

	waitForSSEEvent(t, eventsCh, "update", 2*time.Second)
	waitForSSEEvent(t, eventsCh, "done", 2*time.Second)

	assertStreamClosed(t, eventsCh, 2*time.Second)
	assertHandlerDone(t, done, 2*time.Second)
}

func startJobStream(t *testing.T, server *Server, jobID string) (eventsCh chan sseEvent, done chan struct{}) {
	t.Helper()

	pr, pw := io.Pipe()

	t.Cleanup(func() {
		_ = pr.Close()
	})

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/v1/jobs/"+jobID+"/stream", http.NoBody)

	done = make(chan struct{})

	go func() {
		defer close(done)

		w := newPipeResponseWriter(pw)
		server.handleJobStream(w, req)

		_ = pw.Close()
	}()

	reader := bufio.NewReader(pr)
	eventsCh = make(chan sseEvent, 16)

	go func() {
		defer close(eventsCh)

		for {
			evType, data, err := readNextSSEEvent(reader)
			if err != nil {
				eventsCh <- sseEvent{Err: err}

				return
			}

			eventsCh <- sseEvent{Event: evType, Data: data}
		}
	}()

	return eventsCh, done
}

func waitForSSEEvent(t *testing.T, eventsCh <-chan sseEvent, want string, timeout time.Duration) {
	t.Helper()

	deadline := time.NewTimer(timeout)
	defer deadline.Stop()

	for {
		select {
		case got, ok := <-eventsCh:
			if !ok {
				t.Fatalf("event stream closed before %q", want)
			}

			if got.Err != nil {
				t.Fatalf("stream read error: %v", got.Err)
			}

			if got.Event == want {
				return
			}
		case <-deadline.C:
			t.Fatalf("timed out waiting for SSE event %q", want)
		}
	}
}

func assertStreamClosed(t *testing.T, eventsCh <-chan sseEvent, timeout time.Duration) {
	t.Helper()

	select {
	case got := <-eventsCh:
		if got.Err == nil {
			t.Fatalf("expected stream to close after done, got event=%q data=%q", got.Event, got.Data)
		}

		if !errors.Is(got.Err, io.EOF) && !strings.Contains(got.Err.Error(), "closed") {
			t.Fatalf("expected EOF/closed after done, got: %v", got.Err)
		}
	case <-time.After(timeout):
		t.Fatalf("expected stream to close after done")
	}
}

func assertHandlerDone(t *testing.T, done <-chan struct{}, timeout time.Duration) {
	t.Helper()

	select {
	case <-done:
	case <-time.After(timeout):
		t.Fatalf("expected handler to return after done")
	}
}

func TestTerminalDonePayloadFromUpdate_RejectsInvalidTerminalState(t *testing.T) {
	t.Parallel()

	payload, err := json.Marshal(map[string]string{
		"type":  "complete",
		"state": "SCANNING",
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	_, isTerminal, parseErr := terminalDonePayloadFromUpdate(payload)
	if isTerminal {
		t.Fatal("expected invalid terminal state payload to be rejected")
	}

	if parseErr == nil {
		t.Fatal("expected parse error for invalid terminal state payload")
	}
}

func TestTerminalDonePayloadFromUpdate_DerivesStateFromCompleteType(t *testing.T) {
	t.Parallel()

	payload, err := json.Marshal(map[string]string{
		"type": "complete",
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	done, isTerminal, parseErr := terminalDonePayloadFromUpdate(payload)
	if parseErr != nil {
		t.Fatalf("unexpected parse error: %v", parseErr)
	}

	if !isTerminal {
		t.Fatal("expected complete type to be terminal")
	}

	if done.State != models.JobStateDone {
		t.Fatalf("expected DONE state, got %q", done.State)
	}
}
