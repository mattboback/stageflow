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

	storage "github.com/mattboback/stageflow/internal/artifacts"
	db "github.com/mattboback/stageflow/internal/jobstore"
	"github.com/mattboback/stageflow/libs/go/bootstrap"
	scanners "github.com/mattboback/stageflow/libs/go/scannerregistry"
	podman "github.com/mattboback/stageflow/services/orchestrator/internal/adapters/runtime"
	"github.com/mattboback/stageflow/services/orchestrator/internal/api"
	"github.com/mattboback/stageflow/services/orchestrator/internal/orchestrator"
)

func main() {
	os.Exit(run())
}

func run() int {
	logger := bootstrap.SetupLogging("orchestrator")

	logger.Info("Starting Orchestrator service...")

	cfg := loadConfig()
	if checkErr1 := cfg.Validate(); checkErr1 != nil {
		logger.Error("Invalid configuration", "error", checkErr1)

		return 1
	}

	scannerRegistry, err := loadScannerRegistry(logger, cfg.ScannerImageOverride)
	if err != nil {
		logger.Error("Failed to load scanner registry", "error", err)

		return 1
	}

	backgroundCtx, stopBackground := context.WithCancel(context.Background())

	defer stopBackground()

	artifactFiles, err := storage.NewFiles(cfg.DataDir)
	if err != nil {
		logger.Error("Open job files", "error", err)
		return 1
	}
	defer artifactFiles.Close()

	if ownershipErr := os.Chown(cfg.DataDir, 65532, 65532); ownershipErr != nil {
		logger.Error("Initialize shared volume permissions", "error", ownershipErr)
		return 1
	}

	if checkErr2 := os.Chown(cfg.DataDir+"/jobs", 65532, 65532); checkErr2 != nil {
		logger.Error("Initialize job directory permissions", "error", checkErr2)
		return 1
	}

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

	podmanClient, err := podman.NewClient(&podman.Config{
		SocketPath:            cfg.PodmanSocket,
		RequestTimeout:        cfg.PodmanRequestTimeout,
		ResponseHeaderTimeout: cfg.PodmanResponseHeaderTimeout,
		DialTimeout:           cfg.PodmanDialTimeout,
	})
	if err != nil {
		logger.Error("Failed to create Podman client", "error", err)

		return 1
	}

	orch := orchestrator.NewOrchestrator(&orchestrator.Config{
		DataDir:              cfg.DataDir,
		DataVolume:           cfg.DataVolume,
		PodmanClient:         podmanClient,
		Database:             database,
		ScannerRegistry:      scannerRegistry,
		ExtractionImage:      cfg.ExtractionImage,
		ScannerImage:         cfg.ScannerImage,
		ScannerImageOverride: cfg.ScannerImageOverride,
		// Timeouts use defaults (5min extraction, 30min scan)
		PodNetwork:      cfg.PodNetwork,
		PodNetnsMode:    cfg.PodNetnsMode,
		PodHostMappings: cfg.PodHostMappings,
		PageLoadTimeout: cfg.PageLoadTimeout,
		Storage:         artifactFiles,
		CheckDisk:       artifactFiles.CheckDisk,
	})

	orch.Start(backgroundCtx)

	apiServer := api.NewServer(&api.Config{
		Port:   cfg.APIPort,
		Health: orch.Health,
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

type monitorWaiter interface {
	WaitForMonitors()
}

type apiShutdowner interface {
	Shutdown(context.Context) error
}

// shutdownService cancels background intake before the bounded HTTP drain.
// Server.Shutdown owns the HTTP handler barrier; only after it returns do the
// message and monitor barriers release dependencies.
func shutdownService(
	ctx context.Context,
	cancel context.CancelFunc,
	api apiShutdowner,
	monitors monitorWaiter,
) error {
	cancel()

	shutdownErr := api.Shutdown(ctx)

	monitors.WaitForMonitors()

	return shutdownErr
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

// loadScannerRegistry builds the scanner registry from the built-in manifests.
func loadScannerRegistry(logger *slog.Logger, defaultImage string) (*scanners.Registry, error) {
	registryConfig, cfgErr := scanners.DefaultConfig()
	if cfgErr != nil {
		return nil, fmt.Errorf("load default scanner config: %w", cfgErr)
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
