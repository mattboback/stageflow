package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/mattboback/stageflow/packages/shared-go/models"
	podman "github.com/mattboback/stageflow/platform/orchestrator/internal/adapters/runtime"
	appjobs "github.com/mattboback/stageflow/platform/orchestrator/internal/application/jobs"
)

// startScanning starts the scanner containers in the pod (one per scanner type).
// Multiple scanners can run in parallel within the same pod.
func (o *Orchestrator) startScanning(ctx context.Context, job *models.Job) error {
	scannerTypes := o.getScannerTypes(job.Config.Modules)
	if len(scannerTypes) == 0 {
		return fmt.Errorf("no scanners resolved for job %s", job.ID)
	}

	if job.State != models.JobStateScanning {
		if !o.canTransition(job.State, models.JobStateScanning) {
			msg := fmt.Sprintf("job %s cannot transition to SCANNING from %s", job.ID, job.State)
			slog.Warn(msg, "job_id", job.ID, "from_state", job.State)
			o.failJobSafeWithDetails(
				ctx,
				job.ID,
				"scanning",
				msg,
				stateTransitionDetails(job.State, models.JobStateScanning),
			)

			return fmt.Errorf("%s", msg)
		}

		if err := o.database.UpdateJobState(ctx, job.ID, models.JobStateScanning); err != nil {
			return fmt.Errorf("failed to update job state: %w", err)
		}

		job.State = models.JobStateScanning

		// Record scan start time for metrics
		if err := o.database.RecordScanStart(ctx, job.ID); err != nil {
			slog.Warn("Failed to record scan start", "job_id", job.ID, "error", err)
		}
	}

	slog.Info(
		"Starting scanners",
		"scanner_count",
		len(scannerTypes),
		"job_id",
		job.ID,
		"pod_id",
		job.PodID,
		"scanners",
		scannerTypes,
	)

	// Set expected scanners in database for completion tracking
	if err := o.database.SetExpectedScanners(ctx, job.ID, scannerTypes); err != nil {
		return fmt.Errorf("failed to set expected scanners: %w", err)
	}

	job.ExpectedScanners = scannerTypes
	job.CompletedScanners = []string{}
	job.ScannerResults = make(map[string]*models.ScannerResult)

	if err := o.startScannersWithTimeout(ctx, job, scannerTypes); err != nil {
		return fmt.Errorf("failed to start scanners: %w", err)
	}

	return nil
}

// getScannerTypes returns the list of scanner types to run based on modules config.
// It uses the scanner registry to resolve module names/aliases to scanner IDs.
func (o *Orchestrator) getScannerTypes(modules []string) []string {
	return o.scannerRegistry.ResolveModules(modules)
}

func (o *Orchestrator) startScannersWithTimeout(ctx context.Context, job *models.Job, scannerTypes []string) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, o.scanTimeout)
	defer cancel()

	errChan := make(chan error, len(scannerTypes))
	for _, scannerType := range scannerTypes {
		go func(st string) {
			errChan <- o.startSingleScanner(timeoutCtx, job, st)
		}(scannerType)
	}

	var firstErr error

	for range scannerTypes {
		select {
		case err := <-errChan:
			if err != nil && firstErr == nil {
				firstErr = err
			}
		case <-timeoutCtx.Done():
			if firstErr == nil {
				firstErr = fmt.Errorf("scanner start timed out after %v", o.scanTimeout)
			}
		}
	}

	return firstErr
}

func (o *Orchestrator) startSingleScanner(ctx context.Context, job *models.Job, scannerType string) error {
	plan, err := o.newScannerLaunchPlanner().Plan(ctx, job, scannerType)
	if err != nil {
		return fmt.Errorf("plan scanner launch for %s: %w", scannerType, err)
	}

	return o.startPlannedScanner(ctx, job, plan)
}

func (o *Orchestrator) startPlannedScanner(
	ctx context.Context,
	job *models.Job,
	plan *appjobs.ScannerLaunchPlan,
) error {
	if plan == nil {
		return errors.New("scanner launch plan is required")
	}

	scannerType := plan.Labels["scanner_type"]

	mounts, err := o.resolveScannerMounts(ctx, plan.Volumes)
	if err != nil {
		return err
	}

	containerReq := &podman.ContainerCreateRequest{
		Name:   plan.Name,
		Image:  plan.Image,
		Pod:    job.PodID,
		User:   plan.User,
		Env:    plan.Env,
		Mounts: mounts,
		Labels: plan.Labels,
		ResourceLimits: &podman.ResourceLimits{
			MemoryLimitMB: plan.ResourceLimits.MemoryLimitMB,
			MemorySwapMB:  plan.ResourceLimits.MemorySwapMB,
		},
	}

	slog.Info("Launching scanner container", "scanner", scannerType, "image", plan.Image, "job_id", job.ID)

	containerResp, err := o.podmanClient.CreateContainer(ctx, containerReq)
	if err != nil {
		return fmt.Errorf("failed to create %s scanner container: %w", scannerType, err)
	}

	slog.Info("Created scanner container", "scanner", scannerType, "container_id", containerResp.ID, "job_id", job.ID)
	o.recordInternalEvent(ctx, job.ID, "orchestrator.container.created", map[string]any{
		"component":    "scanner",
		"scanner_type": scannerType,
		"container_id": containerResp.ID,
	})

	if startErr := o.podmanClient.StartContainer(ctx, containerResp.ID); startErr != nil {
		return fmt.Errorf("failed to start %s scanner container: %w", scannerType, startErr)
	}

	slog.Info("Started scanner container", "scanner", scannerType, "container_id", containerResp.ID, "job_id", job.ID)
	o.recordInternalEvent(ctx, job.ID, "orchestrator.container.started", map[string]any{
		"component":    "scanner",
		"scanner_type": scannerType,
		"container_id": containerResp.ID,
	})

	o.spawnMonitorContainer(backgroundWithCorrelation(ctx), containerResp.ID, job.ID, "scanner-"+scannerType)

	return nil
}

func (o *Orchestrator) resolveScannerMounts(
	ctx context.Context,
	volumes []appjobs.VolumeRequirement,
) ([]podman.VolumeMount, error) {
	mounts := make([]podman.VolumeMount, 0, len(volumes))

	for _, volume := range volumes {
		volumeInfo, err := o.ensureVolume(ctx, volume.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare %s volume: %w", volume.Name, err)
		}

		mounts = append(mounts, podman.VolumeMount{
			Type:        "bind",
			Source:      volumeInfo.Mountpoint,
			Destination: volume.Destination,
			ReadOnly:    volume.ReadOnly,
		})
	}

	return mounts, nil
}

func (o *Orchestrator) scannerLaunchPlannerConfig() appjobs.ScannerLaunchPlannerConfig {
	defaultScannerImage := o.scannerImage
	if defaultScannerImage == "" {
		defaultScannerImage = "localhost/stageflow/scanner-runner:latest"
	}

	return appjobs.ScannerLaunchPlannerConfig{
		ScannerRegistry:      o.scannerRegistry,
		DefaultScannerImage:  defaultScannerImage,
		NatsHost:             o.natsHost,
		MinioHost:            o.minioHost,
		MinioAccessKey:       o.minioAccessKey,
		MinioSecretKey:       o.minioSecretKey,
		MinioUseSSL:          o.minioUseSSL,
		PageLoadTimeout:      o.pageLoadTimeout,
		ScrollTimeout:        o.scrollTimeout,
		PodNetnsMode:         o.podNetnsMode,
		DefaultScannerUser:   "0",
		OpenRouterAPIKey:     os.Getenv("OPENROUTER_API_KEY"),
		OpenRouterAppTitle:   os.Getenv("OPENROUTER_APP_TITLE"),
		OpenRouterAppReferer: os.Getenv("OPENROUTER_APP_REFERER"),
	}
}

func (o *Orchestrator) newScannerLaunchPlanner() *appjobs.ScannerLaunchPlanner {
	return appjobs.NewScannerLaunchPlanner(o.scannerLaunchPlannerConfig())
}

// getScannerImage returns the container image to use for a scanner type.
func (o *Orchestrator) getScannerImage(scannerType string) string {
	if o.scannerRegistry == nil {
		return o.scannerImage
	}

	return o.scannerRegistry.GetImage(scannerType)
}
