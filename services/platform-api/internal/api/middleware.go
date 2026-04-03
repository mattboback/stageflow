package api

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/mattboback/stageflow/libs/go/httputil"
	"github.com/mattboback/stageflow/libs/go/logging"
)

const (
	// defaultRequestTimeout is the maximum duration for processing a request.
	// This prevents slow clients or runaway handlers from consuming resources indefinitely.
	defaultRequestTimeout = 60 * time.Second

	// uploadRequestTimeout is extended for file upload endpoints.
	uploadRequestTimeout = 5 * time.Minute

	defaultRateLimitRequestsPerMinute = 120
	rateLimitWindow                   = time.Minute
	unknownRateLimitKey               = "unknown"
)

type rateWindow struct {
	start time.Time
	count int
}

type inMemoryRateLimiter struct {
	mu      sync.Mutex
	windows map[string]rateWindow
	limit   int
}

func newInMemoryRateLimiter(limit int) *inMemoryRateLimiter {
	return &inMemoryRateLimiter{
		windows: make(map[string]rateWindow),
		limit:   limit,
	}
}

func (l *inMemoryRateLimiter) allow(key string, now time.Time) (bool, int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	window := l.windows[key]
	if window.start.IsZero() || now.Sub(window.start) >= rateLimitWindow {
		window = rateWindow{start: now, count: 0}
	}

	if window.count >= l.limit {
		retryAfter := int(rateLimitWindow.Seconds()) - int(now.Sub(window.start).Seconds())
		if retryAfter < 1 {
			retryAfter = 1
		}

		l.windows[key] = window

		return false, retryAfter
	}

	window.count++
	l.windows[key] = window

	return true, 0
}

var apiRateLimiter = newInMemoryRateLimiter(defaultRateLimitRequestsPerMinute)

func rateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := rateLimitKey(r)

		allowed, retryAfter := apiRateLimiter.allow(key, time.Now().UTC())
		if !allowed {
			detail := httputil.NewRateLimitError(
				retryAfter,
				strconv.Itoa(defaultRateLimitRequestsPerMinute),
				"1 minute",
			)
			httputil.RespondStructuredError(w, http.StatusTooManyRequests, detail)

			return
		}

		next.ServeHTTP(w, r)
	}
}

func rateLimitKey(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		if len(parts) > 0 {
			candidate := strings.TrimSpace(parts[0])
			if candidate != "" {
				return candidate
			}
		}
	}

	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}

	if strings.TrimSpace(r.RemoteAddr) != "" {
		return strings.TrimSpace(r.RemoteAddr)
	}

	return unknownRateLimitKey
}

func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		requestID := uuid.New().String()[:8]
		ctx := logging.WithRequestID(r.Context(), requestID)
		r = r.WithContext(ctx)

		// Add request ID to response header for client correlation
		w.Header().Set("X-Request-ID", requestID)

		lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(lrw, r)

		slog.Info("HTTP request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", lrw.statusCode,
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", requestID,
		)
	}
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *loggingResponseWriter) Flush() {
	flusher, ok := lrw.ResponseWriter.(http.Flusher)
	if !ok {
		return
	}

	flusher.Flush()
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	rawOrigins := os.Getenv("PLATFORM_API_CORS_ALLOW_ORIGINS")
	allowed := parseAllowedOrigins(rawOrigins)

	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && (allowed["*"] || allowed[origin]) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Api-Key")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)

			return
		}

		next.ServeHTTP(w, r)
	}
}

func apiKeyMiddleware(next http.HandlerFunc) http.HandlerFunc {
	expected := strings.TrimSpace(os.Getenv("PLATFORM_API_TOKEN"))
	if expected == "" {
		return next
	}

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

		if provided == "" || provided != expected {
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
// If the handler takes longer than the timeout, the client receives a 503 Service Unavailable.
func timeoutMiddleware(timeout time.Duration, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		r = r.WithContext(ctx)

		done := make(chan struct{})
		tw := newTimeoutResponseWriter()

		go func() {
			next.ServeHTTP(tw, r)
			close(done)
		}()

		select {
		case <-done:
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				writeTimeoutJSON(w, r, timeout)

				return
			}

			// Handler completed normally.
			code, header, body := tw.snapshot()
			for k, vv := range header {
				w.Header()[k] = vv
			}

			w.WriteHeader(code)

			if _, err := w.Write(body); err != nil {
				slog.Warn("Failed to write buffered response", "error", err)
			}
		case <-ctx.Done():
			// Timeout reached.
			tw.markTimedOut()
			writeTimeoutJSON(w, r, timeout)
		}
	}
}

func writeTimeoutJSON(w http.ResponseWriter, r *http.Request, timeout time.Duration) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusServiceUnavailable)

	if _, err := w.Write([]byte(`{"error":"request timeout","code":"REQUEST_TIMEOUT"}`)); err != nil {
		slog.Warn("Failed to write timeout response", "error", err)
	}

	slog.Warn("Request timeout",
		"method", r.Method,
		"path", r.URL.Path,
		"timeout", timeout.String(),
	)
}

type timeoutResponseWriter struct {
	h     http.Header
	buf   bytes.Buffer
	mu    sync.Mutex
	code  int
	wrote bool
	timed bool
}

func newTimeoutResponseWriter() *timeoutResponseWriter {
	return &timeoutResponseWriter{h: make(http.Header)}
}

func (tw *timeoutResponseWriter) Header() http.Header {
	return tw.h
}

func (tw *timeoutResponseWriter) WriteHeader(code int) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if tw.timed {
		return
	}

	if tw.wrote {
		return
	}

	tw.wrote = true
	tw.code = code
}

func (tw *timeoutResponseWriter) Write(b []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if tw.timed {
		return 0, http.ErrHandlerTimeout
	}

	if !tw.wrote {
		tw.wrote = true
		tw.code = http.StatusOK
	}

	return tw.buf.Write(b)
}

func (tw *timeoutResponseWriter) markTimedOut() {
	tw.mu.Lock()
	tw.timed = true
	tw.mu.Unlock()
}

func (tw *timeoutResponseWriter) snapshot() (code int, header http.Header, body []byte) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	code = tw.code
	if !tw.wrote {
		code = http.StatusOK
	}

	header = tw.h.Clone()
	body = append([]byte(nil), tw.buf.Bytes()...)

	return code, header, body
}
