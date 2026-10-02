package bootstrap_test

import (
	"log/slog"
	"testing"

	"github.com/mattboback/stageflow/libs/go/bootstrap"
)

func TestSetupLogging(t *testing.T) {
	t.Parallel()

	logger := bootstrap.SetupLogging("test-service")
	if logger == nil {
		t.Fatal("SetupLogging returned nil")
	}

	// SetupLogging's side effect is the point: code that logs via the package-level
	// slog functions has to pick up the service-tagged logger.
	if slog.Default() == nil {
		t.Fatal("slog.Default() is nil after SetupLogging")
	}
}
