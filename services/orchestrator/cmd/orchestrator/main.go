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

	"github.com/mattboback/stageflow/internal/artifacts"
	"github.com/mattboback/stageflow/internal/jobstore"
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
	if err := cfg.Validate(); err != nil {
		logger.Error("Invalid configuration", "error", err)

		return 1
	}

	scannerRegistry, err := loadScannerRegistry(logger, cfg.ScannerImageOverride)
	if err != nil {
		logger.Error("Failed to load scanner registry", "error", err)

		return 1
	}

	backgroundCtx, stopBackground := context.WithCancel(context.Background())

	defer stopBackground()

	artifactFiles, err := artifacts.NewFiles(cfg.DataDir)
	if err != nil {
		logger.Error("Open job files", "error", err)
		return 1
	}
	defer artifactFiles.Close()

	if ownershipErr := os.Chown(cfg.DataDir, 65532, 65532); ownershipErr != nil {
		logger.Error("Initialize shared volume permissions", "error", ownershipErr)
		return 1
	}

	if chownErr := os.Chown(cfg.DataDir+"/jobs", 65532, 65532); chownErr != nil {
		logger.Error("Initialize job directory permissions", "error", chownErr)
		return 1
	}

	database, err := jobstore.NewDatabase(&jobstore.Config{URL: cfg.DatabaseURL})
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

	podmanClient, err := newPodmanClient(backgroundCtx, cfg)
	if err != nil {
		logger.Error("Failed to prepare the Podman runtime", "error", err)

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
		Files:           artifactFiles,
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

// newPodmanClient creates the Podman client and, in bridge mode, verifies the
// job network before any job can run on it. Host mode has no pod network; it is
// the local-only private-target setup.
func newPodmanClient(ctx context.Context, cfg *Config) (*podman.Client, error) {
	client, err := podman.NewClient(&podman.Config{
		SocketPath:            cfg.PodmanSocket,
		RequestTimeout:        cfg.PodmanRequestTimeout,
		ResponseHeaderTimeout: cfg.PodmanResponseHeaderTimeout,
		DialTimeout:           cfg.PodmanDialTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("create Podman client: %w", err)
	}

	if cfg.PodNetwork == "" {
		return client, nil
	}

	networkCtx, cancel := context.WithTimeout(ctx, cfg.PodmanRequestTimeout)
	defer cancel()

	if networkErr := podman.VerifyJobNetwork(networkCtx, client, cfg.PodNetwork); networkErr != nil {
		return nil, networkErr
	}

	return client, nil
}

type reconcileWaiter interface {
	Wait()
}

type apiShutdowner interface {
	Shutdown(context.Context) error
}

// shutdownService cancels the reconcile loop before the bounded HTTP drain.
// Server.Shutdown owns the HTTP handler barrier; only after it returns does the
// service wait for the loop to stop, so dependencies outlive both.
func shutdownService(
	ctx context.Context,
	cancel context.CancelFunc,
	api apiShutdowner,
	reconciler reconcileWaiter,
) error {
	cancel()

	shutdownErr := api.Shutdown(ctx)

	reconciler.Wait()

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
