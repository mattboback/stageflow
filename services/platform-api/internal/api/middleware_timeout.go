package api

import (
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

const timeoutBody = `{"error":"request timeout","code":"REQUEST_TIMEOUT"}`

// requestReadDeadlineMiddleware overrides the server-wide request-body
// deadline for endpoints that intentionally accept longer uploads. Keeping
// this route-scoped preserves the tighter default deadline everywhere else.
// It must wrap the timeout, whose writer does not support ResponseController.
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

// timeoutMiddleware bounds a handler with http.TimeoutHandler, which buffers
// its output and answers 503 with timeoutBody if the deadline passes first.
func timeoutMiddleware(timeout time.Duration, next http.HandlerFunc) http.HandlerFunc {
	timed := http.TimeoutHandler(logLatePanic(timeout, next), timeout, timeoutBody)

	return func(w http.ResponseWriter, r *http.Request) {
		timed.ServeHTTP(timeoutJSONWriter{w}, r)
	}
}

// timeoutJSONWriter labels TimeoutHandler's 503 body, which it writes without a
// Content-Type. Responses that set their own type keep it.
type timeoutJSONWriter struct {
	http.ResponseWriter
}

func (w timeoutJSONWriter) WriteHeader(code int) {
	if code == http.StatusServiceUnavailable && w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/json")
	}

	w.ResponseWriter.WriteHeader(code)
}

// logLatePanic records a panic raised after the deadline. TimeoutHandler has
// already answered by then and would drop it silently. Earlier panics are
// re-raised, and TimeoutHandler re-panics them on net/http's serving goroutine.
func logLatePanic(timeout time.Duration, next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}

			if r.Context().Err() == nil {
				panic(recovered)
			}

			slog.Error(
				"HTTP handler panicked after its deadline elapsed; response was already sent",
				"panic", recovered,
				"method", r.Method,
				"path", r.URL.Path,
				"timeout", timeout.String(),
				"stack", string(debug.Stack()),
			)
		}()

		next(w, r)
	})
}
