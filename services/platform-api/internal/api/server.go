package api

import (
	"context"
	"net"

	storage "github.com/mattboback/stageflow/internal/artifacts"
	status "github.com/mattboback/stageflow/internal/jobstore"
	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/models"
	"github.com/mattboback/stageflow/libs/go/scannerregistry"
)

// JobPublisher abstracts job.created publishing for dependency injection.
type JobPublisher interface {
	PublishJobCreated(ctx context.Context, envelope *events.Envelope) error
}

// PendingJobPublisher seeds durable status and publishes with a protected handoff.
type PendingJobPublisher interface {
	PublishPendingJob(context.Context, *models.Job, func(context.Context) error) error
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
	statusReader    JobStatusReader
	tombstoneStore  TombstoneStore
	scannerRegistry *scannerregistry.Registry
	ipResolver      ipAddrResolver
	// discoveryMode is public outside tests, which crawl a loopback httptest server.
	discoveryMode targetValidationMode
}

// ServerConfig provides dependencies and endpoints for the public API.
type ServerConfig struct {
	JobDeleter interface {
		DeleteJob(context.Context, string) error
	}
	Port                int
	Storage             storage.Client
	Publisher           JobPublisher
	PendingJobs         PendingJobPublisher
	StatusReader        JobStatusReader
	TombstoneStore      TombstoneStore
	ScannerRegistry     *scannerregistry.Registry
	AllowPrivateTargets bool
	APIToken            string // Optional; see middleware_auth.go
}

// NewServer constructs an API server with injected dependencies.
func NewServer(config *ServerConfig) *Server {
	server := &Server{
		config:          config,
		statusReader:    config.StatusReader,
		tombstoneStore:  config.TombstoneStore,
		scannerRegistry: config.ScannerRegistry,
		ipResolver:      net.DefaultResolver,
	}

	return server
}
