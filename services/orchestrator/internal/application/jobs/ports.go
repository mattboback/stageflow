package jobs

import (
	"context"

	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/models"
)

type JobStore interface {
	CreateJobIfAbsent(ctx context.Context, job *models.Job) (bool, error)
	GetJob(ctx context.Context, jobID string) (*models.Job, error)
	UpdateJobState(ctx context.Context, jobID string, state models.JobState) error
	RecordExtractionComplete(ctx context.Context, jobID string) error
	RecordExtractionStart(ctx context.Context, jobID string) error
	RecordScanStart(ctx context.Context, jobID string) error
	RecordScanComplete(ctx context.Context, jobID string) error
	UpdateJobProgress(ctx context.Context, jobID string, currentPage, totalPages int) error
	UpdateJobExtractionArtifacts(ctx context.Context, jobID, stageLogPath, recipePath string) error
	UpdateJobProvenance(ctx context.Context, jobID, provenancePath string) error
	UpdateJobProvenanceKey(ctx context.Context, jobID, provenanceKey string) error
	UpdateJobPodID(ctx context.Context, jobID, podID string) error
	UpdateJobCompletionArtifacts(
		ctx context.Context,
		jobID, reportJSONPath, reportHTMLPath, stageLogPath, recipePath string,
		totalIssues int,
	) error
	UpdateJobMetrics(
		ctx context.Context,
		jobID string,
		pagesScanned, totalIssues, criticalIssues, seriousIssues, moderateIssues, minorIssues int,
	) error
	SetExpectedScanners(ctx context.Context, jobID string, scanners []string) error
	RecordScannerCompletion(ctx context.Context, jobID string, result *models.ScannerResult) (bool, error)
	RecordScannerFailure(ctx context.Context, jobID, scannerType, errorMsg string) (bool, error)
	CompleteJob(ctx context.Context, jobID string) error
	FailJob(ctx context.Context, jobID, stage, errorMsg, errorDetails string) error
	RecordInternalEvent(ctx context.Context, jobID, event string, payload any) error
}

type Runtime interface {
	PodNetnsMode() string
	AllowsLoopbackTargets() bool
	CreateJobPod(ctx context.Context, job *models.Job) (string, error)
	StartExtractionWorker(ctx context.Context, job *models.Job) error
	ResolveScannerTypes(modules []string) []string
	StartScanner(ctx context.Context, job *models.Job, plan *ScannerLaunchPlan) error
	CleanupJob(ctx context.Context, job *models.Job) error
}

type Artifacts interface {
	BuildAggregatedReport(ctx context.Context, job *models.Job) (string, error)
}

type Publisher interface {
	PublishJobCompleted(ctx context.Context, payload *events.JobCompletedPayload) error
	PublishJobFailed(ctx context.Context, payload *events.JobFailedPayload) error
}
