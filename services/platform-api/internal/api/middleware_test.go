package api

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

type flusherRecorder struct {
	*httptest.ResponseRecorder
	flushed bool
}

func (f *flusherRecorder) Flush() {
	f.flushed = true
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

	if limiter.saturationCount == 0 {
		t.Fatal("expected saturationCount to be incremented on a fail-closed denial")
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
