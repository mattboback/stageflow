package api

import (
	"context"
	"net"

	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/scannerregistry"
	"github.com/mattboback/stageflow/libs/go/storage"
	"github.com/mattboback/stageflow/services/platform-api/internal/sse"
	"github.com/mattboback/stageflow/services/platform-api/internal/status"
)

// JobPublisher abstracts job.created publishing for dependency injection.
type JobPublisher interface {
	PublishJobCreated(ctx context.Context, envelope *events.Envelope) error
}

// JobStatusReader fetches current job status snapshots.
type JobStatusReader interface {
	GetJob(ctx context.Context, jobID string) (*status.JobRecord, error)
}

// Server wires HTTP handlers to storage, status, and publisher dependencies.
type Server struct {
	config          *ServerConfig
	statusReader    JobStatusReader
	pendingJobs     *pendingJobCache
	sseHub          *sse.Hub
	scannerRegistry *scannerregistry.Registry
	ipResolver      ipAddrResolver
}

// ServerConfig provides dependencies and endpoints for the public API.
type ServerConfig struct {
	Port                int
	Storage             storage.Client
	Publisher           JobPublisher
	StatusReader        JobStatusReader
	ScannerRegistry     *scannerregistry.Registry
	AllowPrivateTargets bool
	MinIOEndpoint       string // Internal MinIO endpoint (e.g., "minio:9000")
	MinIOPublicEndpoint string // Public endpoint (e.g., "stageflow.org")
	MinIOPublicUseSSL   bool   // Whether to use HTTPS for public URLs
}

// NewServer constructs an API server with injected dependencies.
func NewServer(config *ServerConfig) *Server {
	return &Server{
		config:          config,
		statusReader:    config.StatusReader,
		pendingJobs:     newPendingJobCache(),
		sseHub:          sse.NewHub(),
		scannerRegistry: config.ScannerRegistry,
		ipResolver:      net.DefaultResolver,
	}
}

// SSEHub returns the server's SSE hub for wiring to event handlers.
func (s *Server) SSEHub() *sse.Hub {
	return s.sseHub
}
