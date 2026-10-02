package orchestrator

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mattboback/stageflow/internal/artifacts"
	"github.com/mattboback/stageflow/internal/jobstore"
	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
	"github.com/mattboback/stageflow/libs/go/models"
	podman "github.com/mattboback/stageflow/services/orchestrator/internal/adapters/runtime"
)

// ReconcileOnce is also boot recovery: every side effect can be retried from
// durable rows and atomic scanner files.
func (o *Orchestrator) ReconcileOnce(ctx context.Context) error {
	o.tickMu.Lock()
	defer o.tickMu.Unlock()

	return o.config.Database.WithReconcileLease(ctx, o.reconcileDurable)
}

//nolint:gocognit,gocyclo // One durable tick orders cleanup, observation, expiry and admission under a database lease.
func (o *Orchestrator) reconcileDurable(ctx context.Context) error {
	jobs, err := o.config.Database.ListJobsByState(ctx, models.AllJobStates...)
	if err != nil {
		return err
	}

	now := time.Now()

	for _, job := range jobs {
		if job.State.IsTerminal() && job.CleanupPending {
			if cleanupJobErr := o.cleanupJob(ctx, job); cleanupJobErr != nil {
				slog.Warn("Terminal cleanup will retry", "job_id", job.ID, "error", cleanupJobErr)
				continue
			}

			if markCleanedErr := o.config.Database.MarkCleaned(ctx, job.ID); markCleanedErr != nil {
				return markCleanedErr
			}
		}
	}

	containers, err := o.config.PodmanClient.ListJobContainers(ctx)
	if err != nil {
		return fmt.Errorf("observe job containers: %w", err)
	}

	var errs []error

	for _, job := range jobs {
		if job.State.IsTerminal() {
			continue
		}

		if expired, stage := o.expired(job, now); expired {
			if failErr := o.fail(ctx, job, stage, stage+" exceeded its absolute deadline"); failErr != nil {
				errs = append(errs, failErr)
			}

			continue
		}

		if job.ClaimedAt == nil {
			continue
		}

		if reconcileJobErr := o.reconcileJob(ctx, job, containers); reconcileJobErr != nil {
			errs = append(errs, fmt.Errorf("job %s: %w", job.ID, reconcileJobErr))
		}
	}
	// The database includes terminal cleanup reservations in its capacity count;
	// timing out a job never releases memory capacity before its pod is removed.
	if claimErr := o.claimNext(ctx, containers); claimErr != nil {
		errs = append(errs, claimErr)
	}

	if o.lastRetention.IsZero() || now.Sub(o.lastRetention) >= 10*time.Minute {
		if retainErr := o.retain(ctx, now); retainErr != nil {
			errs = append(errs, retainErr)
		} else {
			o.lastRetention = now
		}
	}

	return errors.Join(errs...)
}

func (o *Orchestrator) expired(job *models.Job, now time.Time) (bool, string) {
	if job.ClaimedAt == nil {
		return now.Sub(job.CreatedAt) >= o.config.QueueTimeout, "queue"
	}

	switch job.State {
	case models.JobStatePending, models.JobStateReady:
		return now.Sub(*job.ClaimedAt) >= o.config.SetupTimeout, "setup"
	case models.JobStateExtracting:
		start := job.ClaimedAt
		if job.ExtractionStartedAt != nil {
			start = job.ExtractionStartedAt
		}

		return now.Sub(*start) >= o.config.ExtractionTimeout, "extraction"
	case models.JobStateScanning:
		start := job.ClaimedAt
		if job.ScanStartedAt != nil {
			start = job.ScanStartedAt
		}

		return now.Sub(*start) >= o.config.ScanTimeout, "scanning"
	case models.JobStateDone, models.JobStateFailed:
		return false, ""
	case models.JobStateCompleting:
		start := job.ClaimedAt
		if job.ScanStartedAt != nil {
			start = job.ScanStartedAt
		}

		return now.Sub(*start) >= o.config.ScanTimeout+o.config.SetupTimeout, "aggregation"
	}

	return false, ""
}

func (o *Orchestrator) stageContext(ctx context.Context, job *models.Job) (context.Context, context.CancelFunc) {
	if job.ClaimedAt == nil {
		return context.WithCancel(ctx)
	}

	start, timeout := job.ClaimedAt, o.config.SetupTimeout
	switch job.State {
	case models.JobStateExtracting:
		timeout = o.config.ExtractionTimeout

		if job.ExtractionStartedAt != nil {
			start = job.ExtractionStartedAt
		}
	case models.JobStateScanning, models.JobStateCompleting:
		timeout = o.config.ScanTimeout

		if job.ScanStartedAt != nil {
			start = job.ScanStartedAt
		}

		if job.State == models.JobStateCompleting {
			timeout += o.config.SetupTimeout
		}
	case models.JobStatePending, models.JobStateReady, models.JobStateDone, models.JobStateFailed:
	}

	return context.WithDeadline(ctx, start.Add(timeout))
}
func (o *Orchestrator) reconcileJob(ctx context.Context, job *models.Job, containers []podman.ContainerInfo) error {
	bounded, cancel := o.stageContext(ctx, job)
	defer cancel()

	ctx = bounded
	if job.PodID == "" {
		podID, err := o.runtime.CreateJobPod(ctx, job)
		if err != nil {
			return err
		}

		if setPodIDErr := o.config.Database.SetPodID(ctx, job.ID, podID); setPodIDErr != nil {
			return setPodIDErr
		}

		job.PodID = podID
	}

	switch job.State {
	case models.JobStatePending:
		next := models.JobStateReady
		if job.InputType == models.JobInputTypeZip {
			next = models.JobStateExtracting
		}

		return o.config.Database.BeginStage(ctx, job.ID, next)
	case models.JobStateExtracting:
		return o.prepareExtracted(ctx, job, containers)
	case models.JobStateReady:
		return o.config.Database.BeginStage(ctx, job.ID, models.JobStateScanning)
	case models.JobStateScanning:
		return o.reconcileScanning(ctx, job, containers)
	case models.JobStateCompleting:
		return o.aggregate(ctx, job)
	case models.JobStateDone, models.JobStateFailed:
		return nil
	}

	return nil
}

func (o *Orchestrator) reconcileScanning(
	ctx context.Context,
	job *models.Job,
	containers []podman.ContainerInfo,
) error {
	if job.InputType == models.JobInputTypeZip {
		extractor := findContainer(containers, job.ID, "extraction-worker", "")
		if extractor != nil && terminalContainer(extractor) {
			return o.fail(ctx, job, "scanning", "static server stopped after readiness")
		}
	}

	for _, scanner := range job.ExpectedScanners {
		if job.ScannerResults[scanner] != nil {
			continue
		}

		if err := o.reconcileScanner(ctx, job, scanner, containers); err != nil {
			return err
		}
	}

	refreshed, err := o.config.Database.GetJob(ctx, job.ID)
	if err != nil {
		return err
	}

	if len(refreshed.CompletedScanners) != len(refreshed.ExpectedScanners) {
		return nil
	}

	if beginStageErr := o.config.Database.BeginStage(ctx, job.ID, models.JobStateCompleting); beginStageErr != nil {
		return beginStageErr
	}

	return o.aggregate(ctx, refreshed)
}

//nolint:gocyclo // Readiness gates bounded provenance and retries deterministic extractor creation.
func (o *Orchestrator) prepareExtracted(ctx context.Context, job *models.Job, containers []podman.ContainerInfo) error {
	container := findContainer(containers, job.ID, "extraction-worker", "")
	if container != nil && terminalContainer(container) {
		return o.fail(ctx, job, "extraction", "extractor stopped before the audit completed")
	}

	root, err := os.OpenRoot(filepath.Join(o.config.DataDir, "jobs", job.ID))
	if errors.Is(err, os.ErrNotExist) {
		if container == nil {
			_, startErr := o.runtime.StartExtractionWorker(ctx, job, job.PodID)
			return startErr
		}

		return nil
	}

	if err != nil {
		return err
	}

	defer root.Close()

	data, readErr := readBounded(root, ".provenance/provenance.json", artifacts.MaxArtifactBytes)
	if readErr == nil {
		var provenance struct {
			JobID string            `json:"job_id"`
			Pages []json.RawMessage `json:"pages"`
		}
		if decodeErr := json.Unmarshal(
			data,
			&provenance,
		); decodeErr != nil || provenance.JobID != job.ID || len(provenance.Pages) == 0 ||
			len(provenance.Pages) > 100 {
			return o.fail(ctx, job, "extraction", "extractor published invalid page coverage")
		}

		if writeErr := o.config.Files.Write(ctx, job.ID+"/provenance.json", bytes.NewReader(data)); writeErr != nil {
			return writeErr
		}

		if updateScannerProgressErr := o.config.Database.UpdateScannerProgress(
			ctx,
			job.ID,
			job.ExpectedScanners[0],
			0,
			len(provenance.Pages),
		); updateScannerProgressErr != nil {
			return updateScannerProgressErr
		}

		if beginStageErr := o.config.Database.BeginStage(ctx, job.ID, models.JobStateReady); beginStageErr != nil {
			return beginStageErr
		}

		job.State = models.JobStateReady

		return nil
	}

	if !errors.Is(readErr, os.ErrNotExist) {
		return readErr
	}

	if container == nil {
		_, startErr := o.runtime.StartExtractionWorker(ctx, job, job.PodID)
		return startErr
	}

	return nil
}

//nolint:gocognit,gocyclo // Scanner settlement separates running adoption, verified exit and bounded launch attempts.
func (o *Orchestrator) reconcileScanner(
	ctx context.Context,
	job *models.Job,
	name string,
	containers []podman.ContainerInfo,
) error {
	container := findContainer(containers, job.ID, "scanner", name)

	if container == nil && job.ScannerContainerIDs[name] != "" {
		recovered, err := o.recoverScanner(ctx, job, name)
		if err != nil {
			return err
		}

		if recovered == nil {
			return nil
		}

		container = recovered
	}

	if err := o.readProgress(ctx, job, name); err != nil {
		slog.Debug("Scanner progress unavailable", "job_id", job.ID, "scanner", name, "error", err)
	}

	outcome, outcomeErr := o.readOutcome(job.ID, name)
	if container != nil && terminalContainer(container) {
		return o.settleScanner(ctx, job, name, container, outcome, outcomeErr)
	}

	if container != nil {
		// A create response or database write may have been lost. The labeled
		// container proves the launch happened; adopt it without consuming an attempt.
		if job.ScannerContainerIDs[name] != container.ID {
			if err := o.config.Database.SetScannerContainer(ctx, job.ID, name, container.ID); err != nil {
				return err
			}
		}

		if string(container.State) == "created" || string(container.State) == "configured" {
			return o.config.PodmanClient.StartContainer(ctx, container.ID)
		}

		return nil
	}

	if outcomeErr == nil {
		return o.recordFailed(ctx, job, name, "scanner exit status is unavailable")
	}

	if !errors.Is(outcomeErr, os.ErrNotExist) {
		return o.recordFailed(ctx, job, name, "scanner outcome is malformed or exceeds its limit")
	}

	if _, err := o.config.Database.BeginScannerAttempt(ctx, job.ID, name); err != nil {
		if errors.Is(err, jobstore.ErrAttemptsExhausted) {
			return o.recordFailed(ctx, job, name, "scanner could not be launched after two attempts")
		}

		return err
	}

	plan, err := o.planner.Plan(ctx, job, name)
	if err != nil {
		return o.recordFailed(ctx, job, name, "scanner configuration is invalid")
	}

	launched, err := o.runtime.StartScanner(ctx, job, plan)
	if err != nil {
		return err
	}

	return o.config.Database.SetScannerContainer(ctx, job.ID, name, launched.ContainerID)
}

func (o *Orchestrator) recordFailed(ctx context.Context, job *models.Job, name, message string) error {
	return o.config.Database.RecordScannerResult(
		ctx,
		job.ID,
		"failed",
		&models.ScannerResult{ScannerType: name, Error: message},
	)
}

func (o *Orchestrator) aggregate(ctx context.Context, job *models.Job) error {
	successful := false
	for _, result := range job.ScannerResults {
		successful = successful || result.Success
	}

	if !successful {
		return o.fail(ctx, job, "scanning", "all scanners failed")
	}

	key, err := o.aggregator.BuildAggregatedReport(ctx, job)
	if err != nil {
		return err
	}

	reader, err := o.config.Files.Open(ctx, key)
	if err != nil {
		return err
	}
	defer reader.Close()

	var doc report.UnifiedReportV2
	if decodeErr := json.NewDecoder(io.LimitReader(reader, artifacts.MaxArtifactBytes)).Decode(&doc); decodeErr != nil {
		return decodeErr
	}

	return o.config.Database.MarkTerminal(ctx, job.ID, models.JobStateDone, key, doc.Summary.TotalIssues, "", "", "")
}

func findContainer(containers []podman.ContainerInfo, jobID, component, scanner string) *podman.ContainerInfo {
	for index := range containers {
		container := &containers[index]

		labels := container.Labels
		if len(labels) == 0 {
			labels = container.Config.Labels
		}

		if labels["managed_by"] == "orchestrator" && labels["job_id"] == jobID && labels["component"] == component &&
			(scanner == "" || labels["scanner_type"] == scanner) {
			return container
		}
	}

	return nil
}
func terminalContainer(container *podman.ContainerInfo) bool {
	state := strings.ToLower(string(container.State))
	return state == "exited" || state == "stopped" || state == "dead"
}

func (o *Orchestrator) settleScanner(
	ctx context.Context,
	job *models.Job,
	name string,
	container *podman.ContainerInfo,
	outcome *scannerOutcome,
	outcomeErr error,
) error {
	info, err := o.config.PodmanClient.InspectContainer(ctx, container.ID)
	if err != nil {
		if podman.IsNotFound(err) {
			return o.recordFailed(ctx, job, name, "previously launched scanner container disappeared")
		}

		return err
	}

	container = info

	if container.ExitCode != 0 {
		message := fmt.Sprintf("%s exited with code %d", name, container.ExitCode)
		if container.OOMKilled {
			message = name + " ran out of memory"
		}

		return o.recordOutcomeFailure(ctx, job, name, message, outcome)
	}

	if outcomeErr != nil {
		return o.recordFailed(ctx, job, name, "scanner exited without a valid complete outcome")
	}

	if outcome.Status == "failed" {
		return o.recordOutcomeFailure(ctx, job, name, outcome.Error, outcome)
	}

	if resultErr := o.validateResults(job.ID, name, outcome); resultErr != nil {
		return o.recordOutcomeFailure(ctx, job, name, "scanner results are missing, malformed or inconsistent", outcome)
	}

	result := outcome.Result(name, job.ID)

	return o.config.Database.RecordScannerResult(ctx, job.ID, outcome.Status, result)
}

func (o *Orchestrator) recoverScanner(
	ctx context.Context,
	job *models.Job,
	name string,
) (*podman.ContainerInfo, error) {
	recovered, err := o.config.PodmanClient.InspectContainer(ctx, job.ScannerContainerIDs[name])
	if err != nil {
		if podman.IsNotFound(err) {
			return nil, o.recordFailed(ctx, job, name, "previously launched scanner container disappeared")
		}

		return nil, err
	}

	container := findContainer([]podman.ContainerInfo{*recovered}, job.ID, "scanner", name)
	if container == nil {
		return nil, o.recordFailed(ctx, job, name, "scanner container ownership changed")
	}

	return container, nil
}

func (o *Orchestrator) recordOutcomeFailure(
	ctx context.Context,
	job *models.Job,
	name, message string,
	outcome *scannerOutcome,
) error {
	if outcome == nil {
		return o.recordFailed(ctx, job, name, message)
	}

	result := outcome.Result(name, job.ID)
	result.Success = false
	result.ResultsPath = ""
	result.Error = message

	return o.config.Database.RecordScannerResult(ctx, job.ID, "failed", result)
}

func (o *Orchestrator) claimNext(ctx context.Context, containers []podman.ContainerInfo) error {
	if err := o.Health(ctx); err != nil {
		if errors.Is(err, artifacts.ErrLowDisk) {
			return nil
		}

		return err
	}

	claimed, err := o.config.Database.ClaimNext(ctx)
	if err != nil {
		return err
	}

	if claimed == nil {
		return nil
	}

	return o.reconcileJob(ctx, claimed, containers)
}
