package api

import "strings"

const defaultHighlightStyle = "dashed"

var allowedHighlightStyles = map[string]struct{}{
	"dashed": {},
	"solid":  {},
}

func normalizeHighlightStyle(value string) string {
	key := strings.ToLower(strings.TrimSpace(value))
	if _, ok := allowedHighlightStyles[key]; ok {
		return key
	}

	return defaultHighlightStyle
}
