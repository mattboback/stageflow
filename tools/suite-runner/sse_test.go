package main

import (
	"bufio"
	"strings"
	"testing"
)

func TestReadNextSSEEvent(t *testing.T) {
	stream := strings.Join([]string{
		"event: status",
		"data: {\"state\":\"PENDING\"}",
		"",
		"event: update",
		"data: {\"type\":\"progress\",\"state\":\"SCANNING\"}",
		"",
		":keepalive",
		"",
		"event: done",
		"data: {\"type\":\"complete\",\"state\":\"DONE\"}",
		"",
	}, "\n") + "\n"

	r := bufio.NewReader(strings.NewReader(stream))

	ev, data, err := readNextSSEEvent(r)
	if err != nil {
		t.Fatalf("read 1: %v", err)
	}

	if ev != "status" {
		t.Fatalf("expected status, got %q", ev)
	}

	if data != "{\"state\":\"PENDING\"}" {
		t.Fatalf("unexpected data: %q", data)
	}

	ev, data, err = readNextSSEEvent(r)
	if err != nil {
		t.Fatalf("read 2: %v", err)
	}

	if ev != "update" {
		t.Fatalf("expected update, got %q", ev)
	}

	if data != "{\"type\":\"progress\",\"state\":\"SCANNING\"}" {
		t.Fatalf("unexpected data: %q", data)
	}

	ev, data, err = readNextSSEEvent(r)
	if err != nil {
		t.Fatalf("read 3: %v", err)
	}
	// Keepalive is a comment-only message (no event/data).
	if ev != "" || data != "" {
		t.Fatalf("expected empty keepalive message, got event=%q data=%q", ev, data)
	}

	ev, data, err = readNextSSEEvent(r)
	if err != nil {
		t.Fatalf("read 4: %v", err)
	}

	if ev != "done" {
		t.Fatalf("expected done, got %q", ev)
	}

	if data != "{\"type\":\"complete\",\"state\":\"DONE\"}" {
		t.Fatalf("unexpected data: %q", data)
	}
}
