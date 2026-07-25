package api

import (
	"log/slog"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mattboback/stageflow/libs/go/httputil"
)

// Rate limiting for the public API.
//
// Two limiters with different shapes: a fixed window for general API reads, and a
// token bucket for the container-spawning submission endpoints, which are far more
// expensive per request. Both fail closed when their key table fills — see
// saturationTracker for why.

// saturationTracker records fail-closed denials caused by a full key table, and
// throttles the accompanying warning to at most one per rate-limit window.
//
// Both limiters below bound their key table to rateLimiterMaxEntries and deny
// brand-new keys once it is full, so an attacker rotating source IPs cannot evict
// everyone else's entry and switch rate limiting off. That decision is worth
// logging, but under the flood that causes it, logging every request would be its
// own denial of service. Shared because both limiters had identical copies of the
// counter, the timestamp, and the throttling arithmetic.
type saturationTracker struct {
	count   uint64
	lastLog time.Time
}

// recordLocked notes one saturation denial and reports whether the caller should
// emit a warning. The caller must hold the limiter's mutex; the warning itself
// belongs outside it.
func (t *saturationTracker) recordLocked(now time.Time) bool {
	t.count++

	if t.lastLog.IsZero() || now.Sub(t.lastLog) >= rateLimitWindow {
		t.lastLog = now

		return true
	}

	return false
}

// warnSaturated emits the limiter-specific message with the shared fields.
func warnSaturated(message string, saturationTotal uint64) {
	slog.Warn(
		message,
		"max_entries", rateLimiterMaxEntries,
		"saturation_total", saturationTotal,
	)
}

type rateWindow struct {
	start time.Time
	count int
}

type inMemoryRateLimiter struct {
	mu      sync.Mutex
	windows map[string]rateWindow
	limit   int

	// lastEvict throttles the full-table sweep; see evictStaleLocked.
	lastEvict time.Time

	saturation saturationTracker
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
	saturationTotal := l.saturation.count
	l.mu.Unlock()

	if logSaturation {
		warnSaturated(
			"SECURITY: API rate-limiter window table is saturated; denying new clients (fail-closed). "+
				"This typically indicates a high-cardinality source-IP flood against the scan endpoints.",
			saturationTotal,
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
			logNow := l.saturation.recordLocked(now)

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

// evictStaleLocked reclaims windows that have expired, at most once per
// rate-limit window.
//
// The throttle matters: this walks the entire key table while holding l.mu, so
// running it per request made the mitigation O(n) under exactly the condition it
// defends against -- a high-cardinality source-IP flood driving the table toward
// rateLimiterMaxEntries. Every request would pay a 10,000-entry scan under the
// lock, and the rate limiter becomes the bottleneck it was added to prevent.
//
// Once per window is the right cadence rather than a compromise. allowLocked
// already resets an expired window inline for any key that comes back, so the
// sweep only ever reclaims keys that never return -- attacker-rotated addresses.
// Deferring that by up to one window leaves at most two windows of dead keys in
// the table, and errs toward fail-closed, which is the safe direction here.
func (l *inMemoryRateLimiter) evictStaleLocked(now time.Time) {
	if !l.lastEvict.IsZero() && now.Sub(l.lastEvict) < rateLimitWindow {
		return
	}

	l.lastEvict = now

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

	// lastEvict throttles the full-table sweep; see evictStaleLocked.
	lastEvict time.Time

	saturation saturationTracker
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
	saturationTotal := l.saturation.count
	l.mu.Unlock()

	if logSaturation {
		warnSaturated(
			"SECURITY: public-submission rate-limiter table is saturated; denying new clients (fail-closed)",
			saturationTotal,
		)
	}

	return allowed, retryAfter
}

// evictStaleLocked reclaims fully-recovered buckets, at most once per rate-limit
// window. Same reasoning as the fixed-window limiter's: this walks the whole table
// under l.mu, so running it per request turned the flood defense into an O(n) cost
// per request precisely when the table was largest.
func (l *clientTokenBucketLimiter) evictStaleLocked(now time.Time) {
	if !l.lastEvict.IsZero() && now.Sub(l.lastEvict) < rateLimitWindow {
		return
	}

	l.lastEvict = now

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
}

func (l *clientTokenBucketLimiter) allowLocked(key string, now time.Time) (bool, int, bool) {
	l.evictStaleLocked(now)

	bucket, exists := l.buckets[key]
	if !exists {
		if len(l.buckets) >= rateLimiterMaxEntries {
			logNow := l.saturation.recordLocked(now)

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
