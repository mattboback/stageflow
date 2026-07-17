// Package main starts the public platform API service.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mattboback/stageflow/libs/go/bootstrap"
	"github.com/mattboback/stageflow/libs/go/scannerregistry"
	"github.com/mattboback/stageflow/services/platform-api/internal/api"
	"github.com/mattboback/stageflow/services/platform-api/internal/jobstatus"
	"github.com/mattboback/stageflow/services/platform-api/internal/messaging"
	"github.com/mattboback/stageflow/services/platform-api/internal/project"
	"github.com/mattboback/stageflow/services/platform-api/internal/statussource"
)

func main() {
	bootstrap.SetupLogging("platform-api")

	if err := run(); err != nil {
		slog.Error("Platform API exited with error", "error", err)
		os.Exit(1)
	}
}

// validateSecurityPolicies validates the security, auth, and CORS
// configuration before the server starts accepting traffic.
func validateSecurityPolicies() error {
	if err := api.ValidateSecurityConfig(); err != nil {
		return fmt.Errorf("invalid security policy configuration: %w", err)
	}

	if err := api.ValidateAuthConfig(); err != nil {
		return fmt.Errorf("invalid authentication configuration: %w", err)
	}

	if err := api.ValidateCORSConfig(); err != nil {
		return fmt.Errorf("invalid CORS configuration: %w", err)
	}

	return nil
}

//nolint:gocyclo // Startup intentionally validates each dependency and returns its precise failure.
func run() error {
	slog.Info("Starting Platform API...")

	cfg := loadConfig()
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	if err := validateSecurityPolicies(); err != nil {
		return err
	}

	ctx, cancelRun := context.WithCancel(context.Background())
	defer cancelRun()

	natsClient, err := bootstrap.NewNATSClient(ctx, cfg.NATS, bootstrap.NATSOptions{EnsureStreams: true})
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	defer func() {
		if cerr := natsClient.Close(); cerr != nil {
			slog.Error("Failed to close NATS client", "error", cerr)
		}
	}()

	slog.Info("NATS streams initialized")

	minioClient, err := bootstrap.NewMinIOClient(ctx, cfg.MinIO, bootstrap.MinIOOptions{EnsureBuckets: true})
	if err != nil {
		return fmt.Errorf("failed to connect to MinIO: %w", err)
	}

	slog.Info("MinIO buckets initialized")

	statusReader, err := statussource.NewClient(&statussource.Config{
		BaseURL: cfg.OrchestratorAPIURL,
		Timeout: 5 * time.Second,
		Token:   cfg.OrchestratorAPIToken,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize orchestrator status source: %w", err)
	}

	projectStore, err := project.NewStore(cfg.ProjectDBPath)
	if err != nil {
		return fmt.Errorf("failed to initialize project store: %w", err)
	}

	defer func() {
		if cerr := projectStore.Close(); cerr != nil {
			slog.Error("Failed to close project store", "error", cerr)
		}
	}()

	slog.Info("Project store initialized", "path", cfg.ProjectDBPath)

	msgService := messaging.NewService(natsClient)
	scannerRegistry := loadScannerRegistry(slog.Default(), cfg.ScannerConfigPath)

	if scannerRegistry == nil {
		return errors.New("failed to initialize scanner registry")
	}

	server := api.NewServer(&api.ServerConfig{
		Port:                cfg.Port,
		Storage:             minioClient,
		Publisher:           msgService,
		StatusReader:        statusReader,
		ProjectStore:        projectStore,
		ScannerRegistry:     scannerRegistry,
		AllowPrivateTargets: cfg.AllowPrivateTargets,
		MinIOEndpoint:       cfg.MinIO.Endpoint,
		MinIOPublicEndpoint: cfg.MinIO.PublicEndpoint,
		MinIOPublicUseSSL:   cfg.MinIO.PublicUseSSL,
	})

	baselineSummary, reconcileErr := startBaselineMaintenance(ctx, server)
	if reconcileErr != nil {
		slog.Warn(
			"Project baseline reconciliation is degraded; durable operations will retry in the background",
			"error", reconcileErr,
			"journal_replayed", baselineSummary.JournalReplayed,
		)
	} else {
		slog.Info(
			"Project baseline reconciliation complete",
			"journal_replayed", baselineSummary.JournalReplayed,
			"legacy_present", baselineSummary.LegacyPresent,
			"legacy_copied", baselineSummary.LegacyCopied,
			"missing_legacy_count", len(baselineSummary.MissingLegacyProjects),
			"missing_legacy_projects", baselineSummary.MissingLegacyProjects,
		)
	}

	if subscribeErr := subscribeStatusEvents(ctx, msgService, server.JobStatus()); subscribeErr != nil {
		return subscribeErr
	}

	httpServer := newHTTPServer(cfg.Port, server.Router())

	stop := make(chan os.Signal, 1)

	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stop)

	serverErr, err := startHTTPServer(httpServer)
	if err != nil {
		return fmt.Errorf("start HTTP server: %w", err)
	}

	slog.Info("Server listening", "port", cfg.Port)

	select {
	case <-stop:
	case serveErr := <-serverErr:
		if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", serveErr)
		}
	}

	slog.Info("Shutting down server...")
	cancelRun()

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelShutdown()

	if shutdownErr := httpServer.Shutdown(shutdownCtx); shutdownErr != nil {
		slog.Error("Server shutdown error", "error", shutdownErr)
	}

	server.WaitForBaselineReconciler()

	slog.Info("Server stopped")

	return nil
}

func startHTTPServer(server *http.Server) (<-chan error, error) {
	var listenConfig net.ListenConfig

	listener, err := listenConfig.Listen(context.Background(), "tcp", server.Addr)
	if err != nil {
		return nil, err
	}

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Serve(listener)
	}()

	return serverErr, nil
}

type statusEventSubscriber interface {
	SubscribeToStatusEvents(context.Context, messaging.EventHandler) error
}

type baselineMaintainer interface {
	ReconcileProjectBaselines(context.Context) (api.BaselineReconcileSummary, error)
	StartBaselineReconciler(context.Context)
}

func startBaselineMaintenance(
	ctx context.Context,
	maintainer baselineMaintainer,
) (api.BaselineReconcileSummary, error) {
	summary, err := maintainer.ReconcileProjectBaselines(ctx)
	maintainer.StartBaselineReconciler(ctx)

	return summary, err
}

func subscribeStatusEvents(
	ctx context.Context,
	subscriber statusEventSubscriber,
	pipeline *jobstatus.Pipeline,
) error {
	pipelineHandler := jobstatus.NewEventHandler(pipeline)
	if subscribeErr := subscriber.SubscribeToStatusEvents(ctx, pipelineHandler); subscribeErr != nil {
		return fmt.Errorf("failed to subscribe to lifecycle events: %w", subscribeErr)
	}

	slog.Info("Subscribed to lifecycle events")

	return nil
}

func newHTTPServer(port int, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		// Disable WriteTimeout to allow long-lived Server-Sent Events streams.
		// Per-handler timeouts are enforced by middleware for non-streaming routes.
		WriteTimeout: 0,
		IdleTimeout:  60 * time.Second,
	}
}

func loadScannerRegistry(logger *slog.Logger, configPath string) *scannerregistry.Registry {
	registryConfig, err := scannerregistry.DefaultConfig()
	if err != nil {
		logger.Error("Failed to load default scanner config", "error", err)
		return nil
	}

	scannerOverrides := loadScannerOverrides(logger, configPath)

	if scannerOverrides != nil {
		var unknown []string

		registryConfig, unknown = scannerregistry.ApplyOverridesChecked(registryConfig, scannerOverrides)
		if formatErr := scannerregistry.FormatUnknownScannerOverrides(unknown); formatErr != nil {
			logger.Error("Failed to apply scanner overrides", "error", formatErr)

			return nil
		}

		logger.Info("Applied scanner overrides", "override_count", len(scannerOverrides.Scanners))
	}

	registry, err := scannerregistry.InitializeRegistry(registryConfig)
	if err != nil {
		logger.Error("Failed to initialize scanner registry", "error", err)
		return nil
	}

	if registry != nil {
		logger.Info(
			"Scanner registry initialized",
			"scanner_count",
			registry.Count(),
			"enabled_count",
			registry.CountEnabled(),
		)
	}

	return registry
}

func loadScannerOverrides(logger *slog.Logger, configPath string) *scannerregistry.Overrides {
	path := strings.TrimSpace(configPath)
	if path != "" {
		overrides, err := scannerregistry.LoadOverrides(path)
		if err != nil {
			logger.Warn("Failed to load scanner overrides from path, using defaults", "path", path, "error", err)

			return nil
		}

		logger.Info("Loaded scanner overrides", "path", path)

		return overrides
	}

	wd, err := os.Getwd()
	if err != nil {
		return nil
	}

	overrides, err := scannerregistry.LoadOverridesFromDir(wd)
	if err != nil {
		logger.Debug("No scanner overrides found in working directory, using defaults", "dir", wd, "error", err)

		return nil
	}

	logger.Info("Loaded scanner overrides from working directory")

	return overrides
}
