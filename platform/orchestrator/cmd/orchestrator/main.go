// Package main boots the orchestrator service and wires dependencies.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mattboback/stageflow/packages/shared-go/bootstrap"
	"github.com/mattboback/stageflow/packages/shared-go/config"
	scanners "github.com/mattboback/stageflow/packages/shared-go/scannerregistry"
	"github.com/mattboback/stageflow/platform/orchestrator/internal/adapters/messaging"
	db "github.com/mattboback/stageflow/platform/orchestrator/internal/adapters/repository"
	podman "github.com/mattboback/stageflow/platform/orchestrator/internal/adapters/runtime"
	"github.com/mattboback/stageflow/platform/orchestrator/internal/api"
	"github.com/mattboback/stageflow/platform/orchestrator/internal/orchestrator"
)

func main() {
	os.Exit(run())
}

func run() int {
	logger := bootstrap.SetupLogging("orchestrator")

	logger.Info("Starting Orchestrator service...")

	cfg := loadConfig()
	if err := cfg.Validate(); err != nil {
		logger.Error("Invalid configuration", "error", err)

		return 1
	}

	scannerRegistry := loadScannerRegistry(logger, cfg.ScannerImageOverride)

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

	defer func() {
		if closeErr := database.Close(); closeErr != nil {
			logger.Error("Failed to close database", "error", closeErr)
		}
	}()

	if cfg.JobEventsRetentionDays > 0 {
		prunerErr := database.StartJobEventsPruner(backgroundCtx, db.JobEventsPrunerConfig{
			Retention: time.Duration(cfg.JobEventsRetentionDays) * 24 * time.Hour,
			Interval:  time.Duration(cfg.JobEventsPruneIntervalMinutes) * time.Minute,
			BatchSize: cfg.JobEventsPruneBatchSize,
			Logger:    logger.With("component", "job-events-pruner"),
		})
		if prunerErr != nil {
			logger.Error("Failed to start job events pruner", "error", prunerErr)

			return 1
		}
	} else {
		logger.Info("Job events pruner disabled", "retention_days", cfg.JobEventsRetentionDays)
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
		PodmanClient:    podmanClient,
		Database:        database,
		Publisher:       publisher,
		ScannerRegistry: scannerRegistry,
		NatsURL:         cfg.NATS.URL,
		MinIOEndpoint:   cfg.MinIO.Endpoint,
		MinIOAccessKey:  cfg.MinIO.AccessKey,
		MinIOSecretKey:  cfg.MinIO.SecretKey,
		MinIOUseSSL:     cfg.MinIO.UseSSL,
		NatsHost:        cfg.NatsHost,
		MinioHost:       cfg.MinioHost,
		ExtractionImage: cfg.ExtractionImage,
		ScannerImage:    cfg.ScannerImage,
		// Timeouts use defaults (5min extraction, 30min scan)
		PodNetwork:      cfg.PodNetwork,
		PodNetnsMode:    cfg.PodNetnsMode,
		PodHostMappings: cfg.PodHostMappings,
		PageLoadTimeout: cfg.PageLoadTimeout,
		ScrollTimeout:   cfg.ScrollTimeout,
		StagingStorage:  minioClient,
		Storage:         minioClient,
	})

	consumer := messaging.NewConsumer(natsClient, orch)

	if startErr := consumer.Start(backgroundCtx); startErr != nil {
		logger.Error("Failed to start consumer", "error", startErr)

		return 1
	}

	orch.Start(backgroundCtx)

	apiServer := api.NewServer(&api.Config{
		Database:     database,
		PodmanClient: podmanClient,
		Publisher:    publisher,
		Port:         cfg.APIPort,
	})

	sigChan := make(chan os.Signal, 1)

	go func() {
		if apiStartErr := apiServer.Start(); apiStartErr != nil {
			logger.Error("Failed to start API server", "error", apiStartErr)

			sigChan <- os.Interrupt
		}
	}()

	logger.Info("Orchestrator service started successfully", "api_port", cfg.APIPort)

	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down orchestrator service...")
	stopBackground()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if shutdownErr := apiServer.Shutdown(shutdownCtx); shutdownErr != nil {
		logger.Error("Failed to shutdown API server", "error", shutdownErr)
	}

	return 0
}

// loadScannerRegistry loads the scanner registry from configuration files or defaults.
// It looks for scanner overrides in the following order:
// 1. SCANNER_CONFIG_PATH environment variable
// 2. ./config/scanners.yaml (relative to working directory)
// 3. Default built-in configuration.
func loadScannerRegistry(logger *slog.Logger, defaultImage string) *scanners.Registry {
	configPath := config.GetEnv("SCANNER_CONFIG_PATH", "")

	registryConfig := scanners.DefaultConfig()

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
		registryConfig = scanners.ApplyOverrides(registryConfig, scannerOverrides)
		logger.Info("Applied scanner overrides", "override_count", len(scannerOverrides.Scanners))
	}

	if defaultImage != "" {
		registryConfig.DefaultImage = defaultImage
	}

	registry, err := scanners.InitializeRegistry(registryConfig)
	if err != nil {
		logger.Error("Failed to initialize scanner registry", "error", err)
		// Create minimal registry as fallback
		registry = scanners.NewRegistry(defaultImage)
	}

	logger.Info("Scanner registry initialized",
		"scanner_count", registry.Count(),
		"enabled_count", registry.CountEnabled())

	return registry
}
