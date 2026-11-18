package render

import (
	"fmt"
	"strings"
)

type Format string

const (
	FormatText     Format = "text"
	FormatMarkdown Format = "markdown"
	FormatJSON     Format = "json"
)

func NormalizeFormat(raw string) (Format, error) {
	trimmed := strings.TrimSpace(strings.ToLower(raw))
	if trimmed == "" {
		return FormatText, nil
	}

	format := Format(trimmed)
	switch format {
	case FormatText, FormatMarkdown, FormatJSON:
		return format, nil
	default:
		return "", fmt.Errorf("unsupported --format %q (expected text, markdown, or json)", raw)
	}
}
