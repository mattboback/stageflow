package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mattboback/stageflow/libs/go/httputil"
)

type flusherRecorder struct {
	*httptest.ResponseRecorder
	flushed bool
}

func (f *flusherRecorder) Flush() {
	f.flushed = true
}

type deadlineRecorder struct {
	*httptest.ResponseRecorder
	readDeadline time.Time
}

func (r *deadlineRecorder) SetReadDeadline(deadline time.Time) error {
	r.readDeadline = deadline

	return nil
}

func TestRequestReadDeadlineMiddlewareExtendsUploadDeadline(t *testing.T) {
	t.Parallel()

	started := time.Now()
	called := false
	recorder := &deadlineRecorder{ResponseRecorder: httptest.NewRecorder()}
	handler := requestReadDeadlineMiddleware(uploadRequestTimeout, func(http.ResponseWriter, *http.Request) {
		called = true
	})

	handler(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/jobs/zip", http.NoBody))

	if !called {
		t.Fatal("wrapped upload handler was not called")
	}

	minimum := started.Add(uploadRequestTimeout)
	maximum := time.Now().Add(uploadRequestTimeout)

	if recorder.readDeadline.Before(minimum) || recorder.readDeadline.After(maximum) {
		t.Fatalf(
			"read deadline = %v, want between %v and %v",
			recorder.readDeadline,
			minimum,
			maximum,
		)
	}
}

func TestRouterExtendsReadDeadlineOnlyForZIPUploads(t *testing.T) {
	server, _, _ := newTestServer(t)
	router := server.Router()

	zipRecorder := &deadlineRecorder{ResponseRecorder: httptest.NewRecorder()}
	router.ServeHTTP(
		zipRecorder,
		httptest.NewRequest(http.MethodPost, "/api/v1/jobs/zip", http.NoBody),
	)

	if zipRecorder.readDeadline.IsZero() {
		t.Fatal("ZIP route did not extend its request read deadline")
	}

	urlRecorder := &deadlineRecorder{ResponseRecorder: httptest.NewRecorder()}
	router.ServeHTTP(
		urlRecorder,
		httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls", http.NoBody),
	)

	if !urlRecorder.readDeadline.IsZero() {
		t.Fatalf("non-upload route changed its read deadline to %v", urlRecorder.readDeadline)
	}
}

func TestCORSMiddleware(t *testing.T) {
	t.Setenv("PLATFORM_API_CORS_ALLOW_ORIGINS", "https://example.com")

	handler := corsMiddleware(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("Origin", "https://example.com")

	rr := httptest.NewRecorder()

	handler(rr, req)

	// Check CORS headers
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Errorf("Expected Access-Control-Allow-Origin to echo origin, got %q", got)
	}

	if rr.Header().Get("Access-Control-Allow-Methods") != "GET, POST, PATCH, DELETE, OPTIONS" {
		t.Error("Expected Access-Control-Allow-Methods header")
	}

	if rr.Header().Get("Access-Control-Allow-Headers") != "Content-Type, Authorization, X-Api-Key" {
		t.Error("Expected Access-Control-Allow-Headers header")
	}

	if rr.Header().Get("Vary") != "Origin" {
		t.Error("Expected Vary header to include Origin")
	}
}

func TestCORSMiddleware_NoConfiguredOrigins(t *testing.T) {
	t.Setenv("PLATFORM_API_CORS_ALLOW_ORIGINS", "")

	handler := corsMiddleware(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("Origin", "https://example.com")

	rr := httptest.NewRecorder()

	handler(rr, req)

	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no Access-Control-Allow-Origin header, got %q", got)
	}
}

func TestCORSMiddlewareOptions(t *testing.T) {
	handler := corsMiddleware(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodOptions, "/test", http.NoBody)
	rr := httptest.NewRecorder()

	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200 for OPTIONS, got %d", rr.Code)
	}
}

func TestLoggingMiddleware(t *testing.T) {
	handler := loggingMiddleware(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rr := httptest.NewRecorder()

	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestLoggingResponseWriter(t *testing.T) {
	rr := httptest.NewRecorder()
	lrw := &loggingResponseWriter{
		ResponseWriter: rr,
		statusCode:     http.StatusOK,
	}

	lrw.WriteHeader(http.StatusNotFound)

	if lrw.statusCode != http.StatusNotFound {
		t.Errorf("Expected status code 404, got %d", lrw.statusCode)
	}
}

func TestLoggingResponseWriter_ImplementsFlusher(t *testing.T) {
	rr := &flusherRecorder{ResponseRecorder: httptest.NewRecorder()}
	lrw := &loggingResponseWriter{
		ResponseWriter: rr,
		statusCode:     http.StatusOK,
	}

	flusher, ok := any(lrw).(http.Flusher)
	if !ok {
		t.Fatalf("expected loggingResponseWriter to implement http.Flusher")
	}

	flusher.Flush()

	if !rr.flushed {
		t.Fatalf("expected Flush() to be forwarded to underlying ResponseWriter")
	}
}

func TestAPIKeyMiddleware_NoToken_AllowsRequest(t *testing.T) {
	handler := apiKeyMiddleware(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rr := httptest.NewRecorder()

	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestAPIKeyMiddleware_WithToken_RejectsMissingKey(t *testing.T) {
	t.Setenv("PLATFORM_API_TOKEN", "secret")

	handler := apiKeyMiddleware(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rr := httptest.NewRecorder()

	handler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestAPIKeyMiddleware_WithToken_AllowsMatchingKey(t *testing.T) {
	t.Setenv("PLATFORM_API_TOKEN", "secret")

	handler := apiKeyMiddleware(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("X-Api-Key", "secret")

	rr := httptest.NewRecorder()

	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestAPIKeyMiddleware_WithToken_RejectsWrongKey(t *testing.T) {
	t.Setenv("PLATFORM_API_TOKEN", "secret")

	handler := apiKeyMiddleware(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("X-Api-Key", "wrong")

	rr := httptest.NewRecorder()

	handler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestAPIKeyMiddleware_AuthDisabled_AllowsMissingKey(t *testing.T) {
	// Regression for F4: PLATFORM_API_AUTH_DISABLED=true must bypass auth at the
	// request path even when PLATFORM_API_TOKEN is set (the shipped local default).
	t.Setenv("PLATFORM_API_TOKEN", "secret")
	t.Setenv("PLATFORM_API_AUTH_DISABLED", "true")

	handler := apiKeyMiddleware(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rr := httptest.NewRecorder()

	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 with PLATFORM_API_AUTH_DISABLED=true and no key, got %d", rr.Code)
	}
}

func TestValidateAuthConfig_MissingTokenFails(t *testing.T) {
	t.Setenv("PLATFORM_API_TOKEN", "")
	t.Setenv("PLATFORM_API_AUTH_DISABLED", "")

	if err := ValidateAuthConfig(); err == nil {
		t.Fatalf("expected error when PLATFORM_API_TOKEN is unset")
	}
}

func TestValidateAuthConfig_ExplicitDisableAllowed(t *testing.T) {
	t.Setenv("PLATFORM_API_TOKEN", "")
	t.Setenv("PLATFORM_API_AUTH_DISABLED", "true")

	if err := ValidateAuthConfig(); err != nil {
		t.Fatalf("expected nil error with PLATFORM_API_AUTH_DISABLED=true, got %v", err)
	}
}

func TestValidateAuthConfig_TokenSet(t *testing.T) {
	t.Setenv("PLATFORM_API_TOKEN", "secret")
	t.Setenv("PLATFORM_API_AUTH_DISABLED", "")

	if err := ValidateAuthConfig(); err != nil {
		t.Fatalf("expected nil error with token set, got %v", err)
	}
}

func TestRateLimit_XForwardedFor_UntrustedIgnored(t *testing.T) {
	original := apiRateLimiter
	apiRateLimiter = newInMemoryRateLimiter(1)

	t.Cleanup(func() {
		apiRateLimiter = original

		resetTrustedProxiesForTest()
	})

	t.Setenv("PLATFORM_API_TRUSTED_PROXIES", "")
	resetTrustedProxiesForTest()

	handler := rateLimitMiddleware(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// First request from RemoteAddr fills the bucket.
	req1 := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req1.RemoteAddr = "10.0.0.1:1234"
	req1.Header.Set("X-Forwarded-For", "1.1.1.1")

	rr1 := httptest.NewRecorder()

	handler(rr1, req1)

	if rr1.Code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", rr1.Code)
	}

	// Second request tries to escape the bucket via a spoofed XFF header.
	// Because 10.0.0.1 is not trusted, XFF must be ignored and the key
	// must remain 10.0.0.1.
	req2 := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req2.RemoteAddr = "10.0.0.1:1234"
	req2.Header.Set("X-Forwarded-For", "2.2.2.2")

	rr2 := httptest.NewRecorder()

	handler(rr2, req2)

	if rr2.Code != http.StatusTooManyRequests {
		t.Fatalf("second request with spoofed XFF: expected 429, got %d", rr2.Code)
	}
}

func TestRateLimitKey_DefaultLoopbackProxyUsesForwardedClient(t *testing.T) {
	t.Setenv("PLATFORM_API_TRUSTED_PROXIES", "")
	resetTrustedProxiesForTest()
	t.Cleanup(resetTrustedProxiesForTest)

	first := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls/anonymous", http.NoBody)
	first.RemoteAddr = "127.0.0.1:41001"
	first.Header.Set("X-Forwarded-For", "198.51.100.10")

	second := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls/anonymous", http.NoBody)
	second.RemoteAddr = "127.0.0.1:41002"
	second.Header.Set("X-Forwarded-For", "198.51.100.11")

	if firstKey, secondKey := rateLimitKey(first), rateLimitKey(second); firstKey != "198.51.100.10" ||
		secondKey != "198.51.100.11" {
		t.Fatalf("loopback proxy keys = %q, %q", firstKey, secondKey)
	}
}

func TestRateLimitKey_TrustedProxyRejectsSpoofedLeftmostForwardingValue(t *testing.T) {
	t.Setenv("PLATFORM_API_TRUSTED_PROXIES", "10.0.0.0/8")
	resetTrustedProxiesForTest()
	t.Cleanup(resetTrustedProxiesForTest)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls/anonymous", http.NoBody)
	req.RemoteAddr = "10.0.0.1:41001"
	req.Header.Set("X-Forwarded-For", "192.0.2.200, 203.0.113.25")

	if got := rateLimitKey(req); got != "203.0.113.25" {
		t.Fatalf("rate-limit key = %q, want rightmost untrusted client", got)
	}
}

func TestRateLimitKey_InvalidForwardingChainFallsBackToProxy(t *testing.T) {
	t.Setenv("PLATFORM_API_TRUSTED_PROXIES", "10.0.0.0/8")
	resetTrustedProxiesForTest()
	t.Cleanup(resetTrustedProxiesForTest)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls/anonymous", http.NoBody)
	req.RemoteAddr = "10.0.0.1:41001"
	req.Header.Set("X-Forwarded-For", "not-an-ip")

	if got := rateLimitKey(req); got != "10.0.0.1" {
		t.Fatalf("rate-limit key = %q, want immediate proxy fallback", got)
	}
}

func TestRateLimit_XForwardedFor_TrustedHonored(t *testing.T) {
	original := apiRateLimiter
	apiRateLimiter = newInMemoryRateLimiter(1)

	t.Cleanup(func() {
		apiRateLimiter = original

		resetTrustedProxiesForTest()
	})

	t.Setenv("PLATFORM_API_TRUSTED_PROXIES", "10.0.0.0/8")
	resetTrustedProxiesForTest()

	handler := rateLimitMiddleware(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Two requests from different upstream IPs via a trusted proxy should
	// each get their own bucket.
	req1 := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req1.RemoteAddr = "10.0.0.1:1234"
	req1.Header.Set("X-Forwarded-For", "1.1.1.1")

	rr1 := httptest.NewRecorder()

	handler(rr1, req1)

	if rr1.Code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", rr1.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req2.RemoteAddr = "10.0.0.1:1234"
	req2.Header.Set("X-Forwarded-For", "2.2.2.2")

	rr2 := httptest.NewRecorder()

	handler(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("second request with distinct trusted XFF: expected 200, got %d", rr2.Code)
	}
}

func TestRateLimitMiddleware_ReturnsTooManyRequests(t *testing.T) {
	original := apiRateLimiter
	apiRateLimiter = newInMemoryRateLimiter(1)

	t.Cleanup(func() {
		apiRateLimiter = original
	})

	handler := rateLimitMiddleware(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req1 := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req1.RemoteAddr = "127.0.0.1:1234"
	rr1 := httptest.NewRecorder()
	handler(rr1, req1)

	if rr1.Code != http.StatusOK {
		t.Fatalf("expected first request 200, got %d", rr1.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req2.RemoteAddr = "127.0.0.1:1234"
	rr2 := httptest.NewRecorder()
	handler(rr2, req2)

	if rr2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second request 429, got %d", rr2.Code)
	}
}

func TestClientTokenBucketLimiterEnforcesBurstAndRefill(t *testing.T) {
	limiter := newClientTokenBucketLimiter(6, 3)
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)

	for i := range 3 {
		if allowed, retryAfter := limiter.allow("client-a", now); !allowed || retryAfter != 0 {
			t.Fatalf("burst request %d = (%v, %d), want (true, 0)", i+1, allowed, retryAfter)
		}
	}

	if allowed, retryAfter := limiter.allow("client-a", now); allowed || retryAfter != 10 {
		t.Fatalf("request past burst = (%v, %d), want (false, 10)", allowed, retryAfter)
	}

	if allowed, retryAfter := limiter.allow("client-a", now.Add(10*time.Second)); !allowed || retryAfter != 0 {
		t.Fatalf("request after one-token refill = (%v, %d), want (true, 0)", allowed, retryAfter)
	}

	if allowed, _ := limiter.allow("client-b", now); !allowed {
		t.Fatal("expected a different client to have an independent burst")
	}
}

func TestPublicSubmissionRateLimiterLoadsConfig(t *testing.T) {
	t.Setenv(publicSubmissionRateEnv, "12")
	t.Setenv(publicSubmissionBurstEnv, "4")

	limiter := newPublicSubmissionRateLimiterFromEnv()
	if limiter.requestsPerMinute != 12 {
		t.Fatalf("requestsPerMinute = %v, want 12", limiter.requestsPerMinute)
	}

	if limiter.burst != 4 {
		t.Fatalf("burst = %v, want 4", limiter.burst)
	}
}

func TestPublicSubmissionRateLimitMiddlewareReturnsRetryAfter(t *testing.T) {
	limiter := newClientTokenBucketLimiter(6, 1)
	handler := publicSubmissionRateLimitMiddleware(limiter, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	first := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls/anonymous", http.NoBody)
	first.RemoteAddr = "127.0.0.1:1234"
	firstResponse := httptest.NewRecorder()
	handler(firstResponse, first)

	if firstResponse.Code != http.StatusCreated {
		t.Fatalf("first request: expected 201, got %d", firstResponse.Code)
	}

	second := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/urls/browser-auth", http.NoBody)
	second.RemoteAddr = "127.0.0.1:5678"
	secondResponse := httptest.NewRecorder()
	handler(secondResponse, second)

	if secondResponse.Code != http.StatusTooManyRequests {
		t.Fatalf("second request: expected 429, got %d", secondResponse.Code)
	}

	if got := secondResponse.Header().Get("Retry-After"); got == "" {
		t.Fatal("expected Retry-After header")
	}

	var response httputil.ErrorResponse
	if err := json.NewDecoder(secondResponse.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Error.Code != httputil.ErrCodeRateLimitExceeded {
		t.Fatalf("error code = %q, want %q", response.Error.Code, httputil.ErrCodeRateLimitExceeded)
	}

	if response.Error.Meta["burst"] != "1" {
		t.Fatalf("burst metadata = %q, want 1", response.Error.Meta["burst"])
	}
}

func TestValidateCORSConfig_WildcardWithAuthFails(t *testing.T) {
	t.Setenv("PLATFORM_API_CORS_ALLOW_ORIGINS", "https://stageflow.org,*")
	t.Setenv("PLATFORM_API_AUTH_DISABLED", "")

	if err := ValidateCORSConfig(); err == nil {
		t.Fatal("expected error for wildcard CORS while auth is enabled")
	}
}

func TestValidateCORSConfig_WildcardWithAuthDisabledWarnsOnly(t *testing.T) {
	t.Setenv("PLATFORM_API_CORS_ALLOW_ORIGINS", "*")
	t.Setenv("PLATFORM_API_AUTH_DISABLED", "true")

	if err := ValidateCORSConfig(); err != nil {
		t.Fatalf("expected no error for wildcard CORS when auth is disabled, got %v", err)
	}
}

func TestValidateCORSConfig_ExplicitOriginsOK(t *testing.T) {
	t.Setenv("PLATFORM_API_CORS_ALLOW_ORIGINS", "https://stageflow.org,https://www.stageflow.org")
	t.Setenv("PLATFORM_API_AUTH_DISABLED", "")

	if err := ValidateCORSConfig(); err != nil {
		t.Fatalf("expected no error for explicit origins, got %v", err)
	}
}

func TestRateLimiter_FailsClosedWhenTableSaturated(t *testing.T) {
	limiter := newInMemoryRateLimiter(100)
	now := time.Now().UTC()

	// Fill the window table to capacity with distinct, in-window keys.
	for i := range rateLimiterMaxEntries {
		allowed, _ := limiter.allow("filler-"+strconv.Itoa(i), now)
		if !allowed {
			t.Fatalf("filler %d should be allowed while filling the table", i)
		}
	}

	// A brand-new key must now be DENIED (fail closed), not silently allowed.
	allowed, retryAfter := limiter.allow("new-client", now)
	if allowed {
		t.Fatal("expected new client to be denied when the table is saturated (fail closed)")
	}

	if retryAfter < 1 {
		t.Fatalf("expected a positive Retry-After on saturation, got %d", retryAfter)
	}

	if limiter.saturation.count == 0 {
		t.Fatal("expected the saturation counter to be incremented on a fail-closed denial")
	}

	// An already-tracked key must keep being served from its existing window,
	// so saturation never starves established legitimate clients.
	if ok, _ := limiter.allow("filler-0", now); !ok {
		t.Fatal("expected an already-tracked key to remain allowed under saturation")
	}
}

func TestTimeoutMiddleware_AllowsFastHandler(t *testing.T) {
	handler := timeoutMiddleware(50*time.Millisecond, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		if _, err := w.Write([]byte(`{"ok":true}`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rr := httptest.NewRecorder()

	handler(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected %d, got %d", http.StatusCreated, rr.Code)
	}

	if got := rr.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", got)
	}
}

func TestTimeoutMiddleware_TimesOutAndIgnoresLateWrites(t *testing.T) {
	lateWriteDone := make(chan struct{})

	handler := timeoutMiddleware(10*time.Millisecond, func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(50 * time.Millisecond)

		// Write after timeout - expected to fail with timeout error
		_, _ = w.Write([]byte("late"))

		close(lateWriteDone)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rr := httptest.NewRecorder()

	handler(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected %d, got %d", http.StatusServiceUnavailable, rr.Code)
	}

	if got := rr.Body.String(); strings.Contains(got, "late") {
		t.Fatalf("expected late writes to be ignored, got body %q", got)
	}

	select {
	case <-lateWriteDone:
		// ok
	case <-time.After(250 * time.Millisecond):
		t.Fatalf("expected handler goroutine to finish")
	}
}

func TestTimeoutMiddleware_RepanicsOnServingGoroutine(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("handler exploded")
	handler := timeoutMiddleware(time.Second, func(http.ResponseWriter, *http.Request) {
		panic(sentinel)
	})

	defer func() {
		recovered := recover()
		if !errors.Is(asError(recovered), sentinel) {
			t.Fatalf("recovered panic = %v, want %v", recovered, sentinel)
		}
	}()

	handler(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/panic", http.NoBody))
}

func asError(value any) error {
	err, _ := value.(error)

	return err
}
