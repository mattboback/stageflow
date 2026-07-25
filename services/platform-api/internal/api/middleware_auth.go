package api

import (
	"crypto/subtle"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/mattboback/stageflow/libs/go/logging"
)

// API key authentication and its startup validation.

// ValidateAuthConfig enforces that PLATFORM_API_TOKEN is set at process
// startup. Operators can opt out explicitly by setting
// PLATFORM_API_AUTH_DISABLED=true, which makes the misconfiguration visible
// in logs rather than silently disabling authentication.
func ValidateAuthConfig() error {
	token := strings.TrimSpace(os.Getenv("PLATFORM_API_TOKEN"))
	disabled := strings.EqualFold(strings.TrimSpace(os.Getenv("PLATFORM_API_AUTH_DISABLED")), "true")

	if token == "" && !disabled {
		return errors.New(
			"PLATFORM_API_TOKEN is required (set PLATFORM_API_AUTH_DISABLED=true to run without authentication)",
		)
	}

	if disabled {
		slog.Warn(
			"SECURITY: Platform API authentication is DISABLED (PLATFORM_API_AUTH_DISABLED=true). "+
				"Every endpoint is unauthenticated. Only run this on a loopback/trusted network. "+
				"Set PLATFORM_API_TOKEN and unset PLATFORM_API_AUTH_DISABLED before exposing this API.",
			"auth_disabled", true,
			"token_set", token != "",
		)
	}

	return nil
}

// ValidateCORSConfig rejects a wildcard CORS origin when authentication is
// enabled. With "*" configured, corsMiddleware echoes back any Origin, so an
// authenticated API would accept credentialed cross-origin requests from any
// site — a CSRF anti-pattern. Operators must enumerate explicit origins. When
// authentication is disabled the wildcard is only warned about, since there are

func apiKeyMiddleware(next http.HandlerFunc) http.HandlerFunc {
	// Operators can explicitly run without authentication (local/dev) by setting
	// PLATFORM_API_AUTH_DISABLED=true. This must bypass the request-path check even
	// when PLATFORM_API_TOKEN is set; ValidateAuthConfig already logs the warning.
	if strings.EqualFold(strings.TrimSpace(os.Getenv("PLATFORM_API_AUTH_DISABLED")), "true") {
		return next
	}

	expected := strings.TrimSpace(os.Getenv("PLATFORM_API_TOKEN"))
	if expected == "" {
		return next
	}

	expectedBytes := []byte(expected)

	return func(w http.ResponseWriter, r *http.Request) {
		// Let CORS middleware short-circuit OPTIONS without requiring auth.
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)

			return
		}

		provided := strings.TrimSpace(r.Header.Get("X-Api-Key"))
		if provided == "" {
			auth := strings.TrimSpace(r.Header.Get("Authorization"))
			if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
				provided = strings.TrimSpace(auth[len("bearer "):])
			}
		}

		if provided == "" || subtle.ConstantTimeCompare([]byte(provided), expectedBytes) != 1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)

			if _, err := w.Write([]byte(`{"error":"unauthorized","code":"UNAUTHORIZED"}`)); err != nil {
				logging.Warn(r.Context(), "Failed to write unauthorized response", "error", err)
			}

			return
		}

		next.ServeHTTP(w, r)
	}
}
