// Package bootstrap provides shared service startup helpers.
package bootstrap

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mattboback/stageflow/libs/go/config"
	"github.com/mattboback/stageflow/libs/go/logging"
	"github.com/mattboback/stageflow/libs/go/messaging"
	"github.com/mattboback/stageflow/libs/go/storage"
)

// NATSOptions controls optional setup behavior for NATS clients.
type NATSOptions struct {
	EnsureStreams       bool
	IgnoreEnsureFailure bool
}

// MinIOOptions controls optional setup behavior for MinIO clients.
type MinIOOptions struct {
	EnsureBuckets       bool
	IgnoreEnsureFailure bool
}

// SetupLogging initializes a structured logger and installs it as slog.Default.
func SetupLogging(serviceName string) *slog.Logger {
	logger := logging.NewDefault(serviceName)
	logging.SetDefault(logger)

	return logger
}

// NewNATSClient builds a NATS client and optionally ensures streams exist.
// If IgnoreEnsureFailure is true, the client is returned even when stream setup fails.
func NewNATSClient(ctx context.Context, cfg config.NATSConfig, opts NATSOptions) (*messaging.Client, error) {
	client, err := messaging.NewClient(&cfg)
	if err != nil {
		return nil, err
	}

	if opts.EnsureStreams {
		if ensureErr := client.EnsureStreams(ctx); ensureErr != nil {
			if opts.IgnoreEnsureFailure {
				return client, ensureErr
			}

			_ = client.Close()

			return nil, fmt.Errorf("ensure NATS streams: %w", ensureErr)
		}
	}

	return client, nil
}

// NewMinIOClient builds a MinIO client and optionally ensures buckets exist.
// If IgnoreEnsureFailure is true, the client is returned even when bucket setup fails.
func NewMinIOClient(ctx context.Context, cfg config.MinIOConfig, opts MinIOOptions) (*storage.MinIOClient, error) {
	client, err := storage.NewMinIOClient(&cfg)
	if err != nil {
		return nil, err
	}

	if opts.EnsureBuckets {
		if ensureErr := client.EnsureBuckets(ctx); ensureErr != nil {
			if opts.IgnoreEnsureFailure {
				return client, ensureErr
			}

			return nil, fmt.Errorf("ensure MinIO buckets: %w", ensureErr)
		}
	}

	return client, nil
}
