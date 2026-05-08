package jobstream

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type closeTrackingBody struct {
	io.Reader
	closed bool
}

func (b *closeTrackingBody) Close() error {
	b.closed = true

	return nil
}

type readErrorBody struct {
	err    error
	closed bool
}

func (b *readErrorBody) Read(_ []byte) (int, error) {
	return 0, b.err
}

func (b *readErrorBody) Close() error {
	b.closed = true

	return nil
}

func TestDrainAndCloseResponseBodyClosesAfterDrainError(t *testing.T) {
	t.Parallel()

	readErr := errors.New("drain failed")
	body := &readErrorBody{err: readErr}

	err := drainAndCloseResponseBody(body)
	if !errors.Is(err, readErr) {
		t.Fatalf("expected drain error %v, got %v", readErr, err)
	}

	if !body.closed {
		t.Fatal("expected response body to be closed after drain error")
	}
}

func TestHandleSSEEvent_PrintsScannerCompletionSummary(t *testing.T) {
	t.Parallel()

	state := newStreamState()
	out := &bytes.Buffer{}

	statusPayload := `{"id":"job-1","state":"SCANNING","expected_scanners":["axe","lighthouse"],"completed_scanners":[],"remaining_scanners":["axe","lighthouse"]}`

	done, err := handleSSEEvent("status", statusPayload, out, state)
	if err != nil {
		t.Fatalf("handle status: %v", err)
	}

	if done {
		t.Fatal("status event should not terminate stream")
	}

	updatePayload := `{"type":"scanner_complete","state":"SCANNING","scanner_type":"axe","pages_scanned":1,"violations":3,"timing":{"total_ms":2100}}`

	done, err = handleSSEEvent("update", updatePayload, out, state)
	if err != nil {
		t.Fatalf("handle update: %v", err)
	}

	if done {
		t.Fatal("scanner_complete update should not terminate stream")
	}

	output := out.String()

	if want := "scanner: axe complete (2.1s, 1 page, 3 issues); remaining: lighthouse"; !bytes.Contains(
		[]byte(output),
		[]byte(want),
	) {
		t.Fatalf("expected output to contain %q, got %q", want, output)
	}
}

func TestEmitScannerCompletionsFromStatus_PrintsNewlyCompletedScannersOnce(t *testing.T) {
	t.Parallel()

	state := newStreamState()
	out := &bytes.Buffer{}

	status := &apiclient.JobStatus{
		State:             "SCANNING",
		ExpectedScanners:  []string{"axe", "lighthouse"},
		CompletedScanners: []string{"axe"},
		RemainingScanners: []string{"lighthouse"},
	}

	if err := emitScannerCompletionsFromStatus(out, state, status); err != nil {
		t.Fatalf("emitScannerCompletionsFromStatus: %v", err)
	}

	if got := out.String(); got != "scanner: axe complete; remaining: lighthouse\n" {
		t.Fatalf("unexpected first output: %q", got)
	}

	if err := emitScannerCompletionsFromStatus(out, state, status); err != nil {
		t.Fatalf("emitScannerCompletionsFromStatus second call: %v", err)
	}

	if got := out.String(); got != "scanner: axe complete; remaining: lighthouse\n" {
		t.Fatalf("expected duplicate status snapshot to be suppressed, got %q", got)
	}
}

func TestPollJobStateEscapesJobIDPathSegment(t *testing.T) {
	t.Parallel()

	client := apiclient.NewClient(
		"http://stageflow.test",
		"",
		&http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.URL.EscapedPath() != "/api/v1/jobs/job%2Fwith%3Freserved" {
					t.Fatalf("unexpected request path: %s", req.URL.EscapedPath())
				}

				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body: io.NopCloser(
						bytes.NewBufferString(`{"id":"job/with?reserved","state":"DONE"}`),
					),
				}, nil
			}),
		},
	)

	var out bytes.Buffer
	if err := pollJobState(context.Background(), client, "job/with?reserved", &out); err != nil {
		t.Fatalf("pollJobState: %v", err)
	}
}

func TestSSEJobStateEscapesJobIDAndClosesBodyAfterStatusError(t *testing.T) {
	t.Parallel()

	body := &closeTrackingBody{Reader: bytes.NewBufferString("diagnostic")}
	client := apiclient.NewClient(
		"http://stageflow.test",
		"",
		&http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.URL.EscapedPath() != "/api/v1/jobs/job%2Fwith%3Freserved/stream" {
					t.Fatalf("unexpected request path: %s", req.URL.EscapedPath())
				}

				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Header:     make(http.Header),
					Body:       body,
				}, nil
			}),
		},
	)

	var out bytes.Buffer
	if err := sseJobState(context.Background(), client, "job/with?reserved", &out); err == nil {
		t.Fatal("expected stream status error, got nil")
	}

	if !body.closed {
		t.Fatal("expected SSE response body to be closed")
	}
}
