package api

import (
	"context"
	"net"

	"github.com/mattboback/stageflow/internal/artifacts"
	"github.com/mattboback/stageflow/internal/jobstore"
	"github.com/mattboback/stageflow/libs/go/models"
	"github.com/mattboback/stageflow/libs/go/scannerregistry"
)

// JobAdmitter persists a pending job and its selected scanners atomically.
type JobAdmitter interface {
	AdmitJob(context.Context, *models.Job) error
}

// JobStatusReader fetches current job status snapshots.
type JobStatusReader interface {
	GetJob(ctx context.Context, jobID string) (*jobstore.JobRecord, error)
}

// Server wires HTTP handlers to file storage, job admission and job status.
type Server struct {
	config          *ServerConfig
	statusReader    JobStatusReader
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
	Storage             artifacts.Client
	Admitter            JobAdmitter
	CheckDisk           func(context.Context) error
	ValidateZIP         func(context.Context, string) error
	StatusReader        JobStatusReader
	ScannerRegistry     *scannerregistry.Registry
	AllowPrivateTargets bool
	APIToken            string // Optional; see middleware_auth.go
}

// NewServer constructs an API server with injected dependencies.
func NewServer(config *ServerConfig) *Server {
	server := &Server{
		config:          config,
		statusReader:    config.StatusReader,
		scannerRegistry: config.ScannerRegistry,
		ipResolver:      net.DefaultResolver,
	}

	return server
}
