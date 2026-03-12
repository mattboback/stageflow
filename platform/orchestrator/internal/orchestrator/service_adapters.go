package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/mattboback/stageflow/packages/shared-go/models"
	podman "github.com/mattboback/stageflow/platform/orchestrator/internal/adapters/runtime"
	adapterstorage "github.com/mattboback/stageflow/platform/orchestrator/internal/adapters/storage"
	appjobs "github.com/mattboback/stageflow/platform/orchestrator/internal/application/jobs"
)

func (o *Orchestrator) newService() *appjobs.Service {
	defaultScannerImage := o.scannerImageOverride
	if defaultScannerImage == "" && o.scannerRegistry == nil {
		defaultScannerImage = o.scannerImage
	}
	if defaultScannerImage == "" && o.scannerRegistry == nil {
		defaultScannerImage = "localhost/stageflow/scanner-runner:latest"
	}

	return appjobs.NewService(
		orchestratorJobStore{orchestrator: o},
		orchestratorRuntime{orchestrator: o},
		adapterstorage.NewAggregator(o.storage, o.scannerRegistry),
		o.publisher,
		appjobs.WithScannerLaunchPlanner(appjobs.NewScannerLaunchPlanner(appjobs.ScannerLaunchPlannerConfig{
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
		})),
	)
}

func (o *Orchestrator) runtimeAdapter() *podman.JobRuntime {
	return o.jobRuntime
}

func (o *Orchestrator) refreshJobRuntime() {
	o.jobRuntime = podman.NewJobRuntime(podman.JobRuntimeConfig{
		Client:          o.podmanClient,
		ScannerRegistry: o.scannerRegistry,
		ExtractionImage: o.extractionImage,
		PodNetwork:      o.podNetwork,
		PodNetnsMode:    o.podNetnsMode,
		PodHostMappings: append([]string(nil), o.podHostMappings...),
		NatsHost:        o.natsHost,
		MinioHost:       o.minioHost,
		MinioAccessKey:  o.minioAccessKey,
		MinioSecretKey:  o.minioSecretKey,
		MinioUseSSL:     o.minioUseSSL,
	})
}

type orchestratorJobStore struct {
	orchestrator *Orchestrator
}

func (s orchestratorJobStore) CreateJobIfAbsent(ctx context.Context, job *models.Job) (bool, error) {
	return s.orchestrator.database.CreateJobIfAbsent(ctx, job)
}

func (s orchestratorJobStore) GetJob(ctx context.Context, jobID string) (*models.Job, error) {
	return s.orchestrator.database.GetJob(ctx, jobID)
}

func (s orchestratorJobStore) UpdateJobState(ctx context.Context, jobID string, state models.JobState) error {
	return s.orchestrator.database.UpdateJobState(ctx, jobID, state)
}

func (s orchestratorJobStore) RecordExtractionComplete(ctx context.Context, jobID string) error {
	return s.orchestrator.database.RecordExtractionComplete(ctx, jobID)
}

func (s orchestratorJobStore) RecordExtractionStart(ctx context.Context, jobID string) error {
	return s.orchestrator.database.RecordExtractionStart(ctx, jobID)
}

func (s orchestratorJobStore) RecordScanStart(ctx context.Context, jobID string) error {
	return s.orchestrator.database.RecordScanStart(ctx, jobID)
}

func (s orchestratorJobStore) RecordScanComplete(ctx context.Context, jobID string) error {
	return s.orchestrator.database.RecordScanComplete(ctx, jobID)
}

func (s orchestratorJobStore) UpdateJobProgress(ctx context.Context, jobID string, currentPage, totalPages int) error {
	return s.orchestrator.database.UpdateJobProgress(ctx, jobID, currentPage, totalPages)
}

func (s orchestratorJobStore) UpdateJobExtractionArtifacts(
	ctx context.Context,
	jobID, stageLogPath, recipePath string,
) error {
	return s.orchestrator.database.UpdateJobExtractionArtifacts(ctx, jobID, stageLogPath, recipePath)
}

func (s orchestratorJobStore) UpdateJobProvenance(ctx context.Context, jobID, provenancePath string) error {
	return s.orchestrator.database.UpdateJobProvenance(ctx, jobID, provenancePath)
}

func (s orchestratorJobStore) UpdateJobProvenanceKey(ctx context.Context, jobID, provenanceKey string) error {
	return s.orchestrator.database.UpdateJobProvenanceKey(ctx, jobID, provenanceKey)
}

func (s orchestratorJobStore) UpdateJobPodID(ctx context.Context, jobID, podID string) error {
	return s.orchestrator.database.UpdateJobPodID(ctx, jobID, podID)
}

func (s orchestratorJobStore) UpdateJobCompletionArtifacts(
	ctx context.Context,
	jobID, reportJSONPath, reportHTMLPath, stageLogPath, recipePath string,
	totalIssues int,
) error {
	return s.orchestrator.database.UpdateJobCompletionArtifacts(
		ctx,
		jobID,
		reportJSONPath,
		reportHTMLPath,
		stageLogPath,
		recipePath,
		totalIssues,
	)
}

func (s orchestratorJobStore) UpdateJobMetrics(
	ctx context.Context,
	jobID string,
	pagesScanned, totalIssues, criticalIssues, seriousIssues, moderateIssues, minorIssues int,
) error {
	return s.orchestrator.database.UpdateJobMetrics(
		ctx,
		jobID,
		pagesScanned,
		totalIssues,
		criticalIssues,
		seriousIssues,
		moderateIssues,
		minorIssues,
	)
}

func (s orchestratorJobStore) SetExpectedScanners(ctx context.Context, jobID string, scanners []string) error {
	return s.orchestrator.database.SetExpectedScanners(ctx, jobID, scanners)
}

func (s orchestratorJobStore) RecordScannerCompletion(
	ctx context.Context,
	jobID string,
	result *models.ScannerResult,
) (bool, error) {
	return s.orchestrator.database.RecordScannerCompletion(ctx, jobID, result)
}

func (s orchestratorJobStore) RecordScannerFailure(
	ctx context.Context,
	jobID, scannerType, errorMsg string,
) (bool, error) {
	return s.orchestrator.database.RecordScannerFailure(ctx, jobID, scannerType, errorMsg)
}

func (s orchestratorJobStore) CompleteJob(ctx context.Context, jobID string) error {
	return s.orchestrator.database.CompleteJob(ctx, jobID)
}

func (s orchestratorJobStore) FailJob(
	ctx context.Context,
	jobID, stage, errorMsg, errorDetails string,
) error {
	return s.orchestrator.database.FailJob(ctx, jobID, stage, errorMsg, errorDetails)
}

func (s orchestratorJobStore) RecordInternalEvent(ctx context.Context, jobID, event string, payload any) error {
	s.orchestrator.recordInternalEvent(ctx, jobID, event, payload)
	return nil
}

type orchestratorRuntime struct {
	orchestrator *Orchestrator
}

func (r orchestratorRuntime) PodNetnsMode() string {
	return r.orchestrator.runtimeAdapter().PodNetnsMode()
}

func (r orchestratorRuntime) AllowsLoopbackTargets() bool {
	return r.orchestrator.runtimeAdapter().AllowsLoopbackTargets()
}

func (r orchestratorRuntime) CreateJobPod(ctx context.Context, job *models.Job) (string, error) {
	return r.orchestrator.runtimeAdapter().CreateJobPod(ctx, job)
}

func (r orchestratorRuntime) StartExtractionWorker(ctx context.Context, job *models.Job) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, r.orchestrator.extractionTimeout)
	defer cancel()

	type extractionStartResult struct {
		result *podman.ContainerLaunchResult
		err    error
	}

	resultChan := make(chan extractionStartResult, 1)

	go func() {
		result, err := r.orchestrator.runtimeAdapter().StartExtractionWorker(timeoutCtx, job, job.PodID)
		resultChan <- extractionStartResult{result: result, err: err}
	}()

	select {
	case started := <-resultChan:
		if started.result != nil && started.result.ContainerID != "" {
			slog.Info("Created extraction worker container", "container_id", started.result.ContainerID, "job_id", job.ID)
			r.orchestrator.recordInternalEvent(ctx, job.ID, "orchestrator.container.created", map[string]any{
				"component":    "extraction-worker",
				"container_id": started.result.ContainerID,
			})
		}

		if started.err != nil {
			return started.err
		}

		slog.Info("Started extraction worker container", "container_id", started.result.ContainerID, "job_id", job.ID)
		r.orchestrator.recordInternalEvent(ctx, job.ID, "orchestrator.container.started", map[string]any{
			"component":    "extraction-worker",
			"container_id": started.result.ContainerID,
		})
		r.orchestrator.spawnMonitorContainer(
			backgroundWithCorrelation(ctx),
			started.result.ContainerID,
			job.ID,
			"extraction",
		)

		return nil
	case <-timeoutCtx.Done():
		return fmt.Errorf("extraction worker start timed out after %v", r.orchestrator.extractionTimeout)
	}
}

func (r orchestratorRuntime) ResolveScannerTypes(modules []string) []string {
	return r.orchestrator.runtimeAdapter().ResolveScannerTypes(modules)
}

func (r orchestratorRuntime) StartScanner(
	ctx context.Context,
	job *models.Job,
	plan *appjobs.ScannerLaunchPlan,
) error {
	return r.orchestrator.startPlannedScanner(ctx, job, plan)
}

func (r orchestratorRuntime) CleanupJob(ctx context.Context, job *models.Job) error {
	if job == nil {
		return nil
	}

	if job.PodID != "" {
		if err := r.orchestrator.cleanupPod(ctx, job.PodID); err != nil {
			return err
		}
	}

	r.orchestrator.cleanupVolumes(ctx, job.ID)
	r.orchestrator.cleanupStaging(ctx, job)

	return nil
}
