package orchestrator

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	storage "github.com/mattboback/stageflow/internal/artifacts"
	db "github.com/mattboback/stageflow/internal/jobstore"
	"github.com/mattboback/stageflow/libs/go/models"
	scanners "github.com/mattboback/stageflow/libs/go/scannerregistry"
	podman "github.com/mattboback/stageflow/services/orchestrator/internal/adapters/runtime"
	adapterstorage "github.com/mattboback/stageflow/services/orchestrator/internal/adapters/storage"
)

type PodmanClient interface {
	CreatePod(context.Context, *podman.PodCreateRequest) (*podman.PodCreateResponse, error)
	InspectPod(context.Context, string) (*podman.PodInfo, error)
	StopPod(context.Context, string) error
	RemovePod(context.Context, string, bool) error
	CreateVolume(context.Context, string) error
	InspectVolume(context.Context, string) (*podman.VolumeInfo, error)
	CreateContainer(context.Context, *podman.ContainerCreateRequest) (*podman.ContainerCreateResponse, error)
	InspectContainer(context.Context, string) (*podman.ContainerInfo, error)
	StartContainer(context.Context, string) error
	RemoveContainer(context.Context, string, bool) error
	ListJobContainers(context.Context) ([]podman.ContainerInfo, error)
}

type Config struct {
	Database                                                                 *db.Database
	PodmanClient                                                             PodmanClient
	Storage                                                                  storage.Client
	DataDir, DataVolume                                                      string
	DirectoryOwner                                                           func(string) error
	ScannerRegistry                                                          *scanners.Registry
	ExtractionImage, ScannerImage, ScannerImageOverride                      string
	PodNetwork, PodNetnsMode                                                 string
	PodHostMappings                                                          []string
	PageLoadTimeout                                                          int
	SetupTimeout, ExtractionTimeout, ScanTimeout, QueueTimeout, PollInterval time.Duration
	CheckDisk                                                                func(context.Context) error
}

type Orchestrator struct {
	config        Config
	runtime       *podman.JobRuntime
	planner       *podman.ScannerLaunchPlanner
	aggregator    *adapterstorage.Aggregator
	once          sync.Once
	wg            sync.WaitGroup
	tickMu        sync.Mutex
	lastRetention time.Time
}

func NewOrchestrator(config *Config) *Orchestrator {
	cfg := *config
	if cfg.DataDir == "" {
		cfg.DataDir = "/data"
	}

	if cfg.DataVolume == "" {
		cfg.DataVolume = "stageflow_data"
	}

	if cfg.SetupTimeout == 0 {
		cfg.SetupTimeout = 10 * time.Minute
	}

	if cfg.ExtractionTimeout == 0 {
		cfg.ExtractionTimeout = 5 * time.Minute
	}

	if cfg.ScanTimeout == 0 {
		cfg.ScanTimeout = 30 * time.Minute
	}

	if cfg.QueueTimeout == 0 {
		cfg.QueueTimeout = 45 * time.Minute
	}

	if cfg.PollInterval == 0 {
		cfg.PollInterval = time.Second
	}

	runtime := podman.NewJobRuntime(
		podman.JobRuntimeConfig{
			Client:          cfg.PodmanClient,
			DataDir:         cfg.DataDir,
			DataVolume:      cfg.DataVolume,
			DirectoryOwner:  cfg.DirectoryOwner,
			ScannerRegistry: cfg.ScannerRegistry,
			ExtractionImage: cfg.ExtractionImage,
			PodNetwork:      cfg.PodNetwork,
			PodNetnsMode:    cfg.PodNetnsMode,
			PodHostMappings: cfg.PodHostMappings,
		},
	)

	image := cfg.ScannerImageOverride
	if image == "" {
		image = cfg.ScannerImage
	}

	planner := podman.NewScannerLaunchPlanner(
		podman.ScannerLaunchPlannerConfig{
			DataVolume:          cfg.DataVolume,
			ScannerRegistry:     cfg.ScannerRegistry,
			DefaultScannerImage: image,
			PageLoadTimeout:     cfg.PageLoadTimeout,
			PodNetnsMode:        cfg.PodNetnsMode,
		},
	)

	return &Orchestrator{
		config:     cfg,
		runtime:    runtime,
		planner:    planner,
		aggregator: adapterstorage.NewAggregator(cfg.Storage, cfg.ScannerRegistry),
	}
}

func (o *Orchestrator) Start(ctx context.Context) {
	o.once.Do(func() {
		o.wg.Add(1)
		go func() {
			defer o.wg.Done()

			ticker := time.NewTicker(o.config.PollInterval)
			defer ticker.Stop()

			for {
				if checkErr1 := o.ReconcileOnce(ctx); checkErr1 != nil && !errors.Is(checkErr1, context.Canceled) {
					slog.Warn("Job reconciliation failed", "error", checkErr1)
				}

				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
				}
			}
		}()
	})
}

func (o *Orchestrator) WaitForMonitors() { o.wg.Wait() }

func (o *Orchestrator) Health(ctx context.Context) error {
	if o.config.CheckDisk != nil {
		return o.config.CheckDisk(ctx)
	}

	return nil
}

func (o *Orchestrator) fail(ctx context.Context, job *models.Job, stage, message string) error {
	return o.config.Database.MarkTerminal(ctx, job.ID, models.JobStateFailed, "", 0, message, "", stage)
}
