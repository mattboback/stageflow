package api

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/mattboback/stageflow/libs/go/httputil"
)

const rateLimiterStaleAfter = 5 * time.Minute

type adminRateLimiter struct {
	mu      sync.Mutex
	rps     float64
	burst   float64
	clients map[string]*adminClientLimit
	now     func() time.Time
}

type adminClientLimit struct {
	tokens float64
	last   time.Time
}

func newAdminRateLimiter(rps, burst int) *adminRateLimiter {
	if rps <= 0 {
		return nil
	}

	if burst <= 0 {
		burst = rps
	}

	return &adminRateLimiter{
		rps:     float64(rps),
		burst:   float64(burst),
		clients: make(map[string]*adminClientLimit),
		now:     time.Now,
	}
}

func (l *adminRateLimiter) allow(client string) bool {
	if l == nil {
		return true
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.pruneLocked(now)

	bucket, ok := l.clients[client]
	if !ok {
		bucket = &adminClientLimit{tokens: l.burst, last: now}
		l.clients[client] = bucket
	}

	elapsed := now.Sub(bucket.last).Seconds()
	bucket.tokens = min(l.burst, bucket.tokens+elapsed*l.rps)
	bucket.last = now

	if bucket.tokens < 1 {
		return false
	}

	bucket.tokens--

	return true
}

func (l *adminRateLimiter) pruneLocked(now time.Time) {
	for client, bucket := range l.clients {
		if now.Sub(bucket.last) > rateLimiterStaleAfter {
			delete(l.clients, client)
		}
	}
}

func (s *Server) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			panicValue := recover()
			if panicValue == nil {
				return
			}

			slog.Error(
				"Recovered panic in admin API handler",
				"method",
				r.Method,
				"path",
				r.URL.Path,
				"job_id",
				extractJobIDFromPath(r.URL.Path),
				"remote_addr",
				r.RemoteAddr,
				"panic",
				panicValue,
				"stack",
				string(debug.Stack()),
			)
			httputil.RespondError(w, http.StatusInternalServerError, "Internal server error")
		}()

		next.ServeHTTP(w, r)
	})
}

// statusRecorder captures the response status code so it can be recorded as a
// metric. It defaults to 200 because handlers that never call WriteHeader still
// emit a 200 once the body is written.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (s *Server) recordMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.metrics == nil {
			next.ServeHTTP(w, r)

			return
		}

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		s.metrics.ObserveHTTP(rec.status)
	})
}

func (s *Server) rateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.rateLimiter != nil && r.URL.Path != "/healthz" {
			client := clientKey(r)
			if !s.rateLimiter.allow(client) {
				slog.Warn(
					"Admin API rate limit exceeded",
					"client",
					client,
					"method",
					r.Method,
					"path",
					r.URL.Path,
				)
				httputil.RespondError(w, http.StatusTooManyRequests, "Rate limit exceeded")

				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func clientKey(r *http.Request) string {
	forwardedFor := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if forwardedFor != "" {
		first, _, _ := strings.Cut(forwardedFor, ",")
		if first = strings.TrimSpace(first); first != "" {
			return first
		}
	}

	realIP := strings.TrimSpace(r.Header.Get("X-Real-IP"))
	if realIP != "" {
		return realIP
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}

	if strings.TrimSpace(r.RemoteAddr) != "" {
		return r.RemoteAddr
	}

	return "unknown"
}

func extractJobIDFromPath(path string) string {
	trimmed := strings.TrimPrefix(path, "/api/v1/jobs/")
	if trimmed == path || trimmed == "" {
		return ""
	}

	jobID, _, _ := strings.Cut(trimmed, "/")

	return jobID
}

func (l *adminRateLimiter) String() string {
	if l == nil {
		return "disabled"
	}

	return fmt.Sprintf("rps=%g burst=%g", l.rps, l.burst)
}
