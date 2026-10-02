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

	"github.com/mattboback/stageflow/internal/archive"
	"github.com/mattboback/stageflow/internal/artifacts"
	"github.com/mattboback/stageflow/internal/jobstore"
	"github.com/mattboback/stageflow/libs/go/bootstrap"
	"github.com/mattboback/stageflow/services/platform-api/internal/api"
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

	artifactFiles, err := artifacts.NewFiles(cfg.DataDir)
	if err != nil {
		return err
	}
	defer artifactFiles.Close()

	statusReader, err := jobstore.NewDatabase(&jobstore.Config{URL: cfg.DatabaseURL, RequireSchema: true})
	if err != nil {
		return fmt.Errorf("connect job status database: %w", err)
	}
	defer statusReader.Close()

	server := api.NewServer(&api.ServerConfig{
		Port:       cfg.Port,
		Files:      artifactFiles,
		JobDeleter: statusReader,
		Admitter:   statusReader,
		CheckDisk:  artifactFiles.CheckDisk,
		ValidateZIP: func(ctx context.Context, id string) error {
			file, openErr := artifactFiles.OpenInput(ctx, id)
			if openErr != nil {
				return openErr
			}
			defer file.Close()

			info, statErr := file.Stat()
			if statErr != nil {
				return statErr
			}

			return archive.Validate(file, info.Size())
		},
		StatusReader:        jobstore.StatusReader{Database: statusReader},
		AllowPrivateTargets: cfg.AllowPrivateTargets,
		APIToken:            cfg.APIToken,
	})

	httpServer := newHTTPServer(cfg.Port, server.Router())
	httpServer.BaseContext = func(net.Listener) context.Context { return ctx }

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
