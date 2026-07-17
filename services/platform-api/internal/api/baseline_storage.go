package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
	"github.com/mattboback/stageflow/libs/go/logging"
	"github.com/mattboback/stageflow/libs/go/storage"
	"github.com/mattboback/stageflow/services/platform-api/internal/project"
	"github.com/mattboback/stageflow/services/platform-api/internal/status"
)

var (
	ErrBaselineUnavailable     = errors.New("promoted baseline report is unavailable; promote a new completed job")
	errBaselineOperationQueued = errors.New("baseline operation queued for retry")
	errInvalidBaselineReport   = errors.New("baseline report is invalid")
)

// BaselineReconcileSummary describes the persistent-baseline recovery work
// completed at startup.
type BaselineReconcileSummary struct {
	JournalReplayed       int
	LegacyPresent         int
	LegacyCopied          int
	MissingLegacyProjects []string
}

func baselineReportKey(projectID, jobID string) (string, error) {
	if !validJobID(projectID) || !validJobID(jobID) {
		return "", errors.New("invalid project or baseline job ID")
	}

	return "projects/" + projectID + "/baselines/" + jobID + "/report.json", nil
}

func decodeBaselineReport(data []byte, jobID string) (report.UnifiedReportV2, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	var document report.UnifiedReportV2
	if err := decoder.Decode(&document); err != nil {
		return report.UnifiedReportV2{}, fmt.Errorf("%w: %w", errInvalidBaselineReport, err)
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return report.UnifiedReportV2{}, fmt.Errorf("%w: trailing JSON document", errInvalidBaselineReport)
		}

		return report.UnifiedReportV2{}, fmt.Errorf("%w: trailing content: %w", errInvalidBaselineReport, err)
	}

	if document.Meta.JobId != jobID {
		return report.UnifiedReportV2{}, fmt.Errorf(
			"%w: report job ID %q does not match %q",
			errInvalidBaselineReport,
			document.Meta.JobId,
			jobID,
		)
	}

	return document, nil
}

func (s *Server) copyReportToBaseline(
	ctx context.Context,
	projectID string,
	jobID string,
	reportJSONKey string,
) error {
	sourceKey, ok := jobScopedKey(jobID, reportJSONKey)
	if !ok {
		return fmt.Errorf("%w: non-job-scoped report key for %s", errInvalidBaselineReport, jobID)
	}

	destinationKey, err := baselineReportKey(projectID, jobID)
	if err != nil {
		return err
	}

	reader, err := s.config.Storage.DownloadFile(ctx, storage.BucketArtifacts, sourceKey)
	if err != nil {
		return fmt.Errorf("download baseline source for %s: %w", jobID, err)
	}
	defer reader.Close()

	data, err := readLimitedReportJSON(reader, maxReportJSONBytes)
	if err != nil {
		return fmt.Errorf("read baseline source for %s: %w", jobID, err)
	}

	if _, err = decodeBaselineReport(data, jobID); err != nil {
		return fmt.Errorf("validate baseline source for %s: %w", jobID, err)
	}

	if err = s.config.Storage.UploadFile(
		ctx,
		storage.BucketBaselines,
		destinationKey,
		bytes.NewReader(data),
		int64(len(data)),
	); err != nil {
		return fmt.Errorf("store persistent baseline for %s: %w", jobID, err)
	}

	return nil
}

// ensureBaselineCopy lazily preserves baselines selected before the durable
// baseline bucket existed. It is also used by the diff endpoint so an old
// installation remains usable before its next restart.
func (s *Server) ensureBaselineCopy(ctx context.Context, p *project.Project) error {
	key, err := baselineReportKey(p.ID, p.BaselineJobID)
	if err != nil {
		return err
	}

	exists, err := s.config.Storage.FileExists(ctx, storage.BucketBaselines, key)
	if err != nil {
		return fmt.Errorf("check persistent baseline for %s: %w", p.BaselineJobID, err)
	}

	if exists {
		if _, validateErr := s.downloadBaselineReport(ctx, p.ID, p.BaselineJobID); validateErr == nil {
			return nil
		} else if !errors.Is(validateErr, errInvalidBaselineReport) {
			return fmt.Errorf("validate persistent baseline for %s: %w", p.BaselineJobID, validateErr)
		}

		logging.Warn(
			ctx,
			"Persistent baseline is invalid; attempting recovery from the job artifact",
			"project_id", p.ID,
			"job_id", p.BaselineJobID,
		)
	}

	rec, err := s.jobStatus.Current(ctx, p.BaselineJobID)
	if err != nil {
		if errors.Is(err, status.ErrJobNotFound) {
			return fmt.Errorf("%w: legacy baseline job %s no longer exists", ErrBaselineUnavailable, p.BaselineJobID)
		}

		return fmt.Errorf("load legacy baseline job %s: %w", p.BaselineJobID, err)
	}

	if rec.ReportJSONKey == "" {
		return fmt.Errorf("%w: legacy baseline job %s has no report", ErrBaselineUnavailable, p.BaselineJobID)
	}

	sourceExists, err := s.baselineSourceExists(ctx, p.BaselineJobID, rec.ReportJSONKey)
	if err != nil {
		return fmt.Errorf("check legacy baseline source: %w", err)
	}

	if !sourceExists {
		return fmt.Errorf(
			"%w: legacy baseline job %s artifact has expired",
			ErrBaselineUnavailable,
			p.BaselineJobID,
		)
	}

	if err = s.backfillBaseline(ctx, p.ID, p.BaselineJobID, rec.ReportJSONKey); err != nil {
		return fmt.Errorf("preserve legacy baseline job %s: %w", p.BaselineJobID, err)
	}

	return nil
}

func (s *Server) backfillBaseline(ctx context.Context, projectID, jobID, sourceKey string) error {
	s.baselineMu.Lock()
	defer s.baselineMu.Unlock()

	op, err := s.projectStore.QueueBaselineBackfill(ctx, projectID, jobID, sourceKey)
	if err != nil {
		return fmt.Errorf("queue legacy baseline preservation: %w", err)
	}

	if err = s.runBaselineOperationLocked(ctx, *op); err != nil {
		return fmt.Errorf("%w: %w", errBaselineOperationQueued, err)
	}

	return nil
}

func (s *Server) promoteBaseline(
	ctx context.Context,
	projectID, previousJobID, jobID, sourceKey string,
) error {
	s.baselineMu.Lock()
	defer s.baselineMu.Unlock()

	op, err := s.projectStore.QueueBaselinePromotion(ctx, projectID, previousJobID, jobID, sourceKey)
	if err != nil {
		return err
	}

	if err = s.runBaselineOperationLocked(ctx, *op); err != nil {
		if errors.Is(err, errInvalidBaselineReport) {
			return err
		}

		return fmt.Errorf("%w: %w", errBaselineOperationQueued, err)
	}

	return nil
}

func (s *Server) deleteProjectBaseline(ctx context.Context, projectID string) error {
	s.baselineMu.Lock()
	defer s.baselineMu.Unlock()

	op, err := s.projectStore.QueueProjectDeletion(ctx, projectID)
	if err != nil {
		return err
	}

	if err = s.runBaselineOperationLocked(ctx, *op); err != nil {
		return fmt.Errorf("%w: %w", errBaselineOperationQueued, err)
	}

	return nil
}

func (s *Server) downloadBaselineReport(
	ctx context.Context,
	projectID string,
	jobID string,
) (report.UnifiedReportV2, error) {
	key, err := baselineReportKey(projectID, jobID)
	if err != nil {
		return report.UnifiedReportV2{}, err
	}

	reader, err := s.config.Storage.DownloadFile(ctx, storage.BucketBaselines, key)
	if err != nil {
		return report.UnifiedReportV2{}, fmt.Errorf("download persistent baseline for %s: %w", jobID, err)
	}
	defer reader.Close()

	data, err := readLimitedReportJSON(reader, maxReportJSONBytes)
	if err != nil {
		return report.UnifiedReportV2{}, fmt.Errorf("read persistent baseline for %s: %w", jobID, err)
	}

	document, err := decodeBaselineReport(data, jobID)
	if err != nil {
		return report.UnifiedReportV2{}, fmt.Errorf("validate persistent baseline for %s: %w", jobID, err)
	}

	return document, nil
}

func (s *Server) runBaselineOperationLocked(ctx context.Context, op project.BaselineOperation) error {
	if op.State == project.BaselineOperationObjectPending {
		if err := s.completeBaselineObjectPhase(ctx, op); err != nil {
			return err
		}

		op.State = project.BaselineOperationCommitPending
	}

	return s.commitBaselineOperation(ctx, op)
}

func (s *Server) completeBaselineObjectPhase(ctx context.Context, op project.BaselineOperation) error {
	if err := s.applyBaselineObjectMutation(ctx, op); err != nil {
		return s.handleBaselineObjectFailure(ctx, op, err)
	}

	if err := s.projectStore.MarkBaselineOperationObjectReady(ctx, op.ID); err != nil {
		return fmt.Errorf("mark baseline object mutation complete: %w", err)
	}

	return nil
}

func (s *Server) handleBaselineObjectFailure(
	ctx context.Context,
	op project.BaselineOperation,
	mutationErr error,
) error {
	if errors.Is(mutationErr, errInvalidBaselineReport) {
		if discardErr := s.projectStore.CompleteBaselineOperation(ctx, op); discardErr != nil {
			return errors.Join(mutationErr, fmt.Errorf("discard invalid baseline operation: %w", discardErr))
		}

		return mutationErr
	}

	if op.Kind == project.BaselineOperationBackfill {
		discarded, err := s.discardExpiredBackfill(ctx, op)
		if err != nil {
			return err
		}

		if discarded {
			return fmt.Errorf(
				"%w: legacy baseline job %s artifact has expired",
				ErrBaselineUnavailable,
				op.JobID,
			)
		}
	}

	if err := s.projectStore.RecordBaselineOperationFailure(ctx, op.ID, mutationErr.Error()); err != nil {
		return errors.Join(
			fmt.Errorf("baseline object mutation: %w", mutationErr),
			fmt.Errorf("record baseline operation failure: %w", err),
		)
	}

	return mutationErr
}

func (s *Server) discardExpiredBackfill(
	ctx context.Context,
	op project.BaselineOperation,
) (bool, error) {
	exists, err := s.baselineSourceExists(ctx, op.JobID, op.SourceKey)
	if err != nil {
		return false, fmt.Errorf("recheck legacy baseline source: %w", err)
	}

	if exists {
		return false, nil
	}

	if err = s.projectStore.CompleteBaselineOperation(ctx, op); err != nil {
		return false, fmt.Errorf("discard expired legacy backfill: %w", err)
	}

	return true, nil
}

func (s *Server) commitBaselineOperation(ctx context.Context, op project.BaselineOperation) error {
	var err error
	if op.Kind == project.BaselineOperationPromote {
		err = s.projectStore.CompleteBaselinePromotion(ctx, op)
		if errors.Is(err, project.ErrBaselineSuperseded) {
			// CompleteBaselinePromotion already removed the stale journal entry
			// and queued deletion of its copied object. Propagate the conflict to
			// synchronous callers; startup/background replay handles it as a
			// successfully discarded operation.
			return err
		}
	} else {
		err = s.projectStore.CompleteBaselineOperation(ctx, op)
	}

	if err == nil {
		return nil
	}

	if recordErr := s.projectStore.RecordBaselineOperationFailure(ctx, op.ID, err.Error()); recordErr != nil {
		return errors.Join(
			fmt.Errorf("commit baseline operation: %w", err),
			fmt.Errorf("record baseline operation failure: %w", recordErr),
		)
	}

	return fmt.Errorf("commit baseline operation: %w", err)
}

func (s *Server) baselineSourceExists(ctx context.Context, jobID, rawKey string) (bool, error) {
	key, ok := jobScopedKey(jobID, rawKey)
	if !ok {
		return false, nil
	}

	return s.config.Storage.FileExists(ctx, storage.BucketArtifacts, key)
}

const baselineReconcileInterval = 2 * time.Second

// StartBaselineReconciler retries durable object/database operations while the
// service is running. Startup reconciliation handles crash recovery first.
func (s *Server) StartBaselineReconciler(ctx context.Context) {
	s.startBaselineReconciler(ctx, baselineReconcileInterval)
}

func (s *Server) startBaselineReconciler(ctx context.Context, interval time.Duration) {
	s.baselineWG.Add(1)

	go func() {
		defer s.baselineWG.Done()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				var (
					summary BaselineReconcileSummary
					err     error
				)

				if s.legacySweepDue.Load() {
					summary, err = s.ReconcileProjectBaselines(ctx)
				} else {
					summary.JournalReplayed, err = s.reconcileBaselineJournal(ctx)
				}

				if err != nil {
					logging.Error(ctx, "Background baseline reconciliation failed", "error", err)
					continue
				}

				if summary.JournalReplayed > 0 || summary.LegacyCopied > 0 ||
					len(summary.MissingLegacyProjects) > 0 {
					logging.Info(
						ctx,
						"Background baseline reconciliation complete",
						"journal_replayed", summary.JournalReplayed,
						"legacy_copied", summary.LegacyCopied,
						"missing_legacy_count", len(summary.MissingLegacyProjects),
					)
				}
			}
		}
	}()
}

// WaitForBaselineReconciler joins background baseline maintenance after its
// context is canceled so storage and SQLite dependencies can close safely.
func (s *Server) WaitForBaselineReconciler() {
	if s == nil {
		return
	}

	s.baselineWG.Wait()
}

func (s *Server) applyBaselineObjectMutation(ctx context.Context, op project.BaselineOperation) error {
	switch op.Kind {
	case project.BaselineOperationPromote, project.BaselineOperationBackfill:
		key, err := baselineReportKey(op.ProjectID, op.JobID)
		if err != nil {
			return err
		}

		exists, err := s.config.Storage.FileExists(ctx, storage.BucketBaselines, key)
		if err != nil {
			return fmt.Errorf("check baseline destination: %w", err)
		}

		if exists {
			if _, validateErr := s.downloadBaselineReport(ctx, op.ProjectID, op.JobID); validateErr == nil {
				return nil
			} else if !errors.Is(validateErr, errInvalidBaselineReport) {
				return fmt.Errorf("validate baseline destination: %w", validateErr)
			}

			logging.Warn(
				ctx,
				"Replacing invalid persistent baseline from its source artifact",
				"project_id", op.ProjectID,
				"job_id", op.JobID,
			)
		}

		return s.copyReportToBaseline(ctx, op.ProjectID, op.JobID, op.SourceKey)

	case project.BaselineOperationDeleteObject, project.BaselineOperationDeleteProject:
		if op.JobID == "" {
			return nil
		}

		key, err := baselineReportKey(op.ProjectID, op.JobID)
		if err != nil {
			return err
		}

		if err = s.config.Storage.DeleteFile(ctx, storage.BucketBaselines, key); err != nil {
			return fmt.Errorf("delete persistent baseline %s: %w", key, err)
		}

		return nil

	default:
		return fmt.Errorf("unknown baseline operation kind %q", op.Kind)
	}
}

func (s *Server) reconcileBaselineJournal(ctx context.Context) (int, error) {
	s.baselineMu.Lock()
	defer s.baselineMu.Unlock()

	replayed := 0

	for {
		operations, err := s.projectStore.ListBaselineOperations(ctx)
		if err != nil {
			return replayed, fmt.Errorf("list pending baseline operations: %w", err)
		}

		if len(operations) == 0 {
			return replayed, nil
		}

		succeeded, failures := s.replayBaselineOperations(ctx, operations)
		replayed += succeeded

		if len(failures) > 0 {
			return replayed, errors.Join(failures...)
		}
	}
}

func (s *Server) replayBaselineOperations(
	ctx context.Context,
	operations []project.BaselineOperation,
) (int, []error) {
	var (
		succeeded int
		failures  []error
	)

	for i := range operations {
		if err := s.replayBaselineOperation(ctx, operations[i]); err != nil {
			failures = append(failures, err)
			continue
		}

		succeeded++
	}

	return succeeded, failures
}

func (s *Server) replayBaselineOperation(ctx context.Context, op project.BaselineOperation) error {
	err := s.runBaselineOperationLocked(ctx, op)
	if err == nil {
		return nil
	}

	if errors.Is(err, project.ErrBaselineSuperseded) {
		logging.Warn(
			ctx,
			"Discarded superseded baseline promotion",
			"project_id", op.ProjectID,
			"job_id", op.JobID,
		)

		return nil
	}

	if errors.Is(err, ErrBaselineUnavailable) || errors.Is(err, errInvalidBaselineReport) {
		logging.Error(
			ctx,
			"Discarded unavailable or invalid legacy baseline backfill",
			"project_id", op.ProjectID,
			"job_id", op.JobID,
			"error", err,
		)

		return nil
	}

	return fmt.Errorf("operation %d (%s): %w", op.ID, op.Kind, err)
}

// ReconcileProjectBaselines drains the journal, then preserves baselines made
// before the lifecycle-exempt bucket existed. Already-expired sources are
// reported but do not prevent the operator from re-promoting another job.
func (s *Server) ReconcileProjectBaselines(ctx context.Context) (BaselineReconcileSummary, error) {
	var (
		summary  BaselineReconcileSummary
		failures []error
	)

	replayed, err := s.reconcileBaselineJournal(ctx)
	summary.JournalReplayed = replayed

	if err != nil {
		failures = append(failures, err)
	}

	projects, err := s.projectStore.ListProjects(ctx)
	if err != nil {
		s.legacySweepDue.Store(true)

		failures = append(failures, fmt.Errorf("list projects for baseline backfill: %w", err))

		return summary, errors.Join(failures...)
	}

	legacySweepFailed := false

	for i := range projects {
		if err = s.reconcileProjectBaseline(ctx, &projects[i], &summary); err != nil {
			legacySweepFailed = true

			failures = append(failures, err)
		}
	}

	s.legacySweepDue.Store(legacySweepFailed)

	return summary, errors.Join(failures...)
}

func (s *Server) reconcileProjectBaseline(
	ctx context.Context,
	p *project.Project,
	summary *BaselineReconcileSummary,
) error {
	if p.BaselineJobID == "" {
		return nil
	}

	key, err := baselineReportKey(p.ID, p.BaselineJobID)
	if err != nil {
		return err
	}

	exists, err := s.config.Storage.FileExists(ctx, storage.BucketBaselines, key)
	if err != nil {
		return fmt.Errorf("check project %s baseline: %w", p.Slug, err)
	}

	if exists {
		if _, validateErr := s.downloadBaselineReport(ctx, p.ID, p.BaselineJobID); validateErr == nil {
			summary.LegacyPresent++
			return nil
		} else if !errors.Is(validateErr, errInvalidBaselineReport) {
			return fmt.Errorf("validate project %s baseline: %w", p.Slug, validateErr)
		}
	}

	err = s.ensureBaselineCopy(ctx, p)
	if err == nil {
		summary.LegacyCopied++
		return nil
	}

	if !errors.Is(err, ErrBaselineUnavailable) && !errors.Is(err, errInvalidBaselineReport) {
		return fmt.Errorf("backfill project %s baseline: %w", p.Slug, err)
	}

	summary.MissingLegacyProjects = append(summary.MissingLegacyProjects, p.Slug)
	logging.Error(
		ctx,
		"Legacy project baseline is unavailable; re-promote a completed job",
		"project", p.Slug,
		"job_id", p.BaselineJobID,
		"error", err,
	)

	return nil
}

// BackfillProjectBaselines preserves the legacy helper for callers that do
// not need the reconciliation summary.
func (s *Server) BackfillProjectBaselines(ctx context.Context) error {
	_, err := s.ReconcileProjectBaselines(ctx)
	return err
}
