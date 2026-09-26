package api

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"
)

// The optional API token.
//
// Every route is public. PLATFORM_API_TOKEN, when set,
// identifies a trusted caller: it skips the rate limits, may opt into private
// targets on an instance that allows them. Nothing else
// depends on it, so a missing or wrong key is never a 401 on a public route.

type apiKeyCallerKey struct{}

// identifyAPIKeyCaller marks the request context when the caller presents the
// configured token, via X-Api-Key or Authorization: Bearer.
func (s *Server) identifyAPIKeyCaller(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if apiKeyMatches(r, s.config.APIToken) {
			r = r.WithContext(context.WithValue(r.Context(), apiKeyCallerKey{}, true))
		}

		next.ServeHTTP(w, r)
	}
}

// isAPIKeyCaller reports whether identifyAPIKeyCaller matched the token.
func isAPIKeyCaller(ctx context.Context) bool {
	matched, ok := ctx.Value(apiKeyCallerKey{}).(bool)

	return ok && matched
}

func apiKeyMatches(r *http.Request, token string) bool {
	expected := strings.TrimSpace(token)
	if expected == "" {
		return false
	}

	provided := strings.TrimSpace(r.Header.Get("X-Api-Key"))
	if provided == "" {
		auth := strings.TrimSpace(r.Header.Get("Authorization"))
		if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
			provided = strings.TrimSpace(auth[len("bearer "):])
		}
	}

	return provided != "" && subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}
