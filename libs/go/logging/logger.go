// Package logging provides structured logging utilities for StageFlow services.
// It wraps Go's log/slog package to provide consistent JSON logging with
// domain-specific context fields like job_id, scanner_type, and request_id.
package logging

import (
	"context"
	"log/slog"
	"os"
)

type contextKey string

const (
	jobIDKey     contextKey = "job_id"
	requestIDKey contextKey = "request_id"
	runIDKey     contextKey = "run_id"
	scannerKey   contextKey = "scanner_type"
	componentKey contextKey = "component"
)

// Config controls JSON slog behavior shared across services.
type Config struct {
	// Level sets the minimum log level. Defaults to Info.
	Level       slog.Level
	AddSource   bool
	ServiceName string
}

// New builds a JSON slog.Logger for StageFlow services, honoring Config options.
func New(cfg *Config) *slog.Logger {
	if cfg == nil {
		cfg = &Config{}
	}

	opts := &slog.HandlerOptions{
		Level:     cfg.Level,
		AddSource: cfg.AddSource,
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, opts))
	if cfg.ServiceName == "" {
		return logger
	}

	return logger.With(slog.String("service", cfg.ServiceName))
}

// NewDefault creates a logger with sensible defaults for production.
func NewDefault(serviceName string) *slog.Logger {
	return New(&Config{
		Level:       slog.LevelInfo,
		AddSource:   false,
		ServiceName: serviceName,
	})
}

// SetDefault installs the logger as slog.Default().
func SetDefault(logger *slog.Logger) {
	slog.SetDefault(logger)
}

// WithJobID attaches job_id to context for structured logging.
func WithJobID(ctx context.Context, jobID string) context.Context {
	return context.WithValue(ctx, jobIDKey, jobID)
}

// WithRequestID attaches request_id to context for structured logging.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// WithRunID attaches run_id to context for structured logging.
func WithRunID(ctx context.Context, runID string) context.Context {
	return context.WithValue(ctx, runIDKey, runID)
}

// WithScanner attaches scanner_type to context for structured logging.
func WithScanner(ctx context.Context, scannerType string) context.Context {
	return context.WithValue(ctx, scannerKey, scannerType)
}

// WithComponent attaches component to context for structured logging.
func WithComponent(ctx context.Context, component string) context.Context {
	return context.WithValue(ctx, componentKey, component)
}

// JobID reads job_id from context, returning "" when absent.
func JobID(ctx context.Context) string {
	if v := ctx.Value(jobIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}

	return ""
}

// RequestID reads request_id from context, returning "" when absent.
func RequestID(ctx context.Context) string {
	if v := ctx.Value(requestIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}

	return ""
}

// RunID reads run_id from context, returning "" when absent.
func RunID(ctx context.Context) string {
	if v := ctx.Value(runIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}

	return ""
}

// Scanner reads scanner_type from context, returning "" when absent.
func Scanner(ctx context.Context) string {
	if v := ctx.Value(scannerKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}

	return ""
}

// Component reads component from context, returning "" when absent.
func Component(ctx context.Context) string {
	if v := ctx.Value(componentKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}

	return ""
}

// FromContext extracts context values and returns slog attributes for them.
// This allows callers to include context values in their log entries.
func FromContext(ctx context.Context) []slog.Attr {
	var attrs []slog.Attr

	if jobID := JobID(ctx); jobID != "" {
		attrs = append(attrs, slog.String("job_id", jobID))
	}

	if requestID := RequestID(ctx); requestID != "" {
		attrs = append(attrs, slog.String("request_id", requestID))
	}

	if runID := RunID(ctx); runID != "" {
		attrs = append(attrs, slog.String("run_id", runID))
	}

	if scanner := Scanner(ctx); scanner != "" {
		attrs = append(attrs, slog.String("scanner_type", scanner))
	}

	if component := Component(ctx); component != "" {
		attrs = append(attrs, slog.String("component", component))
	}

	return attrs
}

// L returns a logger with context values automatically included.
// This is a convenience function for getting a context-aware logger.
func L(ctx context.Context) *slog.Logger {
	logger := slog.Default()
	attrs := FromContext(ctx)

	if len(attrs) == 0 {
		return logger
	}

	args := make([]any, 0, len(attrs)*2)
	for _, attr := range attrs {
		args = append(args, attr.Key, attr.Value.Any())
	}

	return logger.With(args...)
}

// Info logs at Info level with context fields applied.
func Info(ctx context.Context, msg string, args ...any) {
	L(ctx).Info(msg, args...)
}

// Warn logs at Warn level with context fields applied.
func Warn(ctx context.Context, msg string, args ...any) {
	L(ctx).Warn(msg, args...)
}

// Error logs at Error level with context fields applied.
func Error(ctx context.Context, msg string, args ...any) {
	L(ctx).Error(msg, args...)
}

// Debug logs at Debug level with context fields applied.
func Debug(ctx context.Context, msg string, args ...any) {
	L(ctx).Debug(msg, args...)
}
