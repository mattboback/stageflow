package api

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"
	"sync"
	"time"

	"github.com/mattboback/stageflow/libs/go/httputil"
)

// Request timeouts.
//
// A handler that outruns its deadline must not write a partial body after the
// timeout response has been sent, so output is buffered in a timeoutResponseWriter
// and only flushed if the handler finishes in time.

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
