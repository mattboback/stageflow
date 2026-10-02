package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

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
		appjobs.WithAuthCleaner(authFileCleaner{o.storage}),
		appjobs.WithScannerLaunchPlanner(appjobs.NewScannerLaunchPlanner(appjobs.ScannerLaunchPlannerConfig{
			ScannerRegistry:     o.scannerRegistry,
			DataVolume:          o.dataVolume,
			DefaultScannerImage: defaultScannerImage,
			NatsHost:            o.natsHost,
			PageLoadTimeout:     o.pageLoadTimeout,
			PodNetnsMode:        o.podNetnsMode,
			DefaultScannerUser:  "65532",
		})),
	)
}

func (o *Orchestrator) refreshJobRuntime() {
	o.jobRuntime = podman.NewJobRuntime(podman.JobRuntimeConfig{
		Client:          o.podmanClient,
		DataDir:         o.dataDir,
		DirectoryOwner:  o.directoryOwner,
		DataVolume:      o.dataVolume,
		ScannerRegistry: o.scannerRegistry,
		ExtractionImage: o.extractionImage,
		PodNetwork:      o.podNetwork,
		PodNetnsMode:    o.podNetnsMode,
		PodHostMappings: append([]string(nil), o.podHostMappings...),
		NatsHost:        o.natsHost,
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
	jobID, reportJSONPath, stageLogPath, recipePath string,
	totalIssues int,
) error {
	return o.database.UpdateJobCompletionArtifacts(
		ctx,
		jobID,
		reportJSONPath,
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

func (o *Orchestrator) ListJobsByState(
	ctx context.Context,
	state models.JobState,
) ([]*models.Job, error) {
	return o.database.ListJobsByState(ctx, state)
}

func (o *Orchestrator) PrepareScannerLaunches(ctx context.Context, jobID string, scanners []string) error {
	return o.database.PrepareScannerLaunches(ctx, jobID, scanners)
}

func (o *Orchestrator) ClaimScannerLaunch(ctx context.Context, jobID, scannerType string) (bool, error) {
	return o.database.ClaimScannerLaunch(ctx, jobID, scannerType)
}

func (o *Orchestrator) ClaimScannerLaunchRecovery(
	ctx context.Context,
	jobID, scannerType string,
) (bool, error) {
	return o.database.ClaimScannerLaunchRecovery(ctx, jobID, scannerType)
}

func (o *Orchestrator) MarkScannerLaunched(
	ctx context.Context,
	jobID, scannerType, containerID string,
) error {
	return o.database.MarkScannerLaunched(ctx, jobID, scannerType, containerID)
}

func (o *Orchestrator) MarkScannerLaunchFailed(
	ctx context.Context,
	jobID, scannerType, errorMessage string,
) error {
	return o.database.MarkScannerLaunchFailed(ctx, jobID, scannerType, errorMessage)
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
) (bool, error) {
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

		o.spawnMonitorContainer(
			o.backgroundWithCorrelation(ctx),
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
) (string, error) {
	return o.startPlannedScanner(ctx, job, plan)
}

func (o *Orchestrator) CleanupJob(ctx context.Context, job *models.Job) error {
	if job == nil {
		return nil
	}

	// Fall back to the deterministic name when no pod ID was recorded. Podman can
	// create the pod and lose the response, so an empty PodID does not mean no pod
	// exists -- and without this the orphan is never removed by anything.
	// Stop/Remove both accept a name in place of an ID.
	podRef := job.PodID
	if podRef == "" {
		podRef = podman.JobPodName(job.ID)
	}

	if err := o.cleanupPod(ctx, podRef); err != nil {
		return err
	}

	root, err := os.OpenRoot(filepath.Join(o.dataDir, "jobs", job.ID))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	if err != nil {
		return err
	}

	defer root.Close()

	var errs []error

	for _, name := range []string{"input.zip", "site", "auth.json", ".provenance"} {
		if removeErr := root.RemoveAll(name); removeErr != nil {
			errs = append(errs, removeErr)
		}
	}

	return errors.Join(errs...)
}

func (o *Orchestrator) UpdateScannerProgress(ctx context.Context, id, scanner string, page, total int) error {
	return o.database.UpdateScannerProgress(ctx, id, scanner, page, total)
}
