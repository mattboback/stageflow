package main

import (
	"bytes"
	"testing"
)

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

	status := &JobStatus{
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
