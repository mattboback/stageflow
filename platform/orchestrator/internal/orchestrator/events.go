package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/mattboback/stageflow/packages/shared-go/events"
	"github.com/mattboback/stageflow/packages/shared-go/models"
)

func (o *Orchestrator) HandleJobCreated(ctx context.Context, payload *events.JobCreatedPayload) error {
	return o.withInboundEvent(ctx, events.EventJobCreated, payload.JobID, payload, func(ctx context.Context) error {
		slog.Info("Handling job.created", "job_id", payload.JobID)

		job := &models.Job{
			ID:        payload.JobID,
			State:     models.JobStatePending,
			InputType: payload.InputType,
			InputPath: payload.InputPath,
			URLs:      payload.URLs,
			Config: models.JobConfig{
				Modules:        payload.Config.Modules,
				ScannerConfigs: payload.Config.ScannerConfigs,
				Screenshot:     payload.Config.Screenshot,
				HighlightStyle: payload.Config.HighlightStyle,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		created, err := o.database.CreateJobIfAbsent(ctx, job)
		if err != nil {
			return fmt.Errorf("failed to create job in database: %w", err)
		}

		if !created {
			existing, getErr := o.database.GetJob(ctx, payload.JobID)
			if getErr != nil {
				return fmt.Errorf("failed to load existing job after duplicate create: %w", getErr)
			}

			switch existing.State {
			case models.JobStatePending:
				slog.Warn("Duplicate job.created for pending job; retrying orchestration", "job_id", payload.JobID)

				job = existing
			case models.JobStateExtracting, models.JobStateReady, models.JobStateScanning, models.JobStateCompleting:
				slog.Debug(
					"Duplicate job.created ignored for in-flight job",
					"job_id",
					payload.JobID,
					"state",
					existing.State,
				)

				return nil
			case models.JobStateDone, models.JobStateFailed:
				slog.Debug(
					"Duplicate job.created ignored for terminal job",
					"job_id",
					payload.JobID,
					"state",
					existing.State,
				)

				return nil
			default:
				return fmt.Errorf("unsupported job state for duplicate job.created: %s", existing.State)
			}
		}

		o.recordInternalEvent(ctx, job.ID, "orchestrator.job.persisted", map[string]any{
			"input_type": payload.InputType,
		})

		// URL jobs bypass extraction and proceed directly to scanning.
		if payload.InputType == inputTypeURLs {
			if urlErr := o.handleURLJob(ctx, job); urlErr != nil {
				o.failJobSafe(ctx, job.ID, "setup", fmt.Sprintf("failed to setup URL job: %v", urlErr))

				return nil
			}

			return nil
		}

		// ZIP jobs require an extraction phase to populate the workspace volume.
		if extractionErr := o.startExtraction(ctx, job); extractionErr != nil {
			o.failJobSafe(
				ctx,
				job.ID,
				"extraction",
				fmt.Sprintf("failed to start extraction: %v", extractionErr),
			)

			return nil
		}

		return nil
	})
}

// HandleExtractionReady handles extraction.ready events.
func (o *Orchestrator) HandleExtractionReady(ctx context.Context, payload *events.ExtractionReadyPayload) error {
	return o.withInboundEvent(
		ctx,
		events.EventExtractionReady,
		payload.JobID,
		payload,
		func(ctx context.Context) error {
			slog.Info("Handling extraction.ready", "job_id", payload.JobID)

			if payload.StageLogPath != "" || payload.RecipePath != "" {
				slog.Debug(
					"Extraction artifacts",
					"job_id",
					payload.JobID,
					"stage_log",
					payload.StageLogPath,
					"recipe",
					payload.RecipePath,
				)
			}

			// Record extraction completion time for metrics
			if extractionErr := o.database.RecordExtractionComplete(ctx, payload.JobID); extractionErr != nil {
				slog.Warn("Failed to record extraction complete", "job_id", payload.JobID, "error", extractionErr)
			}

			job, err := o.database.GetJob(ctx, payload.JobID)
			if err != nil {
				return fmt.Errorf("failed to get job: %w", err)
			}

			o.persistExtractionReadyMetadata(ctx, payload, job)

			shouldScan, ensureErr := o.ensureReadyForScanning(ctx, payload.JobID, job)
			if ensureErr != nil {
				return ensureErr
			}

			if !shouldScan {
				return nil
			}

			if scanStartErr := o.startScanning(ctx, job); scanStartErr != nil {
				o.failJobSafe(ctx, job.ID, "scanning", fmt.Sprintf("failed to start scanning: %v", scanStartErr))

				return nil
			}

			return nil
		},
	)
}

func (o *Orchestrator) persistExtractionReadyMetadata(
	ctx context.Context,
	payload *events.ExtractionReadyPayload,
	job *models.Job,
) {
	if progressErr := o.database.UpdateJobProgress(ctx, payload.JobID, 0, payload.TotalPages); progressErr != nil {
		slog.Warn("Failed to persist total pages from extraction.ready", "job_id", payload.JobID, "error", progressErr)
	} else {
		job.TotalPages = payload.TotalPages
		job.CurrentPage = 0
	}

	if artifactsErr := o.database.UpdateJobExtractionArtifacts(
		ctx,
		payload.JobID,
		payload.StageLogPath,
		payload.RecipePath,
	); artifactsErr != nil {
		slog.Warn("Failed to persist extraction artifacts", "job_id", payload.JobID, "error", artifactsErr)
	} else {
		if payload.StageLogPath != "" {
			job.ExtractionStageLogKey = payload.StageLogPath
		}

		if payload.RecipePath != "" {
			job.ExtractionRecipeKey = payload.RecipePath
		}
	}

	if payload.ProvenancePath != "" {
		if provenanceErr := o.database.UpdateJobProvenance(
			ctx,
			payload.JobID,
			payload.ProvenancePath,
		); provenanceErr != nil {
			slog.Warn("Failed to persist provenance path", "job_id", payload.JobID, "error", provenanceErr)
		} else {
			job.ProvenancePath = payload.ProvenancePath
		}
	}

	if payload.ProvenanceArtifactPath == "" {
		return
	}

	if provenanceKeyErr := o.database.UpdateJobProvenanceKey(
		ctx,
		payload.JobID,
		payload.ProvenanceArtifactPath,
	); provenanceKeyErr != nil {
		slog.Warn("Failed to persist provenance key", "job_id", payload.JobID, "error", provenanceKeyErr)
	} else {
		job.ProvenanceKey = payload.ProvenanceArtifactPath
	}
}

func (o *Orchestrator) ensureReadyForScanning(
	ctx context.Context,
	jobID string,
	job *models.Job,
) (bool, error) {
	//nolint:exhaustive // Early states (Pending, Extracting) handled by default case.
	switch job.State {
	case models.JobStateReady:
		slog.Debug("Job already in READY state", "job_id", job.ID)
		return true, nil
	case models.JobStateScanning, models.JobStateCompleting, models.JobStateDone:
		slog.Debug(
			"Job already past READY, ignoring duplicate extraction.ready",
			"job_id",
			job.ID,
			"state",
			job.State,
		)

		return false, nil
	case models.JobStateFailed:
		slog.Debug("Job already failed, ignoring extraction.ready", "job_id", job.ID)

		return false, nil
	default:
		if !o.stateMachine.CanTransition(job.State, models.JobStateReady) {
			msg := fmt.Sprintf("job %s cannot transition to READY from %s", job.ID, job.State)
			slog.Warn(msg, "job_id", job.ID, "from_state", job.State)
			o.failJobSafeWithDetails(
				ctx,
				job.ID,
				"extraction",
				msg,
				stateTransitionDetails(job.State, models.JobStateReady),
			)

			return false, fmt.Errorf("%s", msg)
		}

		if stateErr := o.database.UpdateJobState(ctx, jobID, models.JobStateReady); stateErr != nil {
			return false, fmt.Errorf("failed to update job state: %w", stateErr)
		}

		o.recordInternalEvent(ctx, job.ID, "orchestrator.state.transition", map[string]any{
			"from": string(job.State),
			"to":   string(models.JobStateReady),
		})

		job.State = models.JobStateReady

		return true, nil
	}
}

// HandleExtractionFailed handles extraction.failed events.
func (o *Orchestrator) HandleExtractionFailed(ctx context.Context, payload *events.ExtractionFailedPayload) error {
	return o.withInboundEvent(
		ctx,
		events.EventExtractionFailed,
		payload.JobID,
		payload,
		func(ctx context.Context) error {
			slog.Error("Handling extraction.failed", "job_id", payload.JobID, "error", payload.Error)

			if payload.StageLogPath != "" || payload.RecipePath != "" {
				slog.Debug(
					"Extraction failure artifacts",
					"job_id",
					payload.JobID,
					"stage_log",
					payload.StageLogPath,
					"recipe",
					payload.RecipePath,
				)
			}

			if artifactsErr := o.database.UpdateJobExtractionArtifacts(
				ctx,
				payload.JobID,
				payload.StageLogPath,
				payload.RecipePath,
			); artifactsErr != nil {
				slog.Warn(
					"Failed to persist extraction failure artifacts",
					"job_id",
					payload.JobID,
					"error",
					artifactsErr,
				)
			}

			return o.failJob(ctx, payload.JobID, "extraction", payload.Error, payload.ErrorDetails)
		},
	)
}

// HandleScanPageCompleted persists scanner progress updates for job status reads.
func (o *Orchestrator) HandleScanPageCompleted(ctx context.Context, payload *events.ScanPageCompletedPayload) error {
	return o.withInboundEvent(
		ctx,
		events.EventScanPageCompleted,
		payload.JobID,
		payload,
		func(ctx context.Context) error {
			job, err := o.database.GetJob(ctx, payload.JobID)
			if err != nil {
				return fmt.Errorf("failed to get job: %w", err)
			}

			if job.State == models.JobStateDone || job.State == models.JobStateFailed {
				slog.Debug("Ignoring scan.page.completed for terminal job", "job_id", payload.JobID, "state", job.State)

				return nil
			}

			if updateErr := o.database.UpdateJobProgress(
				ctx,
				payload.JobID,
				payload.PageIndex,
				payload.TotalPages,
			); updateErr != nil {
				return fmt.Errorf("failed to persist scan progress: %w", updateErr)
			}

			return nil
		},
	)
}

// HandleScanCompleted handles scan.completed events.
//
//nolint:gocognit,gocyclo // Event handler requires multiple state checks and error handling paths
func (o *Orchestrator) HandleScanCompleted(ctx context.Context, payload *events.ScanCompletedPayload) error {
	return o.withInboundEvent(ctx, events.EventScanCompleted, payload.JobID, payload, func(ctx context.Context) error {
		scannerType := payload.ScannerType
		if scannerType == "" {
			return errors.New("scan completed payload missing scanner type")
		}

		slog.Info("Handling scan.completed", "job_id", payload.JobID, "scanner", scannerType)

		if payload.StageLogPath != "" || payload.RecipePath != "" {
			slog.Debug(
				"Scan artifacts",
				"job_id",
				payload.JobID,
				"scanner",
				scannerType,
				"stage_log",
				payload.StageLogPath,
				"recipe",
				payload.RecipePath,
			)
		}

		if payload.Timing != nil {
			slog.Debug("Scan timing",
				"job_id", payload.JobID,
				"scanner", scannerType,
				"total_ms", payload.Timing.TotalMs,
				"page_iteration_ms", payload.Timing.PageIterationMs,
				"write_results_ms", payload.Timing.WriteResultsMs,
				"upload_artifacts_ms", payload.Timing.UploadArtifactsMs,
				"publish_completed_ms", payload.Timing.PublishCompletedMs,
				"finalization_ms", payload.Timing.FinalizationMs,
			)
		}

		if payload.ResultsPath == "" {
			return errors.New("scan completed payload missing results path")
		}

		if progressErr := o.database.UpdateJobProgress(
			ctx,
			payload.JobID,
			payload.TotalPagesScanned,
			payload.TotalPagesScanned,
		); progressErr != nil {
			slog.Warn("Failed to persist scan completion progress", "job_id", payload.JobID, "error", progressErr)
		}

		job, err := o.database.GetJob(ctx, payload.JobID)
		if err != nil {
			return fmt.Errorf("failed to get job: %w", err)
		}

		//nolint:exhaustive // Only terminal states need early-exit; others proceed to recording.
		switch job.State {
		case models.JobStateDone:
			slog.Debug("Job already marked DONE, ignoring scan.completed", "job_id", job.ID, "scanner", scannerType)

			return nil
		case models.JobStateFailed:
			slog.Debug("Job already failed, ignoring scan.completed", "job_id", job.ID, "scanner", scannerType)

			return nil
		}

		// Record this scanner's completion with metrics
		scannerResult := &models.ScannerResult{
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
		}

		allComplete, err := o.database.RecordScannerCompletion(ctx, payload.JobID, scannerResult)
		if err != nil {
			slog.Warn(
				"Failed to record scanner completion",
				"job_id",
				payload.JobID,
				"scanner",
				scannerType,
				"error",
				err,
			)

			return fmt.Errorf("failed to record scanner completion: %w", err)
		}

		slog.Info("Scanner completed", "scanner", scannerType, "job_id", payload.JobID, "all_complete", allComplete)

		if !allComplete {
			slog.Debug("Waiting for remaining scanners", "job_id", payload.JobID)

			return nil
		}

		slog.Info("All scanners complete, proceeding to finalization", "job_id", payload.JobID)

		if job.State != models.JobStateCompleting {
			if !o.stateMachine.CanTransition(job.State, models.JobStateCompleting) {
				msg := fmt.Sprintf("job %s cannot transition to COMPLETING from %s", job.ID, job.State)
				slog.Warn(msg, "job_id", job.ID, "from_state", job.State)
				o.failJobSafeWithDetails(
					ctx,
					job.ID,
					"completing",
					msg,
					stateTransitionDetails(job.State, models.JobStateCompleting),
				)

				return fmt.Errorf("%s", msg)
			}

			updateErr := o.database.UpdateJobState(ctx, payload.JobID, models.JobStateCompleting)
			if updateErr != nil {
				return fmt.Errorf("failed to update job state: %w", updateErr)
			}

			o.recordInternalEvent(ctx, job.ID, "orchestrator.state.transition", map[string]any{
				"from": string(job.State),
				"to":   string(models.JobStateCompleting),
			})

			job.State = models.JobStateCompleting
		}

		// Reload job to include latest per-scanner results for aggregation.
		job, err = o.database.GetJob(ctx, payload.JobID)
		if err != nil {
			return fmt.Errorf("failed to refresh job: %w", err)
		}

		if completeErr := o.completeJobWithAggregatedResults(ctx, job); completeErr != nil {
			stage, root := completionStage(completeErr)
			o.failJobSafe(ctx, job.ID, stage, fmt.Sprintf("failed to complete job: %v", root))

			return nil
		}

		return nil
	})
}

// HandleScanFailed handles scan.failed events.
//
//nolint:gocognit // Event handler requires multiple state checks and error handling paths
func (o *Orchestrator) HandleScanFailed(ctx context.Context, payload *events.ScanFailedPayload) error {
	return o.withInboundEvent(ctx, events.EventScanFailed, payload.JobID, payload, func(ctx context.Context) error {
		scannerType := payload.ScannerType
		if scannerType == "" {
			scannerType = "unknown"
		}

		slog.Error("Handling scan.failed", "job_id", payload.JobID, "scanner", scannerType, "error", payload.Error)

		job, err := o.database.GetJob(ctx, payload.JobID)
		if err != nil {
			return fmt.Errorf("failed to get job: %w", err)
		}

		if job.State == models.JobStateDone || job.State == models.JobStateFailed {
			slog.Debug("Job already in terminal state, ignoring scan.failed", "job_id", job.ID, "state", job.State)

			return nil
		}

		if artifactsErr := o.database.UpdateJobCompletionArtifacts(
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

		allComplete, err := o.database.RecordScannerFailure(ctx, payload.JobID, scannerType, payload.Error)
		if err != nil {
			slog.Warn("Failed to record scanner failure", "job_id", payload.JobID, "scanner", scannerType, "error", err)
		}

		slog.Info("Scanner failed", "scanner", scannerType, "job_id", payload.JobID, "all_complete", allComplete)

		//nolint:nestif // Completion logic requires multiple scanner result checks
		if allComplete {
			refreshedJob, refreshErr := o.database.GetJob(ctx, payload.JobID)
			if refreshErr != nil {
				return fmt.Errorf("failed to refresh job: %w", refreshErr)
			}

			hasSuccess := false

			for _, result := range refreshedJob.ScannerResults {
				if result.Success {
					hasSuccess = true

					break
				}
			}

			if hasSuccess {
				slog.Info("Job has partial success, completing with available results", "job_id", refreshedJob.ID)

				if completeErr := o.completeJobWithAggregatedResults(ctx, refreshedJob); completeErr != nil {
					stage, root := completionStage(completeErr)
					o.failJobSafe(ctx, refreshedJob.ID, stage, fmt.Sprintf("failed to complete job: %v", root))
				}

				return nil
			}

			slog.Error("All scanners failed", "job_id", refreshedJob.ID)

			return o.failJob(
				ctx,
				payload.JobID,
				"scanning",
				"all scanners failed: "+payload.Error,
				payload.ErrorDetails,
			)
		}

		slog.Debug(
			"Waiting for other scanners to complete after failure",
			"job_id",
			payload.JobID,
			"failed_scanner",
			scannerType,
		)

		return nil
	})
}
