package orchestrator

import (
	"context"

	"github.com/mattboback/stageflow/packages/shared-go/models"
	adapterstorage "github.com/mattboback/stageflow/platform/orchestrator/internal/adapters/storage"
	appjobs "github.com/mattboback/stageflow/platform/orchestrator/internal/application/jobs"
)

func (o *Orchestrator) newService() *appjobs.Service {
	return appjobs.NewService(
		orchestratorJobStore{orchestrator: o},
		orchestratorRuntime{orchestrator: o},
		adapterstorage.NewAggregator(o.storage, o.scannerRegistry),
		o.publisher,
	)
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

func (r orchestratorRuntime) StartExtraction(ctx context.Context, job *models.Job) error {
	return r.orchestrator.startExtraction(ctx, job)
}

func (r orchestratorRuntime) PrepareURLJob(ctx context.Context, job *models.Job) error {
	if err := r.orchestrator.validateURLJobTargets(ctx, job); err != nil {
		return err
	}

	if err := r.orchestrator.ensureURLJobPod(ctx, job); err != nil {
		return err
	}

	if err := r.orchestrator.ensureURLJobReady(ctx, job); err != nil {
		return err
	}

	r.orchestrator.persistURLJobProvenanceKey(ctx, job)

	return nil
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
