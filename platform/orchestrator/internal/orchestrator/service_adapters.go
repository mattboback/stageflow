package orchestrator

import (
	"context"
	"reflect"

	"github.com/mattboback/stageflow/packages/shared-go/models"
	podman "github.com/mattboback/stageflow/platform/orchestrator/internal/adapters/runtime"
	adapterstorage "github.com/mattboback/stageflow/platform/orchestrator/internal/adapters/storage"
	appjobs "github.com/mattboback/stageflow/platform/orchestrator/internal/application/jobs"
)

type runtimeAdapterState struct {
	client          PodmanClient
	scannerRegistry any
	extractionImage string
	podNetwork      string
	podNetnsMode    string
	podHostMappings []string
	natsHost        string
	minioHost       string
	minioAccessKey  string
	minioSecretKey  string
	minioUseSSL     bool
}

func (o *Orchestrator) newService() *appjobs.Service {
	return appjobs.NewService(
		orchestratorJobStore{orchestrator: o},
		orchestratorRuntime{orchestrator: o},
		adapterstorage.NewAggregator(o.storage, o.scannerRegistry),
		o.publisher,
		o.scannerLaunchPlannerConfig(),
	)
}

func (o *Orchestrator) runtimeAdapter() *podman.JobRuntime {
	if o.jobRuntime == nil || !reflect.DeepEqual(o.jobRuntimeState, o.currentRuntimeAdapterState()) {
		o.refreshJobRuntime()
	}

	return o.jobRuntime
}

func (o *Orchestrator) currentRuntimeAdapterState() runtimeAdapterState {
	return runtimeAdapterState{
		client:          o.podmanClient,
		scannerRegistry: o.scannerRegistry,
		extractionImage: o.extractionImage,
		podNetwork:      o.podNetwork,
		podNetnsMode:    o.podNetnsMode,
		podHostMappings: append([]string(nil), o.podHostMappings...),
		natsHost:        o.natsHost,
		minioHost:       o.minioHost,
		minioAccessKey:  o.minioAccessKey,
		minioSecretKey:  o.minioSecretKey,
		minioUseSSL:     o.minioUseSSL,
	}
}

func (o *Orchestrator) refreshJobRuntime() {
	state := o.currentRuntimeAdapterState()
	o.jobRuntime = podman.NewJobRuntime(podman.JobRuntimeConfig{
		Client:          state.client,
		ScannerRegistry: o.scannerRegistry,
		ExtractionImage: state.extractionImage,
		PodNetwork:      state.podNetwork,
		PodNetnsMode:    state.podNetnsMode,
		PodHostMappings: state.podHostMappings,
		NatsHost:        state.natsHost,
		MinioHost:       state.minioHost,
		MinioAccessKey:  state.minioAccessKey,
		MinioSecretKey:  state.minioSecretKey,
		MinioUseSSL:     state.minioUseSSL,
	})
	o.jobRuntimeState = state
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
	return r.orchestrator.createJobPod(ctx, job)
}

func (r orchestratorRuntime) StartExtractionWorker(ctx context.Context, job *models.Job) error {
	return r.orchestrator.startExtractionWorkerWithTimeout(ctx, job, job.PodID)
}

func (r orchestratorRuntime) ResolveScannerTypes(modules []string) []string {
	return r.orchestrator.getScannerTypes(modules)
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
