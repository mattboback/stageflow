package api

import (
	"errors"
	"net/http"
	"os"
	"strings"
)

// CORS handling and its startup validation.

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	rawOrigins := os.Getenv("PLATFORM_API_CORS_ALLOW_ORIGINS")
	allowed := parseAllowedOrigins(rawOrigins)

	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && (allowed["*"] || allowed[origin]) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Api-Key")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)

			return
		}

		next.ServeHTTP(w, r)
	}
}

// ValidateCORSConfig rejects a wildcard CORS origin. With "*" configured,
// corsMiddleware echoes back any Origin, so any site could drive the API from a
// visitor's browser, including with a token a page has been given. Operators
// must enumerate explicit origins.
func ValidateCORSConfig() error {
	origins := parseAllowedOrigins(os.Getenv("PLATFORM_API_CORS_ALLOW_ORIGINS"))
	if !origins["*"] {
		return nil
	}

	return errors.New(
		"PLATFORM_API_CORS_ALLOW_ORIGINS=* is not allowed; enumerate explicit origins (e.g. https://stageflow.org)",
	)
}

func parseAllowedOrigins(raw string) map[string]bool {
	out := make(map[string]bool)

	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return out
	}

	parts := strings.Split(trimmed, ",")
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v == "" {
			continue
		}

		out[v] = true
	}

	return out
}
