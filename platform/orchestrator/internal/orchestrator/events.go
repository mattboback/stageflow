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

		if err := o.database.CreateJob(ctx, job); err != nil {
			return fmt.Errorf("failed to create job in database: %w", err)
		}

		o.recordInternalEvent(ctx, job.ID, "orchestrator.job.persisted", map[string]any{
			"input_type": payload.InputType,
		})

		// URL jobs bypass extraction and proceed directly to scanning.
		if payload.InputType == inputTypeURLs {
			if err := o.handleURLJob(ctx, job); err != nil {
				o.failJobSafe(ctx, job.ID, "setup", fmt.Sprintf("failed to setup URL job: %v", err))

				return nil
			}

			return nil
		}

		// ZIP jobs require an extraction phase to populate the workspace volume.
		if err := o.startExtraction(ctx, job); err != nil {
			o.failJobSafe(ctx, job.ID, "extraction", fmt.Sprintf("failed to start extraction: %v", err))

			return nil
		}

		return nil
	})
}

// HandleExtractionReady handles extraction.ready events.
func (o *Orchestrator) HandleExtractionReady(ctx context.Context, payload *events.ExtractionReadyPayload) error {
	return o.withInboundEvent(ctx, events.EventExtractionReady, payload.JobID, payload, func(ctx context.Context) error {
		slog.Info("Handling extraction.ready", "job_id", payload.JobID)

		if payload.StageLogPath != "" || payload.RecipePath != "" {
			slog.Debug("Extraction artifacts", "job_id", payload.JobID, "stage_log", payload.StageLogPath, "recipe", payload.RecipePath)
		}

		// Record extraction completion time for metrics
		if err := o.database.RecordExtractionComplete(ctx, payload.JobID); err != nil {
			slog.Warn("Failed to record extraction complete", "job_id", payload.JobID, "error", err)
		}

		job, err := o.database.GetJob(ctx, payload.JobID)
		if err != nil {
			return fmt.Errorf("failed to get job: %w", err)
		}

		if payload.ProvenancePath != "" {
			if err := o.database.UpdateJobProvenance(ctx, payload.JobID, payload.ProvenancePath); err != nil {
				slog.Warn("Failed to persist provenance path", "job_id", payload.JobID, "error", err)
			} else {
				job.ProvenancePath = payload.ProvenancePath
			}
		}

		if payload.ProvenanceArtifactPath != "" {
			if err := o.database.UpdateJobProvenanceKey(ctx, payload.JobID, payload.ProvenanceArtifactPath); err != nil {
				slog.Warn("Failed to persist provenance key", "job_id", payload.JobID, "error", err)
			} else {
				job.ProvenanceKey = payload.ProvenanceArtifactPath
			}
		}

		switch job.State {
		case models.JobStateReady:
			slog.Debug("Job already in READY state", "job_id", job.ID)
		case models.JobStateScanning, models.JobStateCompleting, models.JobStateDone:
			slog.Debug("Job already past READY, ignoring duplicate extraction.ready", "job_id", job.ID, "state", job.State)

			return nil
		case models.JobStateFailed:
			slog.Debug("Job already failed, ignoring extraction.ready", "job_id", job.ID)

			return nil
		default:
			if !o.stateMachine.CanTransition(job.State, models.JobStateReady) {
				msg := fmt.Sprintf("job %s cannot transition to READY from %s", job.ID, job.State)
				slog.Warn(msg, "job_id", job.ID, "from_state", job.State)
				o.failJobSafeWithDetails(ctx, job.ID, "extraction", msg, stateTransitionDetails(job.State, models.JobStateReady))

				return fmt.Errorf("%s", msg)
			}

			if err := o.database.UpdateJobState(ctx, payload.JobID, models.JobStateReady); err != nil {
				return fmt.Errorf("failed to update job state: %w", err)
			}

			o.recordInternalEvent(ctx, job.ID, "orchestrator.state.transition", map[string]any{
				"from": string(job.State),
				"to":   string(models.JobStateReady),
			})

			job.State = models.JobStateReady
		}

		if err := o.startScanning(ctx, job); err != nil {
			o.failJobSafe(ctx, job.ID, "scanning", fmt.Sprintf("failed to start scanning: %v", err))

			return nil
		}

		return nil
	})
}

// HandleExtractionFailed handles extraction.failed events.
func (o *Orchestrator) HandleExtractionFailed(ctx context.Context, payload *events.ExtractionFailedPayload) error {
	return o.withInboundEvent(ctx, events.EventExtractionFailed, payload.JobID, payload, func(ctx context.Context) error {
		slog.Error("Handling extraction.failed", "job_id", payload.JobID, "error", payload.Error)

		if payload.StageLogPath != "" || payload.RecipePath != "" {
			slog.Debug("Extraction failure artifacts", "job_id", payload.JobID, "stage_log", payload.StageLogPath, "recipe", payload.RecipePath)
		}

		return o.failJob(ctx, payload.JobID, "extraction", payload.Error, payload.ErrorDetails)
	})
}

// HandleScanCompleted handles scan.completed events.
func (o *Orchestrator) HandleScanCompleted(ctx context.Context, payload *events.ScanCompletedPayload) error {
	return o.withInboundEvent(ctx, events.EventScanCompleted, payload.JobID, payload, func(ctx context.Context) error {
		scannerType := payload.ScannerType
		if scannerType == "" {
			return errors.New("scan completed payload missing scanner type")
		}

		slog.Info("Handling scan.completed", "job_id", payload.JobID, "scanner", scannerType)

		if payload.StageLogPath != "" || payload.RecipePath != "" {
			slog.Debug("Scan artifacts", "job_id", payload.JobID, "scanner", scannerType, "stage_log", payload.StageLogPath, "recipe", payload.RecipePath)
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

		job, err := o.database.GetJob(ctx, payload.JobID)
		if err != nil {
			return fmt.Errorf("failed to get job: %w", err)
		}

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
			slog.Warn("Failed to record scanner completion", "job_id", payload.JobID, "scanner", scannerType, "error", err)

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
				o.failJobSafeWithDetails(ctx, job.ID, "completing", msg, stateTransitionDetails(job.State, models.JobStateCompleting))

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

		if err := o.completeJobWithAggregatedResults(ctx, job); err != nil {
			stage, root := completionStage(err)
			o.failJobSafe(ctx, job.ID, stage, fmt.Sprintf("failed to complete job: %v", root))

			return nil
		}

		return nil
	})
}

// HandleScanFailed handles scan.failed events.
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

		allComplete, err := o.database.RecordScannerFailure(ctx, payload.JobID, scannerType, payload.Error)
		if err != nil {
			slog.Warn("Failed to record scanner failure", "job_id", payload.JobID, "scanner", scannerType, "error", err)
		}

		slog.Info("Scanner failed", "scanner", scannerType, "job_id", payload.JobID, "all_complete", allComplete)

		if allComplete {
			job, err = o.database.GetJob(ctx, payload.JobID)
			if err != nil {
				return fmt.Errorf("failed to refresh job: %w", err)
			}

			hasSuccess := false

			for _, result := range job.ScannerResults {
				if result.Success {
					hasSuccess = true

					break
				}
			}

			if hasSuccess {
				slog.Info("Job has partial success, completing with available results", "job_id", job.ID)

				if err := o.completeJobWithAggregatedResults(ctx, job); err != nil {
					stage, root := completionStage(err)
					o.failJobSafe(ctx, job.ID, stage, fmt.Sprintf("failed to complete job: %v", root))
				}

				return nil
			}

			slog.Error("All scanners failed", "job_id", job.ID)

			return o.failJob(ctx, payload.JobID, "scanning", "all scanners failed: "+payload.Error, payload.ErrorDetails)
		}

		slog.Debug("Waiting for other scanners to complete after failure", "job_id", payload.JobID, "failed_scanner", scannerType)

		return nil
	})
}
