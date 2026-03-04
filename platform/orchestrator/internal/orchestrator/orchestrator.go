// Package orchestrator coordinates the lifecycle of scanning jobs.
package orchestrator

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/mattboback/stageflow/packages/shared-go/events"
	scanners "github.com/mattboback/stageflow/packages/shared-go/scannerregistry"
	"github.com/mattboback/stageflow/packages/shared-go/storage"
	"github.com/mattboback/stageflow/platform/orchestrator/internal/db"
	"github.com/mattboback/stageflow/platform/orchestrator/internal/fsm"
	"github.com/mattboback/stageflow/platform/orchestrator/internal/podman"
)

const (
	inputTypeURLs = "urls"

	podNetnsModeBridge = "bridge"
	podNetnsModeHost   = "host"

	hostNetnsNATSURL       = "nats://127.0.0.1:4222"
	hostNetnsMinioEndpoint = "127.0.0.1:9000"
)

// Publisher abstracts job completion/failure emissions for testability.
type Publisher interface {
	PublishJobCompleted(ctx context.Context, payload *events.JobCompletedPayload) error
	PublishJobFailed(ctx context.Context, payload *events.JobFailedPayload) error
}

// PodmanClient abstracts Podman operations so orchestration logic can be unit tested.
type PodmanClient interface {
	CreatePod(ctx context.Context, req *podman.PodCreateRequest) (*podman.PodCreateResponse, error)
	StopPod(ctx context.Context, podID string) error
	RemovePod(ctx context.Context, podID string, force bool) error
	CreateVolume(ctx context.Context, name string) error
	InspectVolume(ctx context.Context, name string) (*podman.VolumeInfo, error)
	CreateContainer(ctx context.Context, req *podman.ContainerCreateRequest) (*podman.ContainerCreateResponse, error)
	StartContainer(ctx context.Context, containerID string) error
	WaitContainer(ctx context.Context, containerID string) (*podman.ContainerWaitResponse, error)
	GetContainerLogs(ctx context.Context, containerID string, stdout, stderr bool) (string, error)
	RemoveContainer(ctx context.Context, containerID string, force bool) error
	RemoveVolume(ctx context.Context, name string, force bool) error
}

// Orchestrator owns the job FSM, Podman pod/container lifecycle, and final aggregation.
type Orchestrator struct {
	podmanClient         PodmanClient
	database             *db.Database
	stateMachine         *fsm.StateMachine
	publisher            Publisher
	scannerRegistry      *scanners.Registry
	monitorWG            sync.WaitGroup
	natsURL              string
	minioEndpoint        string
	minioAccessKey       string
	minioSecretKey       string
	minioUseSSL          bool
	natsHost             string // NATS hostname for job containers
	minioHost            string // MinIO hostname for job containers
	extractionTimeout    time.Duration
	scanTimeout          time.Duration
	extractionImage      string
	scannerImage         string
	podNetwork           string
	podNetnsMode         string
	podHostMappings      []string // Custom host:ip mappings for pods (e.g., "mysite.com:169.254.1.2")
	pageLoadTimeout      int
	scrollTimeout        int
	stagingStorage       storage.Deleter
	storage              storage.Client // Full storage access for report generation
	deadlinePollInterval time.Duration
	deadlineSweepOnce    sync.Once
}

// Config wires dependencies and optional overrides into a new Orchestrator.
type Config struct {
	PodmanClient         PodmanClient
	Database             *db.Database
	Publisher            Publisher
	ScannerRegistry      *scanners.Registry // Optional: scanner registry (if nil, uses default config)
	NatsURL              string
	MinIOEndpoint        string
	MinIOAccessKey       string
	MinIOSecretKey       string
	MinIOUseSSL          bool
	NatsHost             string // NATS hostname for job containers on POD_NETWORK
	MinioHost            string // MinIO hostname for job containers on POD_NETWORK
	ExtractionTimeout    time.Duration
	ScanTimeout          time.Duration
	ExtractionImage      string
	ScannerImage         string
	PodNetwork           string
	PodNetnsMode         string   // Pod network namespace mode for job pods (bridge|host). Host mode is local-only.
	PodHostMappings      []string // Custom host:ip mappings for pods (e.g., "mysite.com:169.254.1.2")
	PageLoadTimeout      int
	ScrollTimeout        int
	StagingStorage       storage.Deleter
	Storage              storage.Client // Full storage for report generation
	DeadlinePollInterval time.Duration
}

// NewOrchestrator builds an Orchestrator, applying defaults.
func NewOrchestrator(config *Config) *Orchestrator {
	extractionTimeout := config.ExtractionTimeout
	if extractionTimeout == 0 {
		extractionTimeout = 5 * time.Minute
	}

	scanTimeout := config.ScanTimeout
	if scanTimeout == 0 {
		scanTimeout = 30 * time.Minute
	}

	extractionImage := config.ExtractionImage
	if extractionImage == "" {
		extractionImage = "localhost/stageflow/extractor:latest"
	}

	scannerImage := config.ScannerImage
	if scannerImage == "" {
		scannerImage = "localhost/stageflow/scanner-runner:latest"
	}

	natsHost := config.NatsHost
	if natsHost == "" {
		natsHost = "nats"
	}

	minioHost := config.MinioHost
	if minioHost == "" {
		minioHost = "minio"
	}

	pageLoadTimeout := config.PageLoadTimeout
	if pageLoadTimeout == 0 {
		pageLoadTimeout = 15_000
	}

	scrollTimeout := config.ScrollTimeout
	if scrollTimeout == 0 {
		scrollTimeout = 300
	}

	deadlinePollInterval := config.DeadlinePollInterval
	if deadlinePollInterval == 0 {
		deadlinePollInterval = 30 * time.Second
	}

	podNetnsMode := config.PodNetnsMode
	if podNetnsMode == "" {
		podNetnsMode = podNetnsModeBridge
	}

	// Resolve scanner registry; default to built‑ins when unset.
	registry := config.ScannerRegistry
	if registry == nil {
		defaultConfig := scanners.DefaultConfig()

		var err error

		registry, err = scanners.InitializeRegistry(defaultConfig)
		if err != nil {
			slog.Error("Failed to initialize default scanner registry", "error", err)
			// Fallback to empty registry so jobs fail deterministically instead of crashing.
			registry = scanners.NewRegistry(scannerImage)
		}
	}

	return &Orchestrator{
		podmanClient:         config.PodmanClient,
		database:             config.Database,
		stateMachine:         fsm.NewStateMachine(),
		publisher:            config.Publisher,
		scannerRegistry:      registry,
		natsURL:              config.NatsURL,
		minioEndpoint:        config.MinIOEndpoint,
		minioAccessKey:       config.MinIOAccessKey,
		minioSecretKey:       config.MinIOSecretKey,
		minioUseSSL:          config.MinIOUseSSL,
		natsHost:             natsHost,
		minioHost:            minioHost,
		extractionTimeout:    extractionTimeout,
		scanTimeout:          scanTimeout,
		extractionImage:      extractionImage,
		scannerImage:         scannerImage,
		podNetwork:           config.PodNetwork,
		podNetnsMode:         podNetnsMode,
		podHostMappings:      config.PodHostMappings,
		pageLoadTimeout:      pageLoadTimeout,
		scrollTimeout:        scrollTimeout,
		stagingStorage:       config.StagingStorage,
		storage:              config.Storage,
		deadlinePollInterval: deadlinePollInterval,
	}
}

// Start launches background maintenance loops.
func (o *Orchestrator) Start(ctx context.Context) {
	if ctx == nil {
		return
	}

	o.deadlineSweepOnce.Do(func() {
		go o.startDeadlineSweeper(ctx)
	})
}

// WaitForMonitors blocks until all container monitor goroutines have returned.
// This is primarily intended for tests to avoid schema teardown races.
func (o *Orchestrator) WaitForMonitors() {
	if o == nil {
		return
	}

	o.monitorWG.Wait()
}
