package api

import (
	"errors"
	"strings"

	"github.com/mattboback/stageflow/libs/go/httputil"
	"github.com/mattboback/stageflow/libs/go/scannerregistry"
)

func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	parts := strings.Split(raw, ",")

	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}

	return out
}

func (s *Server) normalizeModules(explicit []string) ([]string, error) {
	if s.scannerRegistry == nil {
		if len(explicit) == 0 {
			return []string{scannerTypeAxe}, nil
		}

		for _, mod := range explicit {
			if strings.ToLower(strings.TrimSpace(mod)) != scannerTypeAxe {
				return nil, errors.New("scanner registry not configured")
			}
		}

		return []string{scannerTypeAxe}, nil
	}

	return s.scannerRegistry.ResolveModulesStrict(explicit)
}

// unsupportedModuleDetail describes a registry ModuleError for the client.
func (s *Server) unsupportedModuleDetail(err error) (httputil.ErrorDetail, bool) {
	var moduleErr *scannerregistry.ModuleError
	if !errors.As(err, &moduleErr) {
		return httputil.ErrorDetail{}, false
	}

	module := moduleErr.Module
	if module == "" {
		module = "unknown"
	}

	return httputil.NewUnsupportedModuleError(module, s.listSupportedModuleIDs()), true
}

func (s *Server) listSupportedModuleIDs() []string {
	if s.scannerRegistry == nil {
		return []string{scannerTypeAxe}
	}

	defs := s.scannerRegistry.List()

	ids := make([]string, 0, len(defs))
	for _, def := range defs {
		ids = append(ids, def.ID)
	}

	return ids
}
