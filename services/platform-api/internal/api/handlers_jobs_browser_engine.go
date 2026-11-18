package api

import (
	"strings"

	"github.com/mattboback/stageflow/libs/go/httputil"
)

// supportedBrowserEngine is the only engine the scanner image ships. Released
// clients may still send `browser`, so an omitted or chromium value is
// accepted and dropped; any other engine is a client error.
const supportedBrowserEngine = "chromium"

// checkBrowserEngine reports whether a request's browser field is acceptable
// and, when it is not, the validation detail to answer with.
func checkBrowserEngine(value string) (httputil.ErrorDetail, bool) {
	key := strings.ToLower(strings.TrimSpace(value))
	if key == "" || key == supportedBrowserEngine {
		return httputil.ErrorDetail{}, true
	}

	return httputil.NewValidationError(
		"browser",
		"Only the chromium browser engine is available",
		"Omit browser or set it to chromium.",
	), false
}
