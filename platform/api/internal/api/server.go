package api

import (
	"context"
	"net"

	"github.com/mattboback/stageflow/packages/shared-go/events"
	"github.com/mattboback/stageflow/packages/shared-go/scannerregistry"
	"github.com/mattboback/stageflow/packages/shared-go/storage"
	"github.com/mattboback/stageflow/platform/api/internal/sse"
	"github.com/mattboback/stageflow/platform/api/internal/status"
)

// JobPublisher abstracts job.created publishing for dependency injection.
type JobPublisher interface {
	PublishJobCreated(ctx context.Context, envelope *events.Envelope) error
}

// Server wires HTTP handlers to storage, status, and publisher dependencies.
type Server struct {
	config          *ServerConfig
	statusStore     *status.Store
	sseHub          *sse.Hub
	scannerRegistry *scannerregistry.Registry
	ipResolver      ipAddrResolver
}

// ServerConfig provides dependencies and endpoints for the public API.
type ServerConfig struct {
	Port                int
	Storage             storage.Client
	Publisher           JobPublisher
	StatusStore         *status.Store
	ScannerRegistry     *scannerregistry.Registry
	MinIOEndpoint       string // Internal MinIO endpoint (e.g., "minio:9000")
	MinIOPublicEndpoint string // Public endpoint (e.g., "stageflow.org")
	MinIOPublicUseSSL   bool   // Whether to use HTTPS for public URLs
}

// NewServer constructs an API server with injected dependencies.
func NewServer(config *ServerConfig) *Server {
	return &Server{
		config:          config,
		statusStore:     config.StatusStore,
		sseHub:          sse.NewHub(),
		scannerRegistry: config.ScannerRegistry,
		ipResolver:      net.DefaultResolver,
	}
}

// SSEHub returns the server's SSE hub for wiring to event handlers.
func (s *Server) SSEHub() *sse.Hub {
	return s.sseHub
}
