package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/models"
	domainjobs "github.com/mattboback/stageflow/platform/orchestrator/internal/domain/jobs"
)

func (s *Service) CreateJob(ctx context.Context, payload *events.JobCreatedPayload) error {
	now := time.Now()
	job := &models.Job{
		ID:        payload.JobID,
		State:     models.JobStatePending,
		InputType: payload.InputType,
		InputPath: payload.InputPath,
		URLs:      payload.URLs,
		Config: models.JobConfig{
			Modules:             payload.Config.Modules,
			ScannerConfigs:      payload.Config.ScannerConfigs,
			Screenshot:          payload.Config.Screenshot,
			HighlightStyle:      payload.Config.HighlightStyle,
			AllowPrivateTargets: payload.Config.AllowPrivateTargets,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	created, err := s.store.CreateJobIfAbsent(ctx, job)
	if err != nil {
		return fmt.Errorf("failed to create job in database: %w", err)
	}

	if !created {
		existing, getErr := s.store.GetJob(ctx, job.ID)
		if getErr != nil {
			return fmt.Errorf("failed to load duplicate job: %w", getErr)
		}

		action, actionErr := domainjobs.DecideDuplicateJobCreated(existing.State)
		if actionErr != nil {
			return actionErr
		}

		switch action {
		case domainjobs.DuplicateJobCreatedIgnore:
			return nil
		case domainjobs.DuplicateJobCreatedRetryOrchestration:
			job = existing
		default:
			return fmt.Errorf("unsupported duplicate action: %s", action)
		}
	}

	if strings.EqualFold(payload.InputType, "urls") {
		return s.RunURLJob(ctx, job)
	}

	return s.startExtraction(ctx, job)
}

func (s *Service) RunURLJob(ctx context.Context, job *models.Job) error {
	action, err := domainjobs.DecideURLJobPreparation(job.State)
	if err != nil {
		msg := fmt.Sprintf("job %s not ready for READY transition from %s", job.ID, job.State)
		s.failJobSafe(
			ctx,
			job.ID,
			"setup",
			msg,
			describeTransition(job.State, models.JobStateReady),
		)

		return errors.New(msg)
	}

	if action == domainjobs.URLJobPreparationIgnore {
		return nil
	}

	if prepErr := s.prepareURLJob(ctx, job); prepErr != nil {
		return prepErr
	}

	return s.StartScanning(ctx, job)
}

func (s *Service) prepareURLJob(ctx context.Context, job *models.Job) error {
	if err := domainjobs.ValidateURLTargets(job.URLs, s.runtime.AllowsLoopbackTargets()); err != nil {
		s.failJobSafe(
			ctx,
			job.ID,
			"setup",
			err.Error(),
			fmt.Sprintf("pod_netns_mode=%s", s.runtime.PodNetnsMode()),
		)

		return err
	}

	action, err := domainjobs.DecideURLJobPreparation(job.State)
	if err != nil {
		return err
	}

	switch action {
	case domainjobs.URLJobPreparationIgnore:
		return nil
	case domainjobs.URLJobPreparationAlreadyReady, domainjobs.URLJobPreparationAdvanceToReady:
		// Do nothing, proceed
	default:
		return fmt.Errorf("unsupported URL job preparation action: %s", action)
	}

	if podErr := s.ensureJobPod(ctx, job); podErr != nil {
		return podErr
	}

	switch action {
	case domainjobs.URLJobPreparationAlreadyReady:
	case domainjobs.URLJobPreparationAdvanceToReady:
		if updateErr := s.store.UpdateJobState(ctx, job.ID, models.JobStateReady); updateErr != nil {
			return fmt.Errorf("failed to update job state to ready: %w", updateErr)
		}

		job.State = models.JobStateReady
	case domainjobs.URLJobPreparationIgnore:
		return nil
	default:
		return fmt.Errorf("unsupported URL job preparation action: %s", action)
	}

	provenanceKey := job.ID + "/provenance.json"
	if updateErr := s.store.UpdateJobProvenanceKey(ctx, job.ID, provenanceKey); updateErr != nil {
		slog.Warn("Failed to persist provenance key for URL job", "job_id", job.ID, "error", updateErr)
	} else {
		job.ProvenanceKey = provenanceKey
	}

	return nil
}

func (s *Service) startExtraction(ctx context.Context, job *models.Job) error {
	action, err := domainjobs.DecideExtractionStart(job.State)
	if err != nil {
		msg := fmt.Sprintf("job %s cannot transition to EXTRACTING from %s", job.ID, job.State)
		s.failJobSafe(
			ctx,
			job.ID,
			"extraction",
			msg,
			describeTransition(job.State, models.JobStateExtracting),
		)

		return errors.New(msg)
	}

	switch action {
	case domainjobs.ExtractionStartAlreadyExtracting:
	case domainjobs.ExtractionStartAdvance:
	default:
		return fmt.Errorf("unsupported extraction start action: %s", action)
	}

	if podErr := s.ensureJobPod(ctx, job); podErr != nil {
		return podErr
	}

	switch action {
	case domainjobs.ExtractionStartAlreadyExtracting:
	case domainjobs.ExtractionStartAdvance:
		if updateErr := s.store.UpdateJobState(ctx, job.ID, models.JobStateExtracting); updateErr != nil {
			return fmt.Errorf("failed to update job state to extracting: %w", updateErr)
		}

		job.State = models.JobStateExtracting

		if recordErr := s.store.RecordExtractionStart(ctx, job.ID); recordErr != nil {
			slog.Warn("Failed to record extraction start", "job_id", job.ID, "error", recordErr)
		}
	}

	if workerErr := s.runtime.StartExtractionWorker(ctx, job); workerErr != nil {
		return fmt.Errorf("failed to start extraction worker: %w", workerErr)
	}

	return nil
}

func (s *Service) ensureJobPod(ctx context.Context, job *models.Job) error {
	if job.PodID != "" {
		return nil
	}

	podID, err := s.runtime.CreateJobPod(ctx, job)
	if err != nil {
		return err
	}

	if updateErr := s.store.UpdateJobPodID(ctx, job.ID, podID); updateErr != nil {
		return fmt.Errorf("failed to update job pod ID: %w", updateErr)
	}

	job.PodID = podID

	return nil
}

func (s *Service) failJobSafe(ctx context.Context, jobID, stage, message, details string) {
	if err := s.FailJob(ctx, jobID, stage, message, details); err != nil {
		slog.Warn("Failed to mark job as failed", "job_id", jobID, "error", err)
	}
}

func describeTransition(from, to models.JobState) string {
	return fmt.Sprintf("current_state=%s target_state=%s", from, to)
}
