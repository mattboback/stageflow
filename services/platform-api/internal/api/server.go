package api

import (
	"context"
	"net"
	"sync"
	"sync/atomic"

	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/scannerregistry"
	"github.com/mattboback/stageflow/libs/go/storage"
	"github.com/mattboback/stageflow/services/platform-api/internal/jobstatus"
	"github.com/mattboback/stageflow/services/platform-api/internal/project"
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

// ProjectStore abstracts project persistence for handler tests and runtime wiring.
type ProjectStore interface {
	CreateProject(ctx context.Context, p *project.Project) error
	GetProjectBySlug(ctx context.Context, slug string) (*project.Project, error)
	GetProjectByID(ctx context.Context, id string) (*project.Project, error)
	ListProjects(ctx context.Context) ([]project.Project, error)
	UpdateProject(ctx context.Context, id string, u project.Update) error
	DeleteProject(ctx context.Context, id string) error
	RecordProjectJob(ctx context.Context, projectID, jobID string) error
	DeleteProjectJob(ctx context.Context, jobID string) error
	GetProjectForJob(ctx context.Context, jobID string) (*project.Project, error)
	SetBaseline(ctx context.Context, projectID, jobID string) error
	JobBelongsToProject(ctx context.Context, projectID, jobID string) (bool, error)
	QueueBaselinePromotion(
		ctx context.Context,
		projectID, previousJobID, jobID, sourceKey string,
	) (*project.BaselineOperation, error)
	QueueBaselineBackfill(
		ctx context.Context,
		projectID, jobID, sourceKey string,
	) (*project.BaselineOperation, error)
	QueueProjectDeletion(
		ctx context.Context,
		projectID string,
	) (*project.BaselineOperation, error)
	ListBaselineOperations(ctx context.Context) ([]project.BaselineOperation, error)
	MarkBaselineOperationObjectReady(ctx context.Context, id int64) error
	RecordBaselineOperationFailure(ctx context.Context, id int64, cause string) error
	CompleteBaselinePromotion(ctx context.Context, op project.BaselineOperation) error
	CompleteBaselineOperation(ctx context.Context, op project.BaselineOperation) error
}

// Server wires HTTP handlers to storage, status, and publisher dependencies.
type Server struct {
	config          *ServerConfig
	jobStatus       *jobstatus.Pipeline
	projectStore    ProjectStore
	scannerRegistry *scannerregistry.Registry
	ipResolver      ipAddrResolver
	baselineMu      sync.Mutex
	baselineWG      sync.WaitGroup
	legacySweepDue  atomic.Bool
}

// ServerConfig provides dependencies and endpoints for the public API.
type ServerConfig struct {
	Port                int
	Storage             storage.Client
	Publisher           JobPublisher
	StatusReader        JobStatusReader
	ProjectStore        ProjectStore
	ScannerRegistry     *scannerregistry.Registry
	AllowPrivateTargets bool
	MinIOEndpoint       string // Internal MinIO endpoint (e.g., "minio:9000")
	MinIOPublicEndpoint string // Public endpoint (e.g., "stageflow.org")
	MinIOPublicUseSSL   bool   // Whether to use HTTPS for public URLs
}

// NewServer constructs an API server with injected dependencies.
func NewServer(config *ServerConfig) *Server {
	server := &Server{
		config:          config,
		jobStatus:       jobstatus.New(&jobstatus.Config{CurrentReader: config.StatusReader}),
		projectStore:    config.ProjectStore,
		scannerRegistry: config.ScannerRegistry,
		ipResolver:      net.DefaultResolver,
	}
	server.legacySweepDue.Store(true)

	return server
}

func (s *Server) JobStatus() *jobstatus.Pipeline {
	return s.jobStatus
}
