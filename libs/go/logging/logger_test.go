package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestNew(t *testing.T) {
	logger := New(&Config{
		Level:       slog.LevelInfo,
		ServiceName: "test-service",
	})

	if logger == nil {
		t.Fatal("expected logger to be non-nil")
	}
}

func TestNewDefault(t *testing.T) {
	logger := NewDefault("test-service")
	if logger == nil {
		t.Fatal("expected logger to be non-nil")
	}
}

func TestNewWithNilConfig(t *testing.T) {
	logger := New(nil)
	if logger == nil {
		t.Fatal("expected logger to be non-nil with nil config")
	}
}

func TestContextHelpers(t *testing.T) {
	ctx := context.Background()

	// Test WithJobID
	ctx = WithJobID(ctx, "job-123")
	if got := JobID(ctx); got != "job-123" {
		t.Errorf("JobID() = %q, want %q", got, "job-123")
	}

	// Test WithRequestID
	ctx = WithRequestID(ctx, "req-456")
	if got := RequestID(ctx); got != "req-456" {
		t.Errorf("RequestID() = %q, want %q", got, "req-456")
	}

	// Test WithRunID
	ctx = WithRunID(ctx, "run-789")
	if got := RunID(ctx); got != "run-789" {
		t.Errorf("RunID() = %q, want %q", got, "run-789")
	}

	// Test WithScanner
	ctx = WithScanner(ctx, "axe")
	if got := Scanner(ctx); got != "axe" {
		t.Errorf("Scanner() = %q, want %q", got, "axe")
	}

	// Test WithComponent
	ctx = WithComponent(ctx, "orchestrator")
	if got := Component(ctx); got != "orchestrator" {
		t.Errorf("Component() = %q, want %q", got, "orchestrator")
	}
}

func TestContextHelpersEmptyContext(t *testing.T) {
	ctx := context.Background()

	if got := JobID(ctx); got != "" {
		t.Errorf("JobID() = %q, want empty string", got)
	}

	if got := RequestID(ctx); got != "" {
		t.Errorf("RequestID() = %q, want empty string", got)
	}

	if got := RunID(ctx); got != "" {
		t.Errorf("RunID() = %q, want empty string", got)
	}

	if got := Scanner(ctx); got != "" {
		t.Errorf("Scanner() = %q, want empty string", got)
	}

	if got := Component(ctx); got != "" {
		t.Errorf("Component() = %q, want empty string", got)
	}
}

func TestFromContext(t *testing.T) {
	ctx := context.Background()
	ctx = WithJobID(ctx, "job-123")
	ctx = WithRequestID(ctx, "req-456")
	ctx = WithRunID(ctx, "run-789")
	ctx = WithScanner(ctx, "lighthouse")

	attrs := FromContext(ctx)
	if len(attrs) != 4 {
		t.Errorf("FromContext() returned %d attrs, want 4", len(attrs))
	}

	// Check that expected attributes are present
	attrMap := make(map[string]string)
	for _, attr := range attrs {
		attrMap[attr.Key] = attr.Value.String()
	}

	if attrMap["job_id"] != "job-123" {
		t.Errorf("job_id = %q, want %q", attrMap["job_id"], "job-123")
	}

	if attrMap["request_id"] != "req-456" {
		t.Errorf("request_id = %q, want %q", attrMap["request_id"], "req-456")
	}

	if attrMap["run_id"] != "run-789" {
		t.Errorf("run_id = %q, want %q", attrMap["run_id"], "run-789")
	}

	if attrMap["scanner_type"] != "lighthouse" {
		t.Errorf("scanner_type = %q, want %q", attrMap["scanner_type"], "lighthouse")
	}
}

func TestFromContextEmpty(t *testing.T) {
	ctx := context.Background()

	attrs := FromContext(ctx)
	if len(attrs) != 0 {
		t.Errorf("FromContext() returned %d attrs for empty context, want 0", len(attrs))
	}
}

func TestLWithContext(t *testing.T) {
	// Set up a buffer to capture log output
	var buf bytes.Buffer

	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	ctx := context.Background()
	ctx = WithJobID(ctx, "job-test")
	ctx = WithScanner(ctx, "axe")

	L(ctx).Info("test message")

	// Parse the JSON output
	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to parse log output: %v", err)
	}

	if logEntry["msg"] != "test message" {
		t.Errorf("msg = %q, want %q", logEntry["msg"], "test message")
	}

	if logEntry["job_id"] != "job-test" {
		t.Errorf("job_id = %v, want %q", logEntry["job_id"], "job-test")
	}

	if logEntry["scanner_type"] != "axe" {
		t.Errorf("scanner_type = %v, want %q", logEntry["scanner_type"], "axe")
	}
}

func TestConvenienceFunctions(t *testing.T) {
	var buf bytes.Buffer

	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	ctx := WithJobID(context.Background(), "job-conv")

	// Test Info
	buf.Reset()
	Info(ctx, "info message", "key", "value")

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Info: failed to parse log: %v", err)
	}

	if entry["level"] != "INFO" {
		t.Errorf("Info level = %v, want INFO", entry["level"])
	}

	if entry["job_id"] != "job-conv" {
		t.Errorf("Info job_id = %v, want job-conv", entry["job_id"])
	}

	// Test Warn
	buf.Reset()
	Warn(ctx, "warn message")

	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Warn: failed to parse log: %v", err)
	}

	if entry["level"] != "WARN" {
		t.Errorf("Warn level = %v, want WARN", entry["level"])
	}

	// Test Error
	buf.Reset()
	Error(ctx, "error message")

	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Error: failed to parse log: %v", err)
	}

	if entry["level"] != "ERROR" {
		t.Errorf("Error level = %v, want ERROR", entry["level"])
	}

	// Test Debug
	buf.Reset()
	Debug(ctx, "debug message")

	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Debug: failed to parse log: %v", err)
	}

	if entry["level"] != "DEBUG" {
		t.Errorf("Debug level = %v, want DEBUG", entry["level"])
	}
}
