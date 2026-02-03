package api

import (
	"net/http"
	"sort"

	"github.com/mattboback/stageflow/packages/shared-go/httputil"
)

// handleListScanners returns scanner metadata from the shared scanner registry.
func (s *Server) handleListScanners(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

		return
	}

	type capabilities struct {
		OutputFormats        []string `json:"outputFormats,omitempty"`
		SupportsScreenshots  bool     `json:"supportsScreenshots"`
		SupportsConcurrency  bool     `json:"supportsConcurrency"`
		RequiresBrowser      bool     `json:"requiresBrowser"`
		SupportsOffline      bool     `json:"supportsOffline"`
		MaxConcurrency       int      `json:"maxConcurrency,omitempty"`
		EstimatedTimePerPage int      `json:"estimatedTimePerPage,omitempty"`
	}

	type scannerInfo struct {
		ID           string       `json:"id"`
		Name         string       `json:"name"`
		Version      string       `json:"version"`
		Description  string       `json:"description,omitempty"`
		Categories   []string     `json:"categories"`
		Aliases      []string     `json:"aliases,omitempty"`
		Image        string       `json:"image,omitempty"`
		Enabled      bool         `json:"enabled"`
		BuiltIn      bool         `json:"builtIn"`
		Capabilities capabilities `json:"capabilities"`
	}

	type response struct {
		Scanners   []scannerInfo `json:"scanners"`
		Total      int           `json:"total"`
		Enabled    int           `json:"enabled"`
		Categories []string      `json:"categories,omitempty"`
	}

	if s.scannerRegistry == nil {
		httputil.RespondOK(w, response{
			Scanners: []scannerInfo{
				{
					ID:         scannerTypeAxe,
					Name:       "Axe",
					Version:    "unknown",
					Categories: []string{"accessibility"},
					Enabled:    true,
					BuiltIn:    true,
					Capabilities: capabilities{
						OutputFormats:       []string{"json", "html"},
						SupportsScreenshots: true,
						SupportsConcurrency: true,
						RequiresBrowser:     true,
						SupportsOffline:     false,
						MaxConcurrency:      4,
					},
				},
			},
			Total:      1,
			Enabled:    1,
			Categories: []string{"accessibility"},
		})

		return
	}

	categorySet := make(map[string]struct{})
	defs := s.scannerRegistry.List()
	out := make([]scannerInfo, 0, len(defs))
	enabledCount := 0

	for _, def := range defs {
		if def.Enabled {
			enabledCount++

			for _, cat := range def.Categories {
				categorySet[cat] = struct{}{}
			}
		}

		out = append(out, scannerInfo{
			ID:          def.ID,
			Name:        def.Name,
			Version:     def.Version,
			Description: def.Description,
			Categories:  def.Categories,
			Aliases:     def.Aliases,
			Image:       def.Image,
			Enabled:     def.Enabled,
			BuiltIn:     def.BuiltIn,
			Capabilities: capabilities{
				OutputFormats:        def.Capabilities.OutputFormats,
				SupportsScreenshots:  def.Capabilities.SupportsScreenshots,
				SupportsConcurrency:  def.Capabilities.SupportsConcurrency,
				RequiresBrowser:      def.Capabilities.RequiresBrowser,
				SupportsOffline:      def.Capabilities.SupportsOffline,
				MaxConcurrency:       def.Capabilities.MaxConcurrency,
				EstimatedTimePerPage: def.Capabilities.EstimatedTimePerPage,
			},
		})
	}

	categories := make([]string, 0, len(categorySet))
	for cat := range categorySet {
		categories = append(categories, cat)
	}

	sort.Strings(categories)

	httputil.RespondOK(w, response{
		Scanners:   out,
		Total:      len(out),
		Enabled:    enabledCount,
		Categories: categories,
	})
}
