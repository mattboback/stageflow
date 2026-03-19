package api

import (
	"net/http"
	"strings"
)

// Router returns the HTTP router with all public routes configured.
func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()

	// Upload endpoints get extended timeout for large file transfers.
	mux.HandleFunc("/api/v1/jobs/zip", s.withUploadMiddleware(s.handleJobZipUpload))
	mux.HandleFunc("/api/v1/jobs/urls", s.withMiddleware(s.handleJobURLSubmit))
	mux.HandleFunc("/api/v1/jobs/", s.handleJobsRoute)
	mux.HandleFunc("/api/v1/scanners", s.withMiddleware(s.handleListScanners))
	mux.HandleFunc("/healthz", s.handleHealth)

	return mux
}

func (s *Server) handleJobsRoute(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/jobs/")
	parts := strings.Split(path, "/")

	if len(parts) >= 2 && parts[1] == "stream" {
		// SSE endpoint - no timeout middleware (long-lived connection).
		s.withStreamMiddleware(s.handleJobStream)(w, r)

		return
	}

	s.withMiddleware(s.handleJobStatus)(w, r)
}

func (s *Server) withMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return loggingMiddleware(corsMiddleware(apiKeyMiddleware(timeoutMiddleware(defaultRequestTimeout, next))))
}

func (s *Server) withUploadMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return loggingMiddleware(corsMiddleware(apiKeyMiddleware(timeoutMiddleware(uploadRequestTimeout, next))))
}

func (s *Server) withStreamMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return loggingMiddleware(corsMiddleware(apiKeyMiddleware(next)))
}
