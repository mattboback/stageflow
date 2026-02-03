package main

import (
	"bufio"
	"strings"
)

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
