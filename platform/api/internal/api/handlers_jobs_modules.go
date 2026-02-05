package api

import (
	"errors"
	"fmt"
	"strings"
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

	resolved, err := s.scannerRegistry.ResolveModulesStrict(explicit)
	if err != nil {
		supportedIDs := s.listSupportedModuleIDs()

		return nil, fmt.Errorf(
			"unsupported scanner module '%s' (supported: %s)",
			extractModuleName(err.Error()),
			strings.Join(supportedIDs, ", "),
		)
	}

	return resolved, nil
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

// extractModuleName assumes unsupported-module errors wrap the name in single quotes.
func extractModuleName(errMsg string) string {
	start := strings.Index(errMsg, "'")
	if start == -1 {
		return "unknown"
	}

	end := strings.Index(errMsg[start+1:], "'")
	if end == -1 {
		return "unknown"
	}

	return errMsg[start+1 : start+1+end]
}
