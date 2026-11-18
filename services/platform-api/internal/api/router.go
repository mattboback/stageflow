package api

import (
	"net/http"
	"strings"
	"time"
)

// Router returns the HTTP router with all public routes configured.
//
// Routes are public. See middleware_auth.go for the optional API token policy.
func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()
	submissions := newPublicSubmissionRateLimiterFromEnv()

	// Uploads get a longer read deadline and handler timeout.
	mux.HandleFunc("/api/v1/jobs/zip", requestReadDeadlineMiddleware(
		uploadRequestTimeout,
		s.publicRoute(uploadRequestTimeout, submissions, s.handleJobZipUpload),
	))
	mux.HandleFunc("/api/v1/jobs/urls", s.publicRoute(defaultRequestTimeout, submissions, s.handleJobURLSubmit))
	mux.HandleFunc("/api/v1/discover", s.publicRoute(defaultRequestTimeout, submissions, s.handleDiscover))
	mux.HandleFunc("/api/v1/jobs/", s.handleJobsRoute)
	mux.HandleFunc("/api/v1/scanners", s.publicRoute(defaultRequestTimeout, nil, s.handleListScanners))
	mux.HandleFunc("/healthz", s.handleHealth)

	return mux
}

// publicRoute wraps a public handler in logging, CORS, the optional API token
// and the general rate limit. A non-nil submissions limiter adds the stricter
// intake budget. A zero timeout leaves the handler unbounded, for SSE.
func (s *Server) publicRoute(
	timeout time.Duration,
	submissions *clientTokenBucketLimiter,
	next http.HandlerFunc,
) http.HandlerFunc {
	if timeout > 0 {
		next = timeoutMiddleware(timeout, next)
	}

	if submissions != nil {
		next = publicSubmissionRateLimitMiddleware(submissions, next)
	}

	return loggingMiddleware(corsMiddleware(s.identifyAPIKeyCaller(rateLimitMiddleware(next))))
}

func (s *Server) handleJobsRoute(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/jobs/")
	parts := strings.Split(path, "/")

	if r.Method == http.MethodDelete {
		s.publicRoute(defaultRequestTimeout, nil, s.handleJobDelete)(w, r)

		return
	}

	if len(parts) >= 2 && parts[1] == "stream" {
		s.publicRoute(0, nil, s.handleJobStream)(w, r)

		return
	}

	s.publicRoute(defaultRequestTimeout, nil, s.handleJobStatus)(w, r)
}
