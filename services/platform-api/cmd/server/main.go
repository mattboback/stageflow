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
	"syscall"
	"time"

	db "github.com/mattboback/stageflow/internal/jobstore"
	"github.com/mattboback/stageflow/libs/go/bootstrap"
	"github.com/mattboback/stageflow/libs/go/scannerregistry"
	"github.com/mattboback/stageflow/services/platform-api/internal/api"
	"github.com/mattboback/stageflow/services/platform-api/internal/messaging"
	"github.com/mattboback/stageflow/services/platform-api/internal/tombstone"
)

func main() {
	bootstrap.SetupLogging("platform-api")

	if err := run(); err != nil {
		slog.Error("Platform API exited with error", "error", err)
		os.Exit(1)
	}
}

// validateSecurityPolicies validates the security and CORS configuration
// before the server starts accepting traffic.
func validateSecurityPolicies() error {
	if err := api.ValidateSecurityConfig(); err != nil {
		return fmt.Errorf("invalid security policy configuration: %w", err)
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

	statusReader, err := db.NewDatabase(&db.Config{URL: cfg.DatabaseURL})
	if err != nil {
		return fmt.Errorf("connect job status database: %w", err)
	}
	defer statusReader.Close()

	tombstoneStore, err := tombstone.NewStore(cfg.TombstoneDBPath)
	if err != nil {
		return fmt.Errorf("failed to initialize tombstone store: %w", err)
	}

	defer func() {
		if cerr := tombstoneStore.Close(); cerr != nil {
			slog.Error("Failed to close tombstone store", "error", cerr)
		}
	}()

	slog.Info("Tombstone store initialized", "path", cfg.TombstoneDBPath)

	msgService := messaging.NewService(natsClient)
	scannerRegistry := loadScannerRegistry(slog.Default())

	if scannerRegistry == nil {
		return errors.New("failed to initialize scanner registry")
	}

	server := api.NewServer(&api.ServerConfig{
		Port:                cfg.Port,
		Storage:             minioClient,
		Publisher:           msgService,
		PendingJobs:         statusReader,
		StatusReader:        db.StatusReader{Database: statusReader},
		TombstoneStore:      tombstoneStore,
		ScannerRegistry:     scannerRegistry,
		AllowPrivateTargets: cfg.AllowPrivateTargets,
		APIToken:            cfg.APIToken,
		MinIOEndpoint:       cfg.MinIO.Endpoint,
		MinIOPublicEndpoint: cfg.MinIO.PublicEndpoint,
		MinIOPublicUseSSL:   cfg.MinIO.PublicUseSSL,
	})

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

// loadScannerRegistry builds the scanner registry from the built-in manifests.
func loadScannerRegistry(logger *slog.Logger) *scannerregistry.Registry {
	registryConfig, err := scannerregistry.DefaultConfig()
	if err != nil {
		logger.Error("Failed to load default scanner config", "error", err)
		return nil
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
