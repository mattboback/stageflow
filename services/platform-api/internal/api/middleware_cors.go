package api

import (
	"errors"
	"log/slog"
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

// ValidateAuthConfig enforces that PLATFORM_API_TOKEN is set at process
// startup. Operators can opt out explicitly by setting
// PLATFORM_API_AUTH_DISABLED=true, which makes the misconfiguration visible

// ValidateCORSConfig rejects a wildcard CORS origin when authentication is
// enabled. With "*" configured, corsMiddleware echoes back any Origin, so an
// authenticated API would accept credentialed cross-origin requests from any
// site — a CSRF anti-pattern. Operators must enumerate explicit origins. When
// authentication is disabled the wildcard is only warned about, since there are
// no credentials to steal in that mode.
func ValidateCORSConfig() error {
	origins := parseAllowedOrigins(os.Getenv("PLATFORM_API_CORS_ALLOW_ORIGINS"))
	if !origins["*"] {
		return nil
	}

	authDisabled := strings.EqualFold(strings.TrimSpace(os.Getenv("PLATFORM_API_AUTH_DISABLED")), "true")
	if !authDisabled {
		return errors.New(
			"PLATFORM_API_CORS_ALLOW_ORIGINS=* is not allowed while authentication is enabled; " +
				"enumerate explicit origins (e.g. https://stageflow.org)",
		)
	}

	slog.Warn(
		"SECURITY: PLATFORM_API_CORS_ALLOW_ORIGINS=* echoes back any Origin. " +
			"This is tolerated only because authentication is disabled; set explicit origins before enabling auth.",
	)

	return nil
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

// timeoutMiddleware wraps a handler with a context deadline.
