package orchestrator

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mattboback/stageflow/packages/shared-go/models"
	"github.com/mattboback/stageflow/platform/orchestrator/internal/podman"
)

func (o *Orchestrator) handleURLJob(ctx context.Context, job *models.Job) error {
	slog.Info("Setting up URL job", "job_id", job.ID, "url_count", len(job.URLs))

	if job.State == models.JobStateDone || job.State == models.JobStateFailed {
		slog.Debug("Job already in terminal state, ignoring URL setup", "job_id", job.ID, "state", job.State)

		return nil
	}

	if job.State == models.JobStateScanning || job.State == models.JobStateCompleting {
		slog.Debug("Job already in progress, skipping URL setup", "job_id", job.ID, "state", job.State)

		return nil
	}

	if job.PodID == "" {
		podReq := &podman.PodCreateRequest{
			Name: "job-" + job.ID,
			Labels: map[string]string{
				"managed_by": "orchestrator",
				"job_id":     job.ID,
			},
			HostAdd: o.podHostMappings,
		}
		if o.podNetwork != "" {
			podReq.Networks = map[string]podman.PerNetworkOptions{
				o.podNetwork: {},
			}
			podReq.Netns = podman.PodNetns{Nsmode: "bridge"}
		}

		podResp, err := o.podmanClient.CreatePod(ctx, podReq)
		if err != nil {
			return fmt.Errorf("failed to create pod: %w", err)
		}

		if err := o.database.UpdateJobPodID(ctx, job.ID, podResp.ID); err != nil {
			return fmt.Errorf("failed to update job pod ID: %w", err)
		}

		job.PodID = podResp.ID
		slog.Info("Created pod for URL job", "pod_id", podResp.ID, "job_id", job.ID)
	} else {
		slog.Debug("Reusing existing pod for URL job", "pod_id", job.PodID, "job_id", job.ID)
	}

	// Update state to READY (skip EXTRACTING for URL jobs)
	if job.State != models.JobStateReady {
		if !o.stateMachine.CanTransition(job.State, models.JobStateReady) {
			msg := fmt.Sprintf("job %s not ready for READY transition from %s", job.ID, job.State)
			slog.Warn(msg, "job_id", job.ID, "from_state", job.State)
			o.failJobSafeWithDetails(ctx, job.ID, "setup", msg, stateTransitionDetails(job.State, models.JobStateReady))

			return fmt.Errorf("%s", msg)
		}

		if err := o.database.UpdateJobState(ctx, job.ID, models.JobStateReady); err != nil {
			return fmt.Errorf("failed to update job state: %w", err)
		}

		job.State = models.JobStateReady
	}

	// URL jobs generate provenance inside scanner containers. Use a stable MinIO key so
	// job.completed can always reference provenance.json for successful URL runs.
	provenanceKey := job.ID + "/provenance.json"
	if err := o.database.UpdateJobProvenanceKey(ctx, job.ID, provenanceKey); err != nil {
		slog.Warn("Failed to persist provenance key for URL job", "job_id", job.ID, "error", err)
	} else {
		job.ProvenanceKey = provenanceKey
	}

	// Scanners receive SCAN_URLS and generate provenance for URL jobs.
	if err := o.startScanning(ctx, job); err != nil {
		return fmt.Errorf("failed to start scanning: %w", err)
	}

	return nil
}
