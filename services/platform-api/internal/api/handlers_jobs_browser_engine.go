package api

import "strings"

const defaultBrowserEngine = "chromium"

var allowedBrowserEngines = map[string]struct{}{
	"chromium": {},
	"firefox":  {},
	"webkit":   {},
}

// normalizeBrowserEngine validates the requested Playwright engine, defaulting
// unknown or empty values to chromium. This is the single trusted gate: the
// orchestrator forwards the result as the reserved BROWSER_ENGINE env var.
func normalizeBrowserEngine(value string) string {
	key := strings.ToLower(strings.TrimSpace(value))
	if _, ok := allowedBrowserEngines[key]; ok {
		return key
	}

	return defaultBrowserEngine
}
