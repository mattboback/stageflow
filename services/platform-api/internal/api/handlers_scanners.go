package api

import (
	"net/http"

	"github.com/mattboback/stageflow/libs/go/httputil"
	"github.com/mattboback/stageflow/libs/go/scannercatalog"
)

// handleListScanners returns the built-in scanner catalog.
func (s *Server) handleListScanners(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.RespondError(w, http.StatusMethodNotAllowed, "Method not allowed")

		return
	}

	type scannerInfo struct {
		ID          string   `json:"id"`
		Name        string   `json:"name"`
		Version     string   `json:"version"`
		Description string   `json:"description,omitempty"`
		Categories  []string `json:"categories"`
		Aliases     []string `json:"aliases,omitempty"`
		// Every scanner is built in and enabled. Both fields stay because
		// clients read them.
		Enabled      bool                        `json:"enabled"`
		BuiltIn      bool                        `json:"builtIn"`
		Capabilities scannercatalog.Capabilities `json:"capabilities"`
	}

	type response struct {
		Scanners   []scannerInfo `json:"scanners"`
		Total      int           `json:"total"`
		Enabled    int           `json:"enabled"`
		Categories []string      `json:"categories,omitempty"`
	}

	scanners := scannercatalog.All()

	out := make([]scannerInfo, 0, len(scanners))
	for _, scanner := range scanners {
		out = append(out, scannerInfo{
			ID:           scanner.ID,
			Name:         scanner.Name,
			Version:      scanner.Version,
			Description:  scanner.Description,
			Categories:   scanner.Categories,
			Aliases:      scanner.Aliases,
			Enabled:      true,
			BuiltIn:      true,
			Capabilities: scanner.Capabilities,
		})
	}

	httputil.RespondOK(w, response{
		Scanners:   out,
		Total:      len(out),
		Enabled:    len(out),
		Categories: scannercatalog.Categories(),
	})
}
