package main

import (
	"fmt"
	"strings"
)

type outputFormat string

const (
	outputFormatText     outputFormat = "text"
	outputFormatMarkdown outputFormat = "markdown"
	outputFormatJSON     outputFormat = "json"
)

func normalizeOutputFormat(raw string) (outputFormat, error) {
	trimmed := strings.TrimSpace(strings.ToLower(raw))
	if trimmed == "" {
		return outputFormatText, nil
	}

	format := outputFormat(trimmed)
	switch format {
	case outputFormatText, outputFormatMarkdown, outputFormatJSON:
		return format, nil
	default:
		return "", fmt.Errorf("unsupported --format %q (expected text, markdown, or json)", raw)
	}
}
