package main

import (
	"fmt"
	"time"
)

func formatDuration(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func truncate(value string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}

	if len(value) <= maxLen {
		return value
	}

	return value[:maxLen]
}

func abbreviate(value string, prefixLen int) string {
	if value == "" {
		return "-"
	}

	if prefixLen <= 0 {
		return "..."
	}

	if len(value) <= prefixLen {
		return value
	}

	return value[:prefixLen] + "..."
}

func truncateWithEllipsis(value string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}

	if len(value) <= maxLen {
		return value
	}

	if maxLen <= 3 {
		return truncate("...", maxLen)
	}

	return value[:maxLen-3] + "..."
}

func formatJobID(jobID *string) string {
	if jobID == nil || *jobID == "" {
		return "-"
	}

	return abbreviate(*jobID, 8)
}

func formatJobState(state *string) string {
	if state == nil || *state == "" {
		return "-"
	}

	return *state
}
