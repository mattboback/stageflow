package api

import (
	"errors"
	"strings"

	"github.com/mattboback/stageflow/libs/go/httputil"
	"github.com/mattboback/stageflow/libs/go/scannercatalog"
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

// unsupportedModuleDetail describes a catalog ModuleError for the client.
func unsupportedModuleDetail(err error) (httputil.ErrorDetail, bool) {
	var moduleErr *scannercatalog.ModuleError
	if !errors.As(err, &moduleErr) {
		return httputil.ErrorDetail{}, false
	}

	module := moduleErr.Module
	if module == "" {
		module = "unknown"
	}

	return httputil.NewUnsupportedModuleError(module, scannercatalog.IDs()), true
}
