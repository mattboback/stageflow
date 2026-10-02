package api

import (
	"context"
	"io"
	"net"

	"github.com/mattboback/stageflow/internal/jobstore"
	"github.com/mattboback/stageflow/libs/go/models"
)

// JobAdmitter persists a pending job and its selected scanners atomically.
type JobAdmitter interface {
	AdmitJob(context.Context, *models.Job) error
}

// JobStatusReader fetches current job status snapshots.
type JobStatusReader interface {
	GetJob(ctx context.Context, jobID string) (*jobstore.JobRecord, error)
}

// JobFiles stores a job's files: the uploaded ZIP, stored auth state, and the
// results the orchestrator writes. Keys are "<job ID>/<path>".
type JobFiles interface {
	Write(ctx context.Context, key string, reader io.Reader) error
	WriteInput(ctx context.Context, jobID string, reader io.Reader) error
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	Remove(ctx context.Context, key string) error
	RemoveJob(ctx context.Context, jobID string) error
}

// Server wires HTTP handlers to file storage, job admission and job status.
type Server struct {
	config       *ServerConfig
	statusReader JobStatusReader
	ipResolver   ipAddrResolver
	// discoveryMode is public outside tests, which crawl a loopback httptest server.
	discoveryMode targetValidationMode
}

// ServerConfig provides dependencies and endpoints for the public API.
type ServerConfig struct {
	JobDeleter interface {
		DeleteJob(context.Context, string) error
	}
	Port                int
	Files               JobFiles
	Admitter            JobAdmitter
	CheckDisk           func(context.Context) error
	ValidateZIP         func(context.Context, string) error
	StatusReader        JobStatusReader
	AllowPrivateTargets bool
	MaxPages            int
	DisableZIPUploads   bool
	APIToken            string // Optional; see middleware_auth.go
}

// NewServer constructs an API server with injected dependencies.
func NewServer(config *ServerConfig) *Server {
	if config.MaxPages <= 0 {
		config.MaxPages = maxURLCount
	}

	server := &Server{
		config:       config,
		statusReader: config.StatusReader,
		ipResolver:   net.DefaultResolver,
	}

	return server
}
