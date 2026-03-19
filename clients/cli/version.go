package main

import "strings"

var (
	// Populated at build time via -ldflags.
	version = "dev"
	commit  = ""
	date    = ""
)

func formatVersion() string {
	v := strings.TrimSpace(version)
	if v == "" {
		v = "dev"
	}

	c := strings.TrimSpace(commit)
	if len(c) > 12 {
		c = c[:12]
	}

	d := strings.TrimSpace(date)

	parts := []string{v}
	if c != "" {
		parts = append(parts, c)
	}

	if d != "" {
		parts = append(parts, d)
	}

	return strings.Join(parts, " ")
}
