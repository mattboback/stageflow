package jobstream

import (
	"bufio"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestReadNextSSEEvent_EventAndData(t *testing.T) {
	t.Parallel()

	r := bufio.NewReader(strings.NewReader("event: status\ndata: {\"state\":\"SCANNING\"}\n\n"))

	eventType, data, err := readNextSSEEvent(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if eventType != "status" {
		t.Fatalf("eventType = %q, want %q", eventType, "status")
	}

	if data != `{"state":"SCANNING"}` {
		t.Fatalf("data = %q, want %q", data, `{"state":"SCANNING"}`)
	}
}

func TestReadNextSSEEvent_MultiLineDataJoinedWithNewline(t *testing.T) {
	t.Parallel()

	r := bufio.NewReader(strings.NewReader("event: update\ndata: line1\ndata: line2\n\n"))

	eventType, data, err := readNextSSEEvent(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if eventType != "update" {
		t.Fatalf("eventType = %q, want %q", eventType, "update")
	}

	if data != "line1\nline2" {
		t.Fatalf("data = %q, want %q", data, "line1\nline2")
	}
}

func TestReadNextSSEEvent_SkipsKeepaliveCommentsAndHandlesCRLF(t *testing.T) {
	t.Parallel()

	r := bufio.NewReader(strings.NewReader(":keepalive\r\nevent: done\r\ndata: {}\r\n\r\n"))

	eventType, data, err := readNextSSEEvent(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if eventType != "done" {
		t.Fatalf("eventType = %q, want %q", eventType, "done")
	}

	if data != "{}" {
		t.Fatalf("data = %q, want %q", data, "{}")
	}
}

func TestReadNextSSEEvent_PropagatesEOFBeforeDispatch(t *testing.T) {
	t.Parallel()

	r := bufio.NewReader(strings.NewReader("event: status\ndata: partial"))

	_, _, err := readNextSSEEvent(r)
	if !errors.Is(err, io.EOF) {
		t.Fatalf("err = %v, want io.EOF on truncated stream", err)
	}
}
