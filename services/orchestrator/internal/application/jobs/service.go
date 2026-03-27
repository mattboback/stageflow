package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/models"
	domainjobs "github.com/mattboback/stageflow/services/orchestrator/internal/domain/jobs"
)

type Service struct {
	store     JobStore
	runtime   Runtime
	artifacts Artifacts
	publisher Publisher
	planner   *ScannerLaunchPlanner
}

type ServiceOption func(*Service)

func WithScannerLaunchPlanner(planner *ScannerLaunchPlanner) ServiceOption {
	return func(service *Service) {
		if planner != nil {
			service.planner = planner
		}
	}
}

func NewService(
	store JobStore,
	runtime Runtime,
	artifacts Artifacts,
	publisher Publisher,
	opts ...ServiceOption,
) *Service {
	service := &Service{
		store:     store,
		runtime:   runtime,
		artifacts: artifacts,
		publisher: publisher,
		planner:   NewScannerLaunchPlanner(ScannerLaunchPlannerConfig{}),
	}

	for _, opt := range opts {
		opt(service)
	}

	return service
}

func (s *Service) PrepareExtractedJob(ctx context.Context, payload *events.ExtractionReadyPayload) error {
	if err := s.store.RecordExtractionComplete(ctx, payload.JobID); err != nil {
		slog.Warn("Failed to record extraction complete", "job_id", payload.JobID, "error", err)
	}

	job, err := s.store.GetJob(ctx, payload.JobID)
	if err != nil {
		if shouldIgnoreMissingJob(events.EventExtractionReady, payload.JobID, err) {
			return nil
		}

		return fmt.Errorf("failed to get job: %w", err)
	}

	s.persistExtractionReadyMetadata(ctx, payload, job)

	if job.State != models.JobStateReady {
		action, actionErr := domainjobs.DecideExtractionReady(job.State)
		if actionErr != nil {
			return actionErr
		}

		switch action {
		case domainjobs.ExtractionReadyAlreadyReady:
		case domainjobs.ExtractionReadyIgnore:
			return nil
		case domainjobs.ExtractionReadyAdvance:
			if updateErr := s.store.UpdateJobState(ctx, job.ID, models.JobStateReady); updateErr != nil {
				return fmt.Errorf("failed to update job state to ready: %w", updateErr)
			}

			job.State = models.JobStateReady
		default:
			return fmt.Errorf("unsupported extraction.ready action: %s", action)
		}
	}

	return s.StartScanning(ctx, job)
}

func (s *Service) StartScanning(ctx context.Context, job *models.Job) error {
	scannerTypes := s.runtime.ResolveScannerTypes(job.Config.Modules)
	if len(scannerTypes) == 0 {
		return fmt.Errorf("no scanners resolved for job %s", job.ID)
	}

	if job.State != models.JobStateScanning {
		if !domainjobs.CanTransitionTo(job.State, models.JobStateScanning) {
			return fmt.Errorf("job %s cannot transition to SCANNING from %s", job.ID, job.State)
		}

		if err := s.store.UpdateJobState(ctx, job.ID, models.JobStateScanning); err != nil {
			return fmt.Errorf("failed to update job state to scanning: %w", err)
		}

		job.State = models.JobStateScanning

		if err := s.store.RecordScanStart(ctx, job.ID); err != nil {
			slog.Warn("Failed to record scan start", "job_id", job.ID, "error", err)
		}
	}

	if err := s.store.SetExpectedScanners(ctx, job.ID, scannerTypes); err != nil {
		return fmt.Errorf("failed to set expected scanners: %w", err)
	}

	job.ExpectedScanners = append([]string{}, scannerTypes...)
	job.CompletedScanners = nil
	job.ScannerResults = make(map[string]*models.ScannerResult)

	for _, scannerType := range scannerTypes {
		plan, err := s.planner.Plan(ctx, job, scannerType)
		if err != nil {
			return fmt.Errorf("plan scanner launch for %s: %w", scannerType, err)
		}

		if startErr := s.runtime.StartScanner(ctx, job, plan); startErr != nil {
			message := fmt.Sprintf("failed to start scanner %s", scannerType)
			s.failJobSafe(
				ctx,
				job.ID,
				"scanning",
				message,
				fmt.Sprintf("scanner=%s error=%v", scannerType, startErr),
			)

			return fmt.Errorf("start scanner %s: %w", scannerType, startErr)
		}
	}

	return nil
}

func (s *Service) RecordScannerFailure(ctx context.Context, payload *events.ScanFailedPayload) error {
	scannerType := payload.ScannerType
	if scannerType == "" {
		scannerType = "unknown"
	}

	job, err := s.store.GetJob(ctx, payload.JobID)
	if err != nil {
		if shouldIgnoreMissingJob(events.EventScanFailed, payload.JobID, err) {
			return nil
		}

		return fmt.Errorf("failed to get job: %w", err)
	}

	if domainjobs.ShouldIgnoreTerminalEvent(job.State, events.EventScanFailed) {
		return nil
	}

	if artifactsErr := s.store.UpdateJobCompletionArtifacts(
		ctx,
		payload.JobID,
		"",
		"",
		payload.StageLogPath,
		payload.RecipePath,
		0,
	); artifactsErr != nil {
		slog.Warn("Failed to persist scan failure artifacts", "job_id", payload.JobID, "error", artifactsErr)
	}

	allComplete, err := s.store.RecordScannerFailure(ctx, payload.JobID, scannerType, payload.Error)
	if err != nil {
		return fmt.Errorf("failed to record scanner failure: %w", err)
	}

	refreshedJob := job

	if allComplete {
		latestJob, refreshErr := s.store.GetJob(ctx, payload.JobID)
		if refreshErr != nil {
			if shouldIgnoreMissingJob(events.EventScanFailed, payload.JobID, refreshErr) {
				return nil
			}

			return fmt.Errorf("failed to refresh job: %w", refreshErr)
		}

		refreshedJob = latestJob
	}

	switch domainjobs.DecideScanFailureCompletion(refreshedJob, allComplete) {
	case domainjobs.ScanFailureWait:
		return nil
	case domainjobs.ScanFailureCompleteWithPartialResults:
		return s.CompleteJob(ctx, refreshedJob)
	case domainjobs.ScanFailureFailJob:
		return s.FailJob(ctx, payload.JobID, "scanning", "all scanners failed: "+payload.Error, payload.ErrorDetails)
	default:
		return errors.New("unsupported scan failure action")
	}
}

func (s *Service) RecordScannerCompletion(ctx context.Context, payload *events.ScanCompletedPayload) error {
	scannerType := payload.ScannerType
	if scannerType == "" {
		return errors.New("scan completed payload missing scanner type")
	}

	if payload.ResultsPath == "" {
		return errors.New("scan completed payload missing results path")
	}

	if err := s.store.UpdateJobProgress(
		ctx,
		payload.JobID,
		payload.TotalPagesScanned,
		payload.TotalPagesScanned,
	); err != nil {
		slog.Warn("Failed to persist scan completion progress", "job_id", payload.JobID, "error", err)
	}

	job, err := s.store.GetJob(ctx, payload.JobID)
	if err != nil {
		if shouldIgnoreMissingJob(events.EventScanCompleted, payload.JobID, err) {
			return nil
		}

		return fmt.Errorf("failed to get job: %w", err)
	}

	if domainjobs.ShouldIgnoreTerminalEvent(job.State, events.EventScanCompleted) {
		return nil
	}

	allComplete, err := s.store.RecordScannerCompletion(ctx, payload.JobID, &models.ScannerResult{
		ScannerType:    scannerType,
		ResultsPath:    payload.ResultsPath,
		ReportPath:     payload.ReportPath,
		StageLogPath:   payload.StageLogPath,
		RecipePath:     payload.RecipePath,
		Success:        true,
		PagesScanned:   payload.TotalPagesScanned,
		TotalIssues:    payload.Summary.TotalViolations,
		CriticalIssues: payload.Summary.CriticalIssues,
		SeriousIssues:  payload.Summary.SeriousIssues,
		ModerateIssues: payload.Summary.ModerateIssues,
		MinorIssues:    payload.Summary.MinorIssues,
	})
	if err != nil {
		return fmt.Errorf("failed to record scanner completion: %w", err)
	}

	if !allComplete {
		return nil
	}

	refreshedJob, err := s.store.GetJob(ctx, payload.JobID)
	if err != nil {
		if shouldIgnoreMissingJob(events.EventScanCompleted, payload.JobID, err) {
			return nil
		}

		return fmt.Errorf("failed to refresh job: %w", err)
	}

	return s.CompleteJob(ctx, refreshedJob)
}

func (s *Service) CompleteJob(ctx context.Context, job *models.Job) error {
	if job == nil {
		return errors.New("job is required")
	}

	if job.State != models.JobStateCompleting {
		if !domainjobs.CanEnterCompleting(job.State) {
			return fmt.Errorf("job %s cannot transition to COMPLETING from %s", job.ID, job.State)
		}

		if err := s.store.UpdateJobState(ctx, job.ID, models.JobStateCompleting); err != nil {
			return fmt.Errorf("failed to update job state to completing: %w", err)
		}

		job.State = models.JobStateCompleting
	}

	reportJSONPath, err := s.artifacts.BuildAggregatedReport(ctx, job)
	if err != nil {
		return err
	}

	if recordErr := s.store.RecordScanComplete(ctx, job.ID); recordErr != nil {
		slog.Warn("Failed to record scan complete", "job_id", job.ID, "error", recordErr)
	}

	primaryReportPath, primaryStageLogPath, primaryRecipePath, totalIssues := summarizeSuccessfulScannerArtifacts(job)
	scannerArtifacts := buildScannerArtifacts(job)

	if artifactsErr := s.store.UpdateJobCompletionArtifacts(
		ctx,
		job.ID,
		reportJSONPath,
		primaryReportPath,
		primaryStageLogPath,
		primaryRecipePath,
		totalIssues,
	); artifactsErr != nil {
		slog.Warn("Failed to persist completion artifacts", "job_id", job.ID, "error", artifactsErr)
	}

	if completeErr := s.store.CompleteJob(ctx, job.ID); completeErr != nil {
		return fmt.Errorf("complete job: %w", completeErr)
	}

	job.State = models.JobStateDone

	if cleanupErr := s.runtime.CleanupJob(ctx, job); cleanupErr != nil {
		slog.Warn("Failed to cleanup job", "job_id", job.ID, "error", cleanupErr)
	}

	return s.publisher.PublishJobCompleted(ctx, &events.JobCompletedPayload{
		JobID:  job.ID,
		Status: "success",
		Artifacts: events.ArtifactLocations{
			ReportJSON:     reportJSONPath,
			ReportHTML:     primaryReportPath,
			ScanStageLog:   primaryStageLogPath,
			ScanRecipe:     primaryRecipePath,
			ProvenanceJSON: job.ProvenanceKey,
		},
		ScannerArtifacts: scannerArtifacts,
	})
}

func (s *Service) FailJob(ctx context.Context, jobID, stage, message, details string) error {
	job, err := s.store.GetJob(ctx, jobID)
	if err != nil {
		return fmt.Errorf("failed to get job for failure handling: %w", err)
	}

	if domainjobs.IsTerminalState(job.State) {
		return nil
	}

	normalizedStage := domainjobs.NormalizeFailureStage(stage)

	if failErr := s.store.FailJob(ctx, jobID, normalizedStage, message, details); failErr != nil {
		return fmt.Errorf("failed to fail job: %w", failErr)
	}

	job.State = models.JobStateFailed

	if cleanupErr := s.runtime.CleanupJob(ctx, job); cleanupErr != nil {
		slog.Warn("Failed to cleanup job", "job_id", job.ID, "error", cleanupErr)
	}

	return s.publisher.PublishJobFailed(ctx, &events.JobFailedPayload{
		JobID:        jobID,
		Status:       "failed",
		Stage:        normalizedStage,
		Error:        message,
		ErrorDetails: details,
	})
}

func (s *Service) persistExtractionReadyMetadata(
	ctx context.Context,
	payload *events.ExtractionReadyPayload,
	job *models.Job,
) {
	if err := s.store.UpdateJobProgress(ctx, payload.JobID, 0, payload.TotalPages); err != nil {
		slog.Warn("Failed to persist total pages from extraction.ready", "job_id", payload.JobID, "error", err)
	} else {
		job.TotalPages = payload.TotalPages
		job.CurrentPage = 0
	}

	if err := s.store.UpdateJobExtractionArtifacts(
		ctx,
		payload.JobID,
		payload.StageLogPath,
		payload.RecipePath,
	); err != nil {
		slog.Warn("Failed to persist extraction artifacts", "job_id", payload.JobID, "error", err)
	} else {
		job.ExtractionStageLogKey = payload.StageLogPath
		job.ExtractionRecipeKey = payload.RecipePath
	}

	if payload.ProvenancePath != "" {
		if err := s.store.UpdateJobProvenance(ctx, payload.JobID, payload.ProvenancePath); err != nil {
			slog.Warn("Failed to persist provenance path", "job_id", payload.JobID, "error", err)
		} else {
			job.ProvenancePath = payload.ProvenancePath
		}
	}

	if payload.ProvenanceArtifactPath != "" {
		if err := s.store.UpdateJobProvenanceKey(ctx, payload.JobID, payload.ProvenanceArtifactPath); err != nil {
			slog.Warn("Failed to persist provenance key", "job_id", payload.JobID, "error", err)
		} else {
			job.ProvenanceKey = payload.ProvenanceArtifactPath
		}
	}
}

func summarizeSuccessfulScannerArtifacts(job *models.Job) (string, string, string, int) {
	if job == nil {
		return "", "", "", 0
	}

	successfulScanners := make([]string, 0, len(job.ScannerResults))
	totalIssues := 0

	for scannerType, result := range job.ScannerResults {
		if result == nil || !result.Success {
			continue
		}

		successfulScanners = append(successfulScanners, scannerType)
		totalIssues += result.TotalIssues
	}

	primaryScanner, ok := domainjobs.SelectPrimaryScanner(job, successfulScanners)
	if !ok {
		return "", "", "", totalIssues
	}

	result := job.ScannerResults[primaryScanner]
	if result == nil {
		return "", "", "", totalIssues
	}

	return result.ReportPath, result.StageLogPath, result.RecipePath, totalIssues
}

func buildScannerArtifacts(job *models.Job) map[string]events.ScannerArtifacts {
	if job == nil {
		return nil
	}

	scannerArtifacts := make(map[string]events.ScannerArtifacts)

	for scannerType, result := range job.ScannerResults {
		if result == nil || !result.Success {
			continue
		}

		scannerArtifacts[scannerType] = events.ScannerArtifacts{
			ScannerType:  scannerType,
			ResultsPath:  result.ResultsPath,
			ReportPath:   result.ReportPath,
			StageLogPath: result.StageLogPath,
			RecipePath:   result.RecipePath,
		}
	}

	return scannerArtifacts
}
