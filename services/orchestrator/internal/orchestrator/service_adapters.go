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
		orchestratorJobStore{orchestrator: o},
		orchestratorRuntime{orchestrator: o},
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

func (s orchestratorJobStore) ClaimJobCompletion(ctx context.Context, jobID string) (bool, error) {
	return s.orchestrator.database.ClaimJobCompletion(ctx, jobID)
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

func (s orchestratorJobStore) CompleteJobWithTerminalEvent(
	ctx context.Context,
	jobID string,
	payload *events.JobCompletedPayload,
) error {
	return s.orchestrator.database.CompleteJobWithTerminalEvent(ctx, jobID, payload)
}

func (s orchestratorJobStore) FailJobWithTerminalEvent(
	ctx context.Context,
	jobID, stage, errorMsg, errorDetails string,
	payload *events.JobFailedPayload,
) error {
	return s.orchestrator.database.FailJobWithTerminalEvent(ctx, jobID, stage, errorMsg, errorDetails, payload)
}

func (s orchestratorJobStore) ListUnpublishedTerminalEvents(
	ctx context.Context,
	jobID string,
) ([]appjobs.TerminalEvent, error) {
	records, err := s.orchestrator.database.ListUnpublishedTerminalEvents(ctx, jobID)
	if err != nil {
		return nil, err
	}

	out := make([]appjobs.TerminalEvent, 0, len(records))
	for _, record := range records {
		terminalEvent := appjobs.TerminalEvent{Event: record.Event}
		switch record.Event {
		case events.EventJobCompleted:
			var payload events.JobCompletedPayload
			if err := json.Unmarshal([]byte(record.PayloadJSON), &payload); err != nil {
				return nil, fmt.Errorf("decode job.completed terminal payload: %w", err)
			}
			terminalEvent.JobCompleted = &payload
		case events.EventJobFailed:
			var payload events.JobFailedPayload
			if err := json.Unmarshal([]byte(record.PayloadJSON), &payload); err != nil {
				return nil, fmt.Errorf("decode job.failed terminal payload: %w", err)
			}
			terminalEvent.JobFailed = &payload
		default:
			return nil, fmt.Errorf("unknown terminal event: %s", record.Event)
		}

		out = append(out, terminalEvent)
	}

	return out, nil
}

func (s orchestratorJobStore) MarkTerminalEventPublished(ctx context.Context, jobID, event string) error {
	return s.orchestrator.database.MarkTerminalEventPublished(ctx, jobID, event)
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
			slog.Info(
				"Created extraction worker container",
				"container_id",
				started.result.ContainerID,
				"job_id",
				job.ID,
			)
			r.orchestrator.recordInternalEvent(ctx, job.ID, "orchestrator.container.created", map[string]any{
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
