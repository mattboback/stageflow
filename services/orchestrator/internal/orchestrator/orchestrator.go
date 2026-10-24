package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/mattboback/stageflow/internal/jobstore"
	"github.com/mattboback/stageflow/libs/go/models"
	"github.com/mattboback/stageflow/services/orchestrator/internal/aggregate"
	"github.com/mattboback/stageflow/services/orchestrator/internal/podman"
)

type PodmanClient interface {
	CreatePod(context.Context, *podman.PodCreateRequest) (*podman.PodCreateResponse, error)
	InspectPod(context.Context, string) (*podman.PodInfo, error)
	RemovePod(context.Context, string, bool) error
	CreateVolume(context.Context, string) error
	InspectVolume(context.Context, string) (*podman.VolumeInfo, error)
	CreateContainer(context.Context, *podman.ContainerCreateRequest) (*podman.ContainerCreateResponse, error)
	InspectContainer(context.Context, string) (*podman.ContainerInfo, error)
	StartContainer(context.Context, string) error
	ListJobContainers(context.Context) ([]podman.ContainerInfo, error)
	ImageExists(context.Context, string) (bool, error)
}

type Config struct {
	Database                                                                 *jobstore.Database
	PodmanClient                                                             PodmanClient
	Files                                                                    aggregate.JobFiles
	DataDir, DataVolume                                                      string
	DirectoryOwner                                                           func(string) error
	ExtractionImage, ScannerImage                                            string
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
	aggregator    *aggregate.Aggregator
	once          sync.Once
	wg            sync.WaitGroup
	tickMu        sync.Mutex
	lastRetention time.Time

	imagesMu        sync.Mutex
	imagesCheckedAt time.Time
	imagesErr       error
}

// ErrImagesMissing means a job image is absent from the host. Claiming waits,
// because a job claimed now would fail when its containers could not be created.
var ErrImagesMissing = errors.New("job images are missing")

// imageCheckInterval bounds how often Health asks Podman about the job images:
// claiming consults it every reconcile tick.
const imageCheckInterval = 30 * time.Second

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
			ExtractionImage: cfg.ExtractionImage,
			PodNetwork:      cfg.PodNetwork,
			PodNetnsMode:    cfg.PodNetnsMode,
			PodHostMappings: cfg.PodHostMappings,
		},
	)

	planner := podman.NewScannerLaunchPlanner(
		podman.ScannerLaunchPlannerConfig{
			DataVolume:      cfg.DataVolume,
			ScannerImage:    cfg.ScannerImage,
			PageLoadTimeout: cfg.PageLoadTimeout,
		},
	)

	return &Orchestrator{
		config:     cfg,
		runtime:    runtime,
		planner:    planner,
		aggregator: aggregate.New(cfg.Files),
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
				if err := o.ReconcileOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
					slog.Warn("Job reconciliation failed", "error", err)
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

// Wait blocks until the reconcile loop has stopped.
func (o *Orchestrator) Wait() { o.wg.Wait() }

// Health reports whether the orchestrator can run a job: enough free disk and
// both job images present. The healthcheck and claiming both use it.
func (o *Orchestrator) Health(ctx context.Context) error {
	if o.config.CheckDisk != nil {
		if err := o.config.CheckDisk(ctx); err != nil {
			return err
		}
	}

	return o.checkImages(ctx)
}

func (o *Orchestrator) checkImages(ctx context.Context) error {
	o.imagesMu.Lock()
	defer o.imagesMu.Unlock()

	if !o.imagesCheckedAt.IsZero() && time.Since(o.imagesCheckedAt) < imageCheckInterval {
		return o.imagesErr
	}

	var missing []string

	for _, image := range []string{o.config.ExtractionImage, o.config.ScannerImage} {
		exists, err := o.config.PodmanClient.ImageExists(ctx, image)
		if err != nil {
			return err
		}

		if !exists {
			missing = append(missing, image)
		}
	}

	var result error
	if len(missing) > 0 {
		result = fmt.Errorf("%w: %s", ErrImagesMissing, strings.Join(missing, ", "))
	}

	switch {
	case result != nil && o.imagesErr == nil:
		slog.Error("Job images are missing; claiming paused", "missing", missing)
	case result == nil && o.imagesErr != nil:
		slog.Info("Job images are present; claiming resumed")
	}

	o.imagesCheckedAt, o.imagesErr = time.Now(), result

	return result
}

func (o *Orchestrator) fail(ctx context.Context, job *models.Job, stage, message string) error {
	return o.config.Database.MarkTerminal(ctx, job.ID, models.JobStateFailed, "", 0, message, "", stage)
}
