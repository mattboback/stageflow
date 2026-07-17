// Package main boots the orchestrator service and wires dependencies.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mattboback/stageflow/libs/go/bootstrap"
	"github.com/mattboback/stageflow/libs/go/config"
	scanners "github.com/mattboback/stageflow/libs/go/scannerregistry"
	"github.com/mattboback/stageflow/services/orchestrator/internal/adapters/messaging"
	db "github.com/mattboback/stageflow/services/orchestrator/internal/adapters/repository"
	podman "github.com/mattboback/stageflow/services/orchestrator/internal/adapters/runtime"
	"github.com/mattboback/stageflow/services/orchestrator/internal/api"
	"github.com/mattboback/stageflow/services/orchestrator/internal/orchestrator"
)

func main() {
	os.Exit(run())
}

//nolint:gocyclo,gocognit // Startup validates each dependency and preserves precise failure logging.
func run() int {
	logger := bootstrap.SetupLogging("orchestrator")

	logger.Info("Starting Orchestrator service...")

	cfg := loadConfig()
	if err := cfg.Validate(); err != nil {
		logger.Error("Invalid configuration", "error", err)

		return 1
	}

	scannerRegistry, err := loadScannerRegistry(logger, cfg.ScannerImageOverride)
	if err != nil {
		logger.Error("Failed to load scanner registry", "error", err)

		return 1
	}

	ctx := context.Background()
	backgroundCtx, stopBackground := context.WithCancel(context.Background())

	defer stopBackground()

	minioClient, err := bootstrap.NewMinIOClient(ctx, cfg.MinIO, bootstrap.MinIOOptions{})
	if err != nil {
		logger.Error("Failed to create MinIO client", "error", err)

		return 1
	}

	natsClient, err := bootstrap.NewNATSClient(ctx, cfg.NATS, bootstrap.NATSOptions{
		EnsureStreams:       true,
		IgnoreEnsureFailure: true,
	})
	if err != nil {
		if natsClient == nil {
			logger.Error("Failed to create NATS client", "error", err)

			return 1
		}

		logger.Warn("Failed to ensure streams", "error", err)
	}

	defer func() {
		if closeErr := natsClient.Close(); closeErr != nil {
			logger.Error("Failed to close NATS client", "error", closeErr)
		}
	}()

	database, err := db.NewDatabase(&db.Config{URL: cfg.DatabaseURL})
	if err != nil {
		logger.Error("Failed to create database", "error", err)

		return 1
	}

	databaseCloseSafe := true
	defer func() {
		if !databaseCloseSafe {
			logger.Error("Skipping database close because HTTP handlers are still active")

			return
		}

		if closeErr := database.Close(); closeErr != nil {
			logger.Error("Failed to close database", "error", closeErr)
		}
	}()

	sanitizedJobs, sanitizeErr := database.SanitizeLegacyTerminalRecords(ctx)
	if sanitizeErr != nil {
		logger.Error("Failed to sanitize legacy terminal records", "error", sanitizeErr)

		return 1
	}

	if sanitizedJobs > 0 {
		logger.Info("Sanitized legacy terminal records", "jobs", sanitizedJobs)
	}

	if prunerErr := startJobEventsPruner(backgroundCtx, logger, database, cfg); prunerErr != nil {
		logger.Error("Failed to start job events pruner", "error", prunerErr)

		return 1
	}

	podmanClient, err := podman.NewClient(&podman.Config{
		SocketPath: cfg.PodmanSocket,
	})
	if err != nil {
		logger.Error("Failed to create Podman client", "error", err)

		return 1
	}

	publisher := messaging.NewPublisher(natsClient)

	orch := orchestrator.NewOrchestrator(&orchestrator.Config{
		PodmanClient:         podmanClient,
		Database:             database,
		Publisher:            publisher,
		ScannerRegistry:      scannerRegistry,
		NatsURL:              cfg.NATS.URL,
		MinIOEndpoint:        cfg.MinIO.Endpoint,
		MinIOAccessKey:       cfg.MinIO.AccessKey,
		MinIOSecretKey:       cfg.MinIO.SecretKey,
		MinIOUseSSL:          cfg.MinIO.UseSSL,
		NatsHost:             cfg.NatsHost,
		MinioHost:            cfg.MinioHost,
		ExtractionImage:      cfg.ExtractionImage,
		ScannerImage:         cfg.ScannerImage,
		ScannerImageOverride: cfg.ScannerImageOverride,
		// Timeouts use defaults (5min extraction, 30min scan)
		PodNetwork:           cfg.PodNetwork,
		PodNetnsMode:         cfg.PodNetnsMode,
		PodHostMappings:      cfg.PodHostMappings,
		PageLoadTimeout:      cfg.PageLoadTimeout,
		ScrollTimeout:        cfg.ScrollTimeout,
		OpenRouterAPIKey:     cfg.OpenRouterAPIKey,
		OpenRouterAppTitle:   cfg.OpenRouterAppTitle,
		OpenRouterAppReferer: cfg.OpenRouterAppReferer,
		StagingStorage:       minioClient,
		Storage:              minioClient,
	})

	consumer := messaging.NewConsumer(natsClient, orch)

	if startErr := startEventProcessing(backgroundCtx, orch, consumer); startErr != nil {
		logger.Error("Failed to start consumer", "error", startErr)
		stopEventProcessing(stopBackground, natsClient, database, orch)

		return 1
	}

	apiServer := api.NewServer(&api.Config{
		Database:            database,
		PodmanClient:        podmanClient,
		APIToken:            cfg.APIToken,
		Port:                cfg.APIPort,
		AdminRateLimitRPS:   cfg.AdminRateLimitRPS,
		AdminRateLimitBurst: cfg.AdminRateLimitBurst,
		Metrics:             orch.Metrics(),
	})

	sigChan := make(chan os.Signal, 1)
	apiErr := make(chan error, 1)

	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	go func() {
		apiErr <- apiServer.Start()
	}()

	logger.Info("Orchestrator service started successfully", "api_port", cfg.APIPort)

	apiStartErr := waitForShutdown(sigChan, apiErr)
	if apiStartErr != nil {
		logger.Error("Orchestrator API server stopped unexpectedly", "error", apiStartErr)
	}

	logger.Info("Shutting down orchestrator service...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	shutdownErr := shutdownService(
		shutdownCtx,
		stopBackground,
		apiServer,
		natsClient,
		database,
		orch,
	)
	if shutdownErr != nil {
		logger.Error("Failed to shutdown API server", "error", shutdownErr)
	}

	if errors.Is(shutdownErr, api.ErrRequestDrainTimeout) {
		// run() is immediately followed by os.Exit in main. Leaving the pool
		// open until process exit is safer than closing it under a handler that
		// ignored request-context cancellation.
		databaseCloseSafe = false
	}

	if apiStartErr != nil || shutdownErr != nil {
		return 1
	}

	return 0
}

type orchestratorStarter interface {
	Start(context.Context)
}

type consumerStarter interface {
	Start(context.Context) error
}

type consumerStopper interface {
	StopConsumers()
}

type databaseTaskStopper interface {
	StopBackgroundTasks()
}

type monitorWaiter interface {
	WaitForMonitors()
}

type apiShutdowner interface {
	Shutdown(context.Context) error
}

// startEventProcessing completes restart reconciliation before the consumer
// can redeliver an extraction.ready event. This prevents normal and recovery
// claim paths from racing over the same deterministic scanner container.
func startEventProcessing(
	ctx context.Context,
	orchestrator orchestratorStarter,
	consumer consumerStarter,
) error {
	orchestrator.Start(ctx)

	return consumer.Start(ctx)
}

// stopEventProcessing closes intake before waiting on anything that callbacks
// may create. This ordering prevents sync.WaitGroup Add calls from racing Wait
// and keeps the database alive until every message and maintenance task exits.
func stopEventProcessing(
	cancel context.CancelFunc,
	consumers consumerStopper,
	database databaseTaskStopper,
	monitors monitorWaiter,
) {
	cancel()
	consumers.StopConsumers()
	database.StopBackgroundTasks()
	monitors.WaitForMonitors()
}

// shutdownService cancels background intake before the bounded HTTP drain.
// Server.Shutdown owns the HTTP handler barrier; only after it returns do the
// message, database-maintenance, and monitor barriers release dependencies.
func shutdownService(
	ctx context.Context,
	cancel context.CancelFunc,
	api apiShutdowner,
	consumers consumerStopper,
	database databaseTaskStopper,
	monitors monitorWaiter,
) error {
	cancel()

	shutdownErr := api.Shutdown(ctx)

	consumers.StopConsumers()
	database.StopBackgroundTasks()
	monitors.WaitForMonitors()

	return shutdownErr
}

func startJobEventsPruner(
	ctx context.Context,
	logger *slog.Logger,
	database *db.Database,
	cfg *Config,
) error {
	if cfg.JobEventsRetentionDays <= 0 {
		logger.Info("Job events pruner disabled", "retention_days", cfg.JobEventsRetentionDays)

		return nil
	}

	return database.StartJobEventsPruner(ctx, db.JobEventsPrunerConfig{
		Retention: time.Duration(cfg.JobEventsRetentionDays) * 24 * time.Hour,
		Interval:  time.Duration(cfg.JobEventsPruneIntervalMinutes) * time.Minute,
		BatchSize: cfg.JobEventsPruneBatchSize,
		Logger:    logger.With("component", "job-events-pruner"),
	})
}

func waitForShutdown(signals <-chan os.Signal, apiErrors <-chan error) error {
	select {
	case <-signals:
		return nil
	case err := <-apiErrors:
		if err == nil {
			return errors.New("API server stopped unexpectedly")
		}

		return err
	}
}

// loadScannerRegistry loads the scanner registry from configuration files or defaults.
// It looks for scanner overrides in the following order:
// 1. SCANNER_CONFIG_PATH environment variable
// 2. ./config/scanners.yaml (relative to working directory)
// 3. Default built-in configuration.
func loadScannerRegistry(logger *slog.Logger, defaultImage string) (*scanners.Registry, error) {
	configPath := config.GetEnv("SCANNER_CONFIG_PATH", "")

	registryConfig, cfgErr := scanners.DefaultConfig()
	if cfgErr != nil {
		return nil, fmt.Errorf("load default scanner config: %w", cfgErr)
	}

	var scannerOverrides *scanners.Overrides

	if configPath != "" {
		overrides, err := scanners.LoadOverrides(configPath)
		if err != nil {
			logger.Warn("Failed to load scanner overrides from path, using defaults",
				"path", configPath, "error", err)
		} else {
			logger.Info("Loaded scanner overrides", "path", configPath)

			scannerOverrides = overrides
		}
	}

	if scannerOverrides == nil {
		wd, wdErr := os.Getwd()
		if wdErr == nil {
			overrides, err := scanners.LoadOverridesFromDir(wd)
			if err != nil {
				logger.Debug("No scanner overrides found in working directory, using defaults",
					"dir", wd, "error", err)
			} else {
				logger.Info("Loaded scanner overrides from working directory")

				scannerOverrides = overrides
			}
		}
	}

	if scannerOverrides != nil {
		var unknown []string

		registryConfig, unknown = scanners.ApplyOverridesChecked(registryConfig, scannerOverrides)
		if err := scanners.FormatUnknownScannerOverrides(unknown); err != nil {
			return nil, fmt.Errorf("apply scanner overrides: %w", err)
		}

		logger.Info("Applied scanner overrides", "override_count", len(scannerOverrides.Scanners))
	}

	if defaultImage != "" {
		registryConfig.DefaultImage = defaultImage
	}

	registry, err := scanners.InitializeRegistry(registryConfig)
	if err != nil {
		return nil, fmt.Errorf("initialize scanner registry: %w", err)
	}

	logger.Info("Scanner registry initialized",
		"scanner_count", registry.Count(),
		"enabled_count", registry.CountEnabled())

	return registry, nil
}
