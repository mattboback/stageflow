package orchestrator

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"github.com/mattboback/stageflow/packages/shared-go/models"
	appjobs "github.com/mattboback/stageflow/platform/orchestrator/internal/application/jobs"
)

func (o *Orchestrator) startPlannedScanner(
	ctx context.Context,
	job *models.Job,
	plan *appjobs.ScannerLaunchPlan,
) error {
	if plan == nil {
		return errors.New("scanner launch plan is required")
	}

	scannerType := plan.Labels["scanner_type"]
	result, err := o.runtimeAdapter().StartScanner(ctx, job, plan)
	if result != nil && result.ContainerID != "" {
		slog.Info("Created scanner container", "scanner", scannerType, "container_id", result.ContainerID, "job_id", job.ID)
		o.recordInternalEvent(ctx, job.ID, "orchestrator.container.created", map[string]any{
			"component":    "scanner",
			"scanner_type": scannerType,
			"container_id": result.ContainerID,
		})
	}

	if err != nil {
		return err
	}

	slog.Info("Started scanner container", "scanner", scannerType, "container_id", result.ContainerID, "job_id", job.ID)
	o.recordInternalEvent(ctx, job.ID, "orchestrator.container.started", map[string]any{
		"component":    "scanner",
		"scanner_type": scannerType,
		"container_id": result.ContainerID,
	})

	o.spawnMonitorContainer(backgroundWithCorrelation(ctx), result.ContainerID, job.ID, "scanner-"+scannerType)

	return nil
}
func (o *Orchestrator) scannerLaunchPlannerConfig() appjobs.ScannerLaunchPlannerConfig {
	defaultScannerImage := o.scannerImageOverride
	if defaultScannerImage == "" && o.scannerRegistry == nil {
		defaultScannerImage = o.scannerImage
	}
	if defaultScannerImage == "" && o.scannerRegistry == nil {
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
