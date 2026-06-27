package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/models"
	podman "github.com/mattboback/stageflow/services/orchestrator/internal/adapters/runtime"
	adapterstorage "github.com/mattboback/stageflow/services/orchestrator/internal/adapters/storage"
	appjobs "github.com/mattboback/stageflow/services/orchestrator/internal/application/jobs"
)

const defaultLocalScannerImage = "localhost/stageflow/scanner-runner:latest"

func (o *Orchestrator) newService() *appjobs.Service {
	defaultScannerImage := o.scannerImageOverride

	if defaultScannerImage == "" && o.scannerRegistry == nil {
		defaultScannerImage = o.scannerImage
	}

	if defaultScannerImage == "" && o.scannerRegistry == nil {
		defaultScannerImage = defaultLocalScannerImage
	}

	return appjobs.NewService(
		o,
		o,
		adapterstorage.NewAggregator(o.storage, o.scannerRegistry),
		o.publisher,
		appjobs.WithAuthUploader(adapterstorage.NewAuthStorageStateUploader(o.storage)),
		appjobs.WithAuthCleaner(adapterstorage.NewAuthStorageStateCleaner(o.storage)),
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
			OpenRouterAPIKey:     o.openRouterAPIKey,
			OpenRouterAppTitle:   o.openRouterAppTitle,
			OpenRouterAppReferer: o.openRouterAppReferer,
			HostEnv:              os.Getenv,
		})),
	)
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

func (o *Orchestrator) CreateJobIfAbsent(ctx context.Context, job *models.Job) (bool, error) {
	return o.database.CreateJobIfAbsent(ctx, job)
}

func (o *Orchestrator) GetJob(ctx context.Context, jobID string) (*models.Job, error) {
	return o.database.GetJob(ctx, jobID)
}

func (o *Orchestrator) UpdateJobState(ctx context.Context, jobID string, state models.JobState) error {
	return o.database.UpdateJobState(ctx, jobID, state)
}

func (o *Orchestrator) ClaimJobCompletion(ctx context.Context, jobID string) (bool, error) {
	return o.database.ClaimJobCompletion(ctx, jobID)
}

func (o *Orchestrator) RecordExtractionComplete(ctx context.Context, jobID string) error {
	return o.database.RecordExtractionComplete(ctx, jobID)
}

func (o *Orchestrator) RecordExtractionStart(ctx context.Context, jobID string) error {
	return o.database.RecordExtractionStart(ctx, jobID)
}

func (o *Orchestrator) RecordScanStart(ctx context.Context, jobID string) error {
	return o.database.RecordScanStart(ctx, jobID)
}

func (o *Orchestrator) RecordScanComplete(ctx context.Context, jobID string) error {
	return o.database.RecordScanComplete(ctx, jobID)
}

func (o *Orchestrator) UpdateJobProgress(ctx context.Context, jobID string, currentPage, totalPages int) error {
	return o.database.UpdateJobProgress(ctx, jobID, currentPage, totalPages)
}

func (o *Orchestrator) UpdateJobExtractionArtifacts(
	ctx context.Context,
	jobID, stageLogPath, recipePath string,
) error {
	return o.database.UpdateJobExtractionArtifacts(ctx, jobID, stageLogPath, recipePath)
}

func (o *Orchestrator) UpdateJobProvenance(ctx context.Context, jobID, provenancePath string) error {
	return o.database.UpdateJobProvenance(ctx, jobID, provenancePath)
}

func (o *Orchestrator) UpdateJobProvenanceKey(ctx context.Context, jobID, provenanceKey string) error {
	return o.database.UpdateJobProvenanceKey(ctx, jobID, provenanceKey)
}

func (o *Orchestrator) UpdateJobPodID(ctx context.Context, jobID, podID string) error {
	return o.database.UpdateJobPodID(ctx, jobID, podID)
}

func (o *Orchestrator) UpdateJobCompletionArtifacts(
	ctx context.Context,
	jobID, reportJSONPath, reportHTMLPath, stageLogPath, recipePath string,
	totalIssues int,
) error {
	return o.database.UpdateJobCompletionArtifacts(
		ctx,
		jobID,
		reportJSONPath,
		reportHTMLPath,
		stageLogPath,
		recipePath,
		totalIssues,
	)
}

func (o *Orchestrator) UpdateJobMetrics(
	ctx context.Context,
	jobID string,
	pagesScanned, totalIssues, criticalIssues, seriousIssues, moderateIssues, minorIssues int,
) error {
	return o.database.UpdateJobMetrics(
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

func (o *Orchestrator) SetExpectedScanners(ctx context.Context, jobID string, scanners []string) error {
	return o.database.SetExpectedScanners(ctx, jobID, scanners)
}

func (o *Orchestrator) RecordScannerCompletion(
	ctx context.Context,
	jobID string,
	result *models.ScannerResult,
) (bool, error) {
	return o.database.RecordScannerCompletion(ctx, jobID, result)
}

func (o *Orchestrator) RecordScannerFailure(
	ctx context.Context,
	jobID, scannerType, errorMsg string,
) (bool, error) {
	return o.database.RecordScannerFailure(ctx, jobID, scannerType, errorMsg)
}

func (o *Orchestrator) CompleteJobWithTerminalEvent(
	ctx context.Context,
	jobID string,
	payload *events.JobCompletedPayload,
) error {
	return o.database.CompleteJobWithTerminalEvent(ctx, jobID, payload)
}

func (o *Orchestrator) FailJobWithTerminalEvent(
	ctx context.Context,
	jobID, stage, errorMsg, errorDetails string,
	payload *events.JobFailedPayload,
) error {
	return o.database.FailJobWithTerminalEvent(ctx, jobID, stage, errorMsg, errorDetails, payload)
}

func (o *Orchestrator) ListUnpublishedTerminalEvents(
	ctx context.Context,
	jobID string,
) ([]appjobs.TerminalEvent, error) {
	records, err := o.database.ListUnpublishedTerminalEvents(ctx, jobID)
	if err != nil {
		return nil, err
	}

	out := make([]appjobs.TerminalEvent, 0, len(records))
	for _, record := range records {
		terminalEvent := appjobs.TerminalEvent{Event: record.Event}
		switch record.Event {
		case events.EventJobCompleted:
			var payload events.JobCompletedPayload
			if unmarshalErr := json.Unmarshal([]byte(record.PayloadJSON), &payload); unmarshalErr != nil {
				return nil, fmt.Errorf("decode job.completed terminal payload: %w", unmarshalErr)
			}

			terminalEvent.JobCompleted = &payload
		case events.EventJobFailed:
			var payload events.JobFailedPayload
			if unmarshalErr := json.Unmarshal([]byte(record.PayloadJSON), &payload); unmarshalErr != nil {
				return nil, fmt.Errorf("decode job.failed terminal payload: %w", unmarshalErr)
			}

			terminalEvent.JobFailed = &payload
		default:
			return nil, fmt.Errorf("unknown terminal event: %s", record.Event)
		}

		out = append(out, terminalEvent)
	}

	return out, nil
}

func (o *Orchestrator) MarkTerminalEventPublished(ctx context.Context, jobID, event string) error {
	return o.database.MarkTerminalEventPublished(ctx, jobID, event)
}

func (o *Orchestrator) RecordInternalEvent(ctx context.Context, jobID, event string, payload any) error {
	o.recordInternalEvent(ctx, jobID, event, payload)
	return nil
}

func (o *Orchestrator) PodNetnsMode() string {
	return o.jobRuntime.PodNetnsMode()
}

func (o *Orchestrator) AllowsLoopbackTargets() bool {
	return o.jobRuntime.AllowsLoopbackTargets()
}

func (o *Orchestrator) CreateJobPod(ctx context.Context, job *models.Job) (string, error) {
	return o.jobRuntime.CreateJobPod(ctx, job)
}

func (o *Orchestrator) StartExtractionWorker(ctx context.Context, job *models.Job) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, o.extractionTimeout)
	defer cancel()

	type extractionStartResult struct {
		result *podman.ContainerLaunchResult
		err    error
	}

	resultChan := make(chan extractionStartResult, 1)

	go func() {
		result, err := o.jobRuntime.StartExtractionWorker(timeoutCtx, job, job.PodID)
		resultChan <- extractionStartResult{result: result, err: err}
	}()

	select {
	case started := <-resultChan:
		if started.result != nil && started.result.ContainerID != "" {
			slog.Info(
				"Created extraction worker container",
				"container_id",
				started.result.ContainerID,
				"job_id",
				job.ID,
			)
			o.recordInternalEvent(ctx, job.ID, "orchestrator.container.created", map[string]any{
				"component":    "extraction-worker",
				"container_id": started.result.ContainerID,
			})
		}

		if started.err != nil {
			return started.err
		}

		slog.Info(
			"Started extraction worker container",
			"container_id",
			started.result.ContainerID,
			"job_id",
			job.ID,
		)
		o.recordInternalEvent(ctx, job.ID, "orchestrator.container.started", map[string]any{
			"component":    "extraction-worker",
			"container_id": started.result.ContainerID,
		})
		o.spawnMonitorContainer(
			backgroundWithCorrelation(ctx),
			started.result.ContainerID,
			job.ID,
			"extraction",
		)

		return nil
	case <-timeoutCtx.Done():
		return fmt.Errorf("extraction worker start timed out after %v", o.extractionTimeout)
	}
}

func (o *Orchestrator) ResolveScannerTypes(modules []string) []string {
	return o.jobRuntime.ResolveScannerTypes(modules)
}

func (o *Orchestrator) StartScanner(
	ctx context.Context,
	job *models.Job,
	plan *appjobs.ScannerLaunchPlan,
) error {
	return o.startPlannedScanner(ctx, job, plan)
}

func (o *Orchestrator) CleanupJob(ctx context.Context, job *models.Job) error {
	if job == nil {
		return nil
	}

	if job.PodID != "" {
		if err := o.cleanupPod(ctx, job.PodID); err != nil {
			return err
		}
	}

	o.cleanupVolumes(ctx, job.ID)
	o.cleanupStaging(ctx, job)

	return nil
}
