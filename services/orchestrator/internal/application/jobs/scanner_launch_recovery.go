package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/mattboback/stageflow/libs/go/models"
)

// ReconcileScanningJobs resumes every incomplete scanner belonging to a job
// that was left in SCANNING by a previous orchestrator process. Scanner
// container names are deterministic, so Runtime.StartScanner adopts an
// already-created container instead of creating duplicate work.
func (s *Service) ReconcileScanningJobs(ctx context.Context) error {
	jobs, listErr := s.store.ListJobsByState(ctx, models.JobStateScanning)
	if listErr != nil {
		return fmt.Errorf("list scanning jobs: %w", listErr)
	}

	var reconcileErrs []error

	for _, job := range jobs {
		if job == nil {
			continue
		}

		if reconcileErr := s.reconcileScanningJob(ctx, job); reconcileErr != nil {
			reconcileErrs = append(reconcileErrs, reconcileErr)
		}
	}

	return errors.Join(reconcileErrs...)
}

func (s *Service) reconcileScanningJob(ctx context.Context, job *models.Job) error {
	scannerTypes := append([]string(nil), job.ExpectedScanners...)
	if len(scannerTypes) == 0 {
		scannerTypes = s.runtime.ResolveScannerTypes(job.Config.Modules)
	}

	if len(scannerTypes) == 0 {
		err := fmt.Errorf("no scanners resolved for recovering job %s", job.ID)
		s.failJobSafe(ctx, job.ID, "scanning", err.Error(), "startup scanner-launch recovery")

		return err
	}

	if job.PodID == "" {
		err := fmt.Errorf("recovering job %s has no persisted pod ID", job.ID)
		s.failJobSafe(ctx, job.ID, "scanning", err.Error(), "startup scanner-launch recovery")

		return err
	}

	if err := s.store.RecordScanStart(ctx, job.ID); err != nil {
		return fmt.Errorf("restore scan start timestamp for job %s: %w", job.ID, err)
	}

	if err := s.store.PrepareScannerLaunches(ctx, job.ID, scannerTypes); err != nil {
		return fmt.Errorf("prepare scanner launch recovery for job %s: %w", job.ID, err)
	}

	completed := make(map[string]struct{}, len(job.CompletedScanners))
	for _, scannerType := range job.CompletedScanners {
		completed[scannerType] = struct{}{}
	}

	for _, scannerType := range scannerTypes {
		if _, ok := completed[scannerType]; ok {
			continue
		}

		claimed, claimErr := s.store.ClaimScannerLaunchRecovery(ctx, job.ID, scannerType)
		if claimErr != nil {
			return fmt.Errorf("claim scanner recovery for job %s scanner %s: %w", job.ID, scannerType, claimErr)
		}

		if !claimed {
			// The job may have reached a terminal state while startup recovery
			// was enumerating it. In that case there is nothing left to resume.
			continue
		}

		slog.Info("Recovering scanner launch", "job_id", job.ID, "scanner", scannerType)

		if launchErr := s.launchClaimedScanner(ctx, job, scannerType); launchErr != nil {
			return fmt.Errorf("recover job %s: %w", job.ID, launchErr)
		}
	}

	return nil
}

func (s *Service) launchClaimedScanner(ctx context.Context, job *models.Job, scannerType string) error {
	plan, planErr := s.planner.Plan(ctx, job, scannerType)
	if planErr != nil {
		return s.failScannerLaunch(ctx, job.ID, scannerType, fmt.Errorf("plan scanner launch: %w", planErr))
	}

	containerID, startErr := s.runtime.StartScanner(ctx, job, plan)
	if startErr != nil {
		return s.failScannerLaunch(ctx, job.ID, scannerType, fmt.Errorf("start scanner: %w", startErr))
	}

	if containerID == "" {
		return s.failScannerLaunch(
			ctx,
			job.ID,
			scannerType,
			errors.New("scanner runtime returned an empty container ID"),
		)
	}

	markErr := s.store.MarkScannerLaunched(ctx, job.ID, scannerType, containerID)
	if markErr != nil {
		return s.failScannerLaunch(
			ctx,
			job.ID,
			scannerType,
			fmt.Errorf("persist launched container %s: %w", containerID, markErr),
		)
	}

	return nil
}

func (s *Service) failScannerLaunch(ctx context.Context, jobID, scannerType string, launchErr error) error {
	if markErr := s.store.MarkScannerLaunchFailed(ctx, jobID, scannerType, launchErr.Error()); markErr != nil {
		slog.Warn(
			"Failed to persist scanner launch failure",
			"job_id", jobID,
			"scanner", scannerType,
			"error", markErr,
		)
	}

	message := fmt.Sprintf("failed to start scanner %s", scannerType)
	s.failJobSafe(
		ctx,
		jobID,
		"scanning",
		message,
		fmt.Sprintf("scanner=%s error=%v", scannerType, launchErr),
	)

	return fmt.Errorf("start scanner %s: %w", scannerType, launchErr)
}
