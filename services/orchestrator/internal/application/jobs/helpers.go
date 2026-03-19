package jobs

import (
	"log/slog"
	"strings"
)

func shouldIgnoreMissingJob(eventName, jobID string, err error) bool {
	if err == nil {
		return false
	}

	if !strings.Contains(err.Error(), "job not found:") {
		return false
	}

	slog.Warn("Ignoring event for unknown job", "event", eventName, "job_id", jobID, "error", err)

	return true
}
