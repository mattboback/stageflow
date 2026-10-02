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
	"sync/atomic"
	"testing"
	"time"

	status "github.com/mattboback/stageflow/internal/jobstore"

	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/models"
	jobstatus "github.com/mattboback/stageflow/services/platform-api/tests/statusfixture"
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
	server, _, _ := newTestServer(t)

	if _, err := testPipeline(server).Apply(context.Background(), jobstatus.Signal{
		Kind:       jobstatus.SignalJobCreated,
		ObservedAt: time.Now().UTC(),
		JobCreated: &events.JobCreatedPayload{
			JobID:     sseTestJobID,
			InputType: events.InputTypeURLs,
			URLs:      []string{"https://example.com"},
			Config: models.JobConfig{
				Modules: []string{scannerTypeAxe},
			},
		},
	}); err != nil {
		t.Fatalf("create job: %v", err)
	}

	eventsCh, done := startJobStream(t, server, sseTestJobID)

	waitForSSEEvent(t, eventsCh, "status")

	if _, err := testPipeline(server).Apply(context.Background(), jobstatus.Signal{
		Kind:       jobstatus.SignalJobCompleted,
		ObservedAt: time.Now().UTC(),
		JobCompleted: &events.JobCompletedPayload{
			JobID:  sseTestJobID,
			Status: events.JobStatusSuccess,
			Artifacts: events.ArtifactLocations{
				ReportJSON: sseTestJobID + "/report.json",
			},
		},
	}); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	waitForSSEEvent(t, eventsCh, "update")
	waitForSSEEvent(t, eventsCh, "done")

	assertStreamClosed(t, eventsCh, 2*time.Second)
	assertHandlerDone(t, done, 2*time.Second)
}

func startJobStream(t *testing.T, server *Server, jobID string) (eventsCh chan sseEvent, done chan struct{}) {
	t.Helper()
	return startJobStreamContext(context.Background(), t, server, jobID)
}

func startJobStreamContext(
	ctx context.Context,
	t *testing.T,
	server *Server,
	jobID string,
) (eventsCh chan sseEvent, done chan struct{}) {
	t.Helper()

	pr, pw := io.Pipe()

	t.Cleanup(func() {
		_ = pr.Close()
	})

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/v1/jobs/"+jobID+"/stream", http.NoBody).
		WithContext(ctx)

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

func waitForSSEEvent(t *testing.T, eventsCh <-chan sseEvent, want string) {
	t.Helper()

	deadline := time.NewTimer(2 * time.Second)
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

func TestJobIDFromJobPath_RejectsMalformedID(t *testing.T) {
	if _, ok := jobIDFromJobPath("/api/v1/jobs/bad id/stream", "stream"); ok {
		t.Fatal("expected a job ID with a space to be rejected")
	}

	valid := "123e4567-e89b-12d3-a456-426614174000"
	if id, ok := jobIDFromJobPath("/api/v1/jobs/"+valid+"/stream", "stream"); !ok || id != valid {
		t.Fatalf("expected valid UUID to be accepted, got id=%q ok=%v", id, ok)
	}
}

func TestValidJobID(t *testing.T) {
	for _, good := range []string{
		"123e4567-e89b-12d3-a456-426614174000", // production UUID
		"job-structured-screenshots",           // readable fixture id
		"12345",
	} {
		if !validJobID(good) {
			t.Fatalf("expected %q to be accepted", good)
		}
	}

	for _, bad := range []string{
		"",
		"../../etc/passwd",      // path traversal
		"drop table jobs",       // spaces / SQL metachars
		"id\ninjected-log-line", // control char (log injection)
		strings.Repeat("a", 65), // over length bound
	} {
		if validJobID(bad) {
			t.Fatalf("expected %q to be rejected", bad)
		}
	}
}

type errorAfterInitialReader struct{ reads atomic.Int32 }

func (r *errorAfterInitialReader) GetJob(context.Context, string) (*status.JobRecord, error) {
	if r.reads.Add(1) > 1 {
		return nil, errors.New("database unavailable")
	}

	return &status.JobRecord{JobID: sseTestJobID, State: models.JobStateScanning}, nil
}
func TestStreamDatabaseErrorClosesWithoutDeclaringJobFailed(t *testing.T) {
	server, _, _ := newTestServer(t)
	reader := &errorAfterInitialReader{}
	server.statusReader = reader
	events, done := startJobStream(t, server, sseTestJobID)
	waitForSSEEvent(t, events, "status")
	assertStreamClosed(t, events, 2*time.Second)
	assertHandlerDone(t, done, time.Second)

	if reader.reads.Load() != 2 {
		t.Fatalf("expected one refresh, got %d", reader.reads.Load())
	}
}
func TestStreamCancellationStopsSnapshotPolling(t *testing.T) {
	server, _, _ := newTestServer(t)
	reader := &errorAfterInitialReader{}
	server.statusReader = reader

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events, done := startJobStreamContext(ctx, t, server, sseTestJobID)
	waitForSSEEvent(t, events, "status")
	cancel()
	assertHandlerDone(t, done, time.Second)

	if reader.reads.Load() != 1 {
		t.Fatalf("polled after disconnect: %d", reader.reads.Load())
	}
}

func TestTerminalInitialSnapshotDoesNotReplayReady(t *testing.T) {
	server, _, _ := newTestServer(t)
	server.statusReader = &staticJobReader{rec: &status.JobRecord{
		JobID: sseTestJobID, State: models.JobStateDone, TotalPages: 30,
		ExpectedScanners: []string{"axe"}, CompletedScanners: []string{"axe"},
		ScannerStatuses: map[string]models.ScannerStatus{"axe": {State: "completed", PagesScanned: 30, TotalPages: 30}},
	}}
	events, done := startJobStream(t, server, sseTestJobID)

	for _, want := range []string{"status", "done"} {
		select {
		case event := <-events:
			if event.Event != want {
				t.Fatalf("got %q, want %q: %s", event.Event, want, event.Data)
			}

			var payload struct {
				State models.JobState `json:"state"`
			}
			if err := json.Unmarshal([]byte(event.Data), &payload); err != nil {
				t.Fatal(err)
			}

			if payload.State != models.JobStateDone {
				t.Fatalf("state regressed on reload: %s", event.Data)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("missing %s event", want)
		}
	}

	assertHandlerDone(t, done, time.Second)
}
