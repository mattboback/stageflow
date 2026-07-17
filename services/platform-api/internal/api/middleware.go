package api

import (
	"bytes"
	"context"
	"crypto/subtle"
	"errors"
	"log/slog"
	"math"
	"net"
	"net/http"
	"os"
	"runtime/debug"
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
	rateLimiterMaxEntries             = 10000
	maxTimeoutResponseBytes           = 72 * 1024 * 1024

	defaultPublicSubmissionRequestsPerMinute = 6
	defaultPublicSubmissionBurst             = 3
	publicSubmissionRateEnv                  = "PLATFORM_API_PUBLIC_SUBMISSION_RATE_LIMIT_RPM"
	publicSubmissionBurstEnv                 = "PLATFORM_API_PUBLIC_SUBMISSION_RATE_LIMIT_BURST"
	defaultTrustedProxyCIDRs                 = "127.0.0.1/32,::1/128"
)

type rateWindow struct {
	start time.Time
	count int
}

type inMemoryRateLimiter struct {
	mu      sync.Mutex
	windows map[string]rateWindow
	limit   int

	// saturationCount tracks how many requests were denied because the window
	// table was at capacity. lastSaturationLog throttles the accompanying warn
	// log to at most once per window so a flood cannot spam the logs.
	saturationCount   uint64
	lastSaturationLog time.Time
}

func newInMemoryRateLimiter(limit int) *inMemoryRateLimiter {
	return &inMemoryRateLimiter{
		windows: make(map[string]rateWindow),
		limit:   limit,
	}
}

func (l *inMemoryRateLimiter) allow(key string, now time.Time) (bool, int) {
	l.mu.Lock()
	allowed, retryAfter, logSaturation := l.allowLocked(key, now)
	saturationTotal := l.saturationCount
	l.mu.Unlock()

	if logSaturation {
		slog.Warn(
			"SECURITY: API rate-limiter window table is saturated; denying new clients (fail-closed). "+
				"This typically indicates a high-cardinality source-IP flood against the scan endpoints.",
			"max_entries", rateLimiterMaxEntries,
			"saturation_total", saturationTotal,
		)
	}

	return allowed, retryAfter
}

// allowLocked holds l.mu and returns (allowed, retryAfter, logSaturation). The
// third value asks the caller to emit a (throttled) saturation log outside the
// lock.
func (l *inMemoryRateLimiter) allowLocked(key string, now time.Time) (bool, int, bool) {
	l.evictStaleLocked(now)

	window, exists := l.windows[key]
	if !exists || now.Sub(window.start) >= rateLimitWindow {
		// A brand-new key would grow the table. When it is already full, fail
		// CLOSED (deny) rather than open. Returning "allowed" here let an
		// attacker who saturates the table — e.g. by rotating through many
		// source IPs — disable rate limiting for everyone, turning the
		// container-spawning scan endpoints into an unbounded resource sink.
		// Only brand-new keys are denied; already-tracked clients (handled
		// below) keep their window, so legitimate steady traffic is unaffected.
		if !exists && len(l.windows) >= rateLimiterMaxEntries {
			l.saturationCount++

			logNow := l.lastSaturationLog.IsZero() || now.Sub(l.lastSaturationLog) >= rateLimitWindow
			if logNow {
				l.lastSaturationLog = now
			}

			return false, int(rateLimitWindow.Seconds()), logNow
		}

		window = rateWindow{start: now, count: 0}
	}

	if window.count >= l.limit {
		retryAfter := int(rateLimitWindow.Seconds()) - int(now.Sub(window.start).Seconds())
		if retryAfter < 1 {
			retryAfter = 1
		}

		l.windows[key] = window

		return false, retryAfter, false
	}

	window.count++
	l.windows[key] = window

	return true, 0, false
}

func (l *inMemoryRateLimiter) evictStaleLocked(now time.Time) {
	for key, window := range l.windows {
		if now.Sub(window.start) >= rateLimitWindow {
			delete(l.windows, key)
		}
	}
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

type clientTokenBucket struct {
	tokens     float64
	lastUpdate time.Time
}

// clientTokenBucketLimiter protects the public container-spawning endpoints
// independently of the general API read limiter. It is intentionally local to
// one Server router; multi-instance deployments should enforce an equivalent
// shared limit at their edge.
type clientTokenBucketLimiter struct {
	mu                sync.Mutex
	buckets           map[string]clientTokenBucket
	requestsPerMinute float64
	burst             float64

	saturationCount   uint64
	lastSaturationLog time.Time
}

func newClientTokenBucketLimiter(requestsPerMinute, burst int) *clientTokenBucketLimiter {
	return &clientTokenBucketLimiter{
		buckets:           make(map[string]clientTokenBucket),
		requestsPerMinute: float64(requestsPerMinute),
		burst:             float64(burst),
	}
}

func newPublicSubmissionRateLimiterFromEnv() *clientTokenBucketLimiter {
	rpm := positiveIntEnv(publicSubmissionRateEnv, defaultPublicSubmissionRequestsPerMinute)
	burst := positiveIntEnv(publicSubmissionBurstEnv, defaultPublicSubmissionBurst)

	return newClientTokenBucketLimiter(rpm, burst)
}

func positiveIntEnv(name string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		slog.Warn("Ignoring invalid positive-integer rate-limit setting", "name", name, "value", raw)

		return fallback
	}

	return value
}

func (l *clientTokenBucketLimiter) allow(key string, now time.Time) (bool, int) {
	l.mu.Lock()
	allowed, retryAfter, logSaturation := l.allowLocked(key, now)
	saturationTotal := l.saturationCount
	l.mu.Unlock()

	if logSaturation {
		slog.Warn(
			"SECURITY: public-submission rate-limiter table is saturated; denying new clients (fail-closed)",
			"max_entries", rateLimiterMaxEntries,
			"saturation_total", saturationTotal,
		)
	}

	return allowed, retryAfter
}

func (l *clientTokenBucketLimiter) allowLocked(key string, now time.Time) (bool, int, bool) {
	// A bucket recovers fully within at most burst/rate minutes. Keeping it for
	// at least one minute avoids churn while bounding attacker-controlled keys.
	staleAfter := time.Duration(math.Ceil(l.burst/l.requestsPerMinute*float64(time.Minute))) * time.Nanosecond
	if staleAfter < time.Minute {
		staleAfter = time.Minute
	}

	for bucketKey, bucket := range l.buckets {
		if now.Sub(bucket.lastUpdate) >= staleAfter {
			delete(l.buckets, bucketKey)
		}
	}

	bucket, exists := l.buckets[key]
	if !exists {
		if len(l.buckets) >= rateLimiterMaxEntries {
			l.saturationCount++

			logNow := l.lastSaturationLog.IsZero() || now.Sub(l.lastSaturationLog) >= rateLimitWindow
			if logNow {
				l.lastSaturationLog = now
			}

			return false, int(rateLimitWindow.Seconds()), logNow
		}

		bucket = clientTokenBucket{tokens: l.burst, lastUpdate: now}
	}

	elapsedMinutes := now.Sub(bucket.lastUpdate).Minutes()
	if elapsedMinutes > 0 {
		bucket.tokens = math.Min(l.burst, bucket.tokens+(elapsedMinutes*l.requestsPerMinute))
	}

	bucket.lastUpdate = now

	if bucket.tokens < 1 {
		secondsPerToken := 60 / l.requestsPerMinute

		retryAfter := int(math.Ceil((1 - bucket.tokens) * secondsPerToken))
		if retryAfter < 1 {
			retryAfter = 1
		}

		l.buckets[key] = bucket

		return false, retryAfter, false
	}

	bucket.tokens--
	l.buckets[key] = bucket

	return true, 0, false
}

func publicSubmissionRateLimitMiddleware(
	limiter *clientTokenBucketLimiter,
	next http.HandlerFunc,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		allowed, retryAfter := limiter.allow(rateLimitKey(r), time.Now().UTC())
		if !allowed {
			detail := httputil.NewRateLimitError(
				retryAfter,
				strconv.Itoa(int(limiter.requestsPerMinute)),
				"1 minute",
			)
			detail.Details = "This client has exceeded the public scan submission limit."
			detail.Suggestion = "Wait for the Retry-After interval before submitting another scan."
			detail.Meta["burst"] = strconv.Itoa(int(limiter.burst))
			httputil.RespondStructuredError(w, http.StatusTooManyRequests, detail)

			return
		}

		next.ServeHTTP(w, r)
	}
}

// requestReadDeadlineMiddleware overrides the server-wide request-body
// deadline for endpoints that intentionally accept longer uploads. Keeping
// this route-scoped preserves the tighter default deadline everywhere else.
func requestReadDeadlineMiddleware(timeout time.Duration, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deadline := time.Now().Add(timeout)
		if err := http.NewResponseController(w).SetReadDeadline(deadline); err != nil &&
			!errors.Is(err, http.ErrNotSupported) {
			slog.WarnContext(r.Context(), "Failed to extend request read deadline", "error", err)
		}

		next.ServeHTTP(w, r)
	}
}

// trustedProxies caches the parsed PLATFORM_API_TRUSTED_PROXIES list. It is
// populated lazily on first use so t.Setenv in tests can change the value
// before the rate limiter reads it. Call resetTrustedProxiesForTest to force
// a reload.
var (
	trustedProxiesOnce sync.Once
	trustedProxiesVal  []*net.IPNet
)

func trustedProxies() []*net.IPNet {
	trustedProxiesOnce.Do(func() {
		trustedProxiesVal = loadTrustedProxies()
	})

	return trustedProxiesVal
}

func resetTrustedProxiesForTest() {
	trustedProxiesOnce = sync.Once{}
	trustedProxiesVal = nil
}

func loadTrustedProxies() []*net.IPNet {
	raw := strings.TrimSpace(os.Getenv("PLATFORM_API_TRUSTED_PROXIES"))
	if raw == "" {
		// The supplied Caddy topology connects over loopback. Trusting only
		// loopback by default gives public users independent limiter buckets
		// without allowing network clients to spoof forwarding headers.
		raw = defaultTrustedProxyCIDRs
	}

	var nets []*net.IPNet

	for _, part := range strings.Split(raw, ",") {
		cidr := strings.TrimSpace(part)
		if cidr == "" {
			continue
		}

		if !strings.Contains(cidr, "/") {
			if ip := net.ParseIP(cidr); ip != nil {
				if ip.To4() != nil {
					cidr += "/32"
				} else {
					cidr += "/128"
				}
			}
		}

		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			slog.Warn("Ignoring invalid CIDR in PLATFORM_API_TRUSTED_PROXIES", "value", cidr, "error", err)

			continue
		}

		nets = append(nets, ipNet)
	}

	return nets
}

func remoteAddrIsTrusted(remoteAddr string) bool {
	nets := trustedProxies()
	if len(nets) == 0 {
		return false
	}

	host, _, err := net.SplitHostPort(strings.TrimSpace(remoteAddr))
	if err != nil || host == "" {
		host = strings.TrimSpace(remoteAddr)
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}

	for _, n := range nets {
		if n.Contains(ip) {
			return true
		}
	}

	return false
}

func rateLimitKey(r *http.Request) string {
	if forwardedKey := trustedForwardedRateLimitKey(r); forwardedKey != "" {
		return forwardedKey
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

func trustedForwardedRateLimitKey(r *http.Request) string {
	if !remoteAddrIsTrusted(r.RemoteAddr) {
		return ""
	}

	forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if forwarded == "" {
		return ""
	}

	parts := strings.Split(forwarded, ",")

	var leftmostValid string

	// X-Forwarded-For is appended from left to right by proxies. Walk from the
	// trusted edge toward the client and select the first untrusted address so
	// a caller-supplied leftmost value cannot mint fresh limiter buckets.
	for index := len(parts) - 1; index >= 0; index-- {
		ip := parseForwardedIP(parts[index])
		if ip == nil {
			return ""
		}

		leftmostValid = ip.String()
		if !ipIsTrustedProxy(ip) {
			return leftmostValid
		}
	}

	return leftmostValid
}

func parseForwardedIP(value string) net.IP {
	value = strings.TrimSpace(value)
	if host, _, err := net.SplitHostPort(value); err == nil {
		value = host
	}

	return net.ParseIP(strings.Trim(value, "[]"))
}

func ipIsTrustedProxy(ip net.IP) bool {
	for _, network := range trustedProxies() {
		if network.Contains(ip) {
			return true
		}
	}

	return false
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

		done := make(chan timeoutHandlerResult, 1)
		tw := newTimeoutResponseWriter()

		go runTimedHandler(done, tw, r, next)

		select {
		case result := <-done:
			handleTimedResult(ctx, w, r, timeout, tw, result)
		case <-ctx.Done():
			// Timeout reached.
			tw.markTimedOut()
			writeTimeoutJSON(w, r, timeout)
		}
	}
}

func runTimedHandler(
	done chan<- timeoutHandlerResult,
	tw *timeoutResponseWriter,
	r *http.Request,
	next http.HandlerFunc,
) {
	result := timeoutHandlerResult{}

	defer func() {
		if recovered := recover(); recovered != nil {
			result.panicValue = recovered
			result.panicStack = debug.Stack()
		}

		done <- result
	}()

	next.ServeHTTP(tw, r)
}

func handleTimedResult(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	timeout time.Duration,
	tw *timeoutResponseWriter,
	result timeoutHandlerResult,
) {
	if result.panicValue != nil {
		slog.Error("HTTP handler panic", "panic", result.panicValue, "stack", string(result.panicStack))
		// Re-panic on net/http's serving goroutine so its recovery boundary
		// handles the failure. A panic in the worker goroutine would bypass
		// that boundary and terminate the process.
		panic(result.panicValue)
	}

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		writeTimeoutJSON(w, r, timeout)

		return
	}

	writeBufferedTimedResponse(w, tw)
}

func writeBufferedTimedResponse(w http.ResponseWriter, tw *timeoutResponseWriter) {
	code, header, body, overflowed := tw.snapshot()
	if overflowed {
		httputil.RespondError(w, http.StatusInternalServerError, "Response exceeded server buffer limit")

		return
	}

	for key, values := range header {
		w.Header()[key] = values
	}

	w.WriteHeader(code)

	if _, err := w.Write(body); err != nil {
		slog.Warn("Failed to write buffered response", "error", err)
	}
}

type timeoutHandlerResult struct {
	panicValue any
	panicStack []byte
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
	h          http.Header
	buf        bytes.Buffer
	mu         sync.Mutex
	code       int
	wrote      bool
	timed      bool
	overflowed bool
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

	if len(b) > maxTimeoutResponseBytes-tw.buf.Len() {
		tw.overflowed = true

		return 0, http.ErrContentLength
	}

	return tw.buf.Write(b)
}

func (tw *timeoutResponseWriter) markTimedOut() {
	tw.mu.Lock()
	tw.timed = true
	tw.mu.Unlock()
}

func (tw *timeoutResponseWriter) snapshot() (code int, header http.Header, body []byte, overflowed bool) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	code = tw.code
	if !tw.wrote {
		code = http.StatusOK
	}

	header = tw.h.Clone()
	body = append([]byte(nil), tw.buf.Bytes()...)
	overflowed = tw.overflowed

	return code, header, body, overflowed
}
