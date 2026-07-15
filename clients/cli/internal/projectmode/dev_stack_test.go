package projectmode

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

func TestMergeEnv(t *testing.T) {
	base := []string{"A=1", "B=2"}
	overlay := map[string]string{
		"B": "3",
		"C": "4",
	}

	got := mergeEnv(base, overlay)

	gotMap := map[string]string{}

	for _, item := range got {
		k, v, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}

		gotMap[k] = v
	}

	if gotMap["A"] != "1" {
		t.Fatalf("expected A=1, got %q", gotMap["A"])
	}

	if gotMap["B"] != "3" {
		t.Fatalf("expected B=3, got %q", gotMap["B"])
	}

	if gotMap["C"] != "4" {
		t.Fatalf("expected C=4, got %q", gotMap["C"])
	}
}

func TestParseSignal(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    syscall.Signal
		wantErr bool
	}{
		{name: "SIGINT", raw: "SIGINT", want: syscall.SIGINT},
		{name: "INT", raw: "INT", want: syscall.SIGINT},
		{name: "sigint", raw: "sigint", want: syscall.SIGINT},
		{name: "SIGTERM", raw: "SIGTERM", want: syscall.SIGTERM},
		{name: "TERM", raw: "TERM", want: syscall.SIGTERM},
		{name: "sigterm", raw: "sigterm", want: syscall.SIGTERM},
		{name: "SIGKILL", raw: "SIGKILL", want: syscall.SIGKILL},
		{name: "KILL", raw: "KILL", want: syscall.SIGKILL},
		{name: "sigkill", raw: "sigkill", want: syscall.SIGKILL},
		{name: "unsupported", raw: "SIGHUP", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSignal(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseSignal(%q) err = nil, want non-nil", tt.raw)
				}

				return
			}

			if err != nil {
				t.Fatalf("parseSignal(%q) err = %v, want nil", tt.raw, err)
			}

			if got != tt.want {
				t.Fatalf("parseSignal(%q) = %v, want %v", tt.raw, got, tt.want)
			}
		})
	}
}

func TestWaitForReady(t *testing.T) {
	t.Run("ready immediately", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		proc := &RunningProcess{waitCh: make(chan error, 1)}
		cfg := DevReadyConfig{URL: srv.URL, Timeout: "200ms", Interval: "10ms"}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := WaitForReady(ctx, srv.Client(), proc, cfg, ioDiscard{}); err != nil {
			t.Fatalf("WaitForReady error: %v", err)
		}
	})

	t.Run("becomes ready", func(t *testing.T) {
		var calls atomic.Int32

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			call := calls.Add(1)
			if call == 1 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		proc := &RunningProcess{waitCh: make(chan error, 1)}
		cfg := DevReadyConfig{URL: srv.URL, Timeout: "200ms", Interval: "10ms"}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := WaitForReady(ctx, srv.Client(), proc, cfg, ioDiscard{}); err != nil {
			t.Fatalf("WaitForReady error: %v", err)
		}

		if calls.Load() < 2 {
			t.Fatalf("expected at least 2 readiness checks, got %d", calls.Load())
		}
	})

	t.Run("timeout", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer srv.Close()

		proc := &RunningProcess{waitCh: make(chan error, 1)}
		cfg := DevReadyConfig{URL: srv.URL, Timeout: "80ms", Interval: "10ms"}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		err := WaitForReady(ctx, srv.Client(), proc, cfg, ioDiscard{})
		if err == nil {
			t.Fatalf("WaitForReady err = nil, want non-nil")
		}

		if !strings.Contains(err.Error(), "not ready within") {
			t.Fatalf("WaitForReady err = %q, want to contain %q", err.Error(), "not ready within")
		}
	})

	t.Run("process exits early", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer srv.Close()

		proc := &RunningProcess{waitCh: make(chan error, 1)}
		proc.waitCh <- errors.New("boom")

		cfg := DevReadyConfig{URL: srv.URL, Timeout: "200ms", Interval: "10ms"}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		err := WaitForReady(ctx, srv.Client(), proc, cfg, ioDiscard{})
		if err == nil {
			t.Fatalf("WaitForReady err = nil, want non-nil")
		}

		if !strings.Contains(err.Error(), "exited before ready") {
			t.Fatalf("WaitForReady err = %q, want to contain %q", err.Error(), "exited before ready")
		}

		if !strings.Contains(err.Error(), "boom") {
			// The wrapped error should include the process failure.
			t.Fatalf("WaitForReady err = %q, want to include process error", err.Error())
		}
	})
}

// ioDiscard is a tiny io.Writer without pulling in io.Discard for older toolchains.
type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
