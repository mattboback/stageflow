// Package bootstrap provides shared service startup helpers.
package bootstrap

import (
	"log/slog"

	"github.com/mattboback/stageflow/libs/go/logging"
)

// SetupLogging initializes a structured logger and installs it as slog.Default.
func SetupLogging(serviceName string) *slog.Logger {
	logger := logging.NewDefault(serviceName)
	logging.SetDefault(logger)

	return logger
}
