package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/mattboback/stageflow/libs/go/httputil"
)

const maxDiscoverBodySize = 4 << 10

type discoverRequest struct {
	URL string `json:"url"`
}

// handleDiscover lists the pages of a site so a visitor does not have to type
// every URL. It is public, so it shares the anonymous submission rate limit and
// never opts into private targets.
func (s *Server) handleDiscover(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxDiscoverBodySize)

	var req discoverRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "Invalid JSON body")

		return
	}

	target := strings.TrimSpace(req.URL)

	err := validateTargetURLsWithResolver(r.Context(), s.ipResolver, []string{target}, s.discoveryMode)
	if err != nil {
		httputil.RespondStructuredError(w, http.StatusBadRequest, httputil.NewValidationError(
			"url",
			"The site URL is invalid or not allowed",
			"Use a public http/https URL without embedded credentials.",
		))

		return
	}

	// validateTargetURLsWithResolver already parsed this URL successfully.
	start, _ := url.Parse(target)

	httputil.RespondOK(w, discoverPages(r.Context(), newDiscoveryClient(s.discoveryMode), start))
}
