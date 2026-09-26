package api

import (
	"context"
	"net"

	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/scannerregistry"
	"github.com/mattboback/stageflow/libs/go/storage"
	"github.com/mattboback/stageflow/services/platform-api/internal/jobstatus"
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

// TombstoneStore preserves deletion across cache expiry and API restarts.
type TombstoneStore interface {
	TombstoneJob(context.Context, string) error
	JobIsDeleted(context.Context, string) (bool, error)
}

// Server wires HTTP handlers to storage, status, and publisher dependencies.
type Server struct {
	config          *ServerConfig
	jobStatus       *jobstatus.Pipeline
	tombstoneStore  TombstoneStore
	scannerRegistry *scannerregistry.Registry
	ipResolver      ipAddrResolver
	// discoveryMode is public outside tests, which crawl a loopback httptest server.
	discoveryMode targetValidationMode
}

// ServerConfig provides dependencies and endpoints for the public API.
type ServerConfig struct {
	Port                int
	Storage             storage.Client
	Publisher           JobPublisher
	StatusReader        JobStatusReader
	TombstoneStore      TombstoneStore
	ScannerRegistry     *scannerregistry.Registry
	AllowPrivateTargets bool
	APIToken            string // Optional; see middleware_auth.go
	MinIOEndpoint       string // Internal MinIO endpoint (e.g., "minio:9000")
	MinIOPublicEndpoint string // Public endpoint (e.g., "stageflow.org")
	MinIOPublicUseSSL   bool   // Whether to use HTTPS for public URLs
}

// NewServer constructs an API server with injected dependencies.
func NewServer(config *ServerConfig) *Server {
	server := &Server{
		config:          config,
		jobStatus:       jobstatus.New(&jobstatus.Config{CurrentReader: config.StatusReader}),
		tombstoneStore:  config.TombstoneStore,
		scannerRegistry: config.ScannerRegistry,
		ipResolver:      net.DefaultResolver,
	}

	return server
}

func (s *Server) JobStatus() *jobstatus.Pipeline {
	return s.jobStatus
}
