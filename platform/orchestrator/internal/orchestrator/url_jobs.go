package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"strings"

	"github.com/mattboback/stageflow/packages/shared-go/models"
	podman "github.com/mattboback/stageflow/platform/orchestrator/internal/adapters/runtime"
)

func (o *Orchestrator) handleURLJob(ctx context.Context, job *models.Job) error {
	if job == nil {
		return errors.New("job is nil")
	}

	slog.Info("Setting up URL job", "job_id", job.ID, "url_count", len(job.URLs))

	if o.shouldSkipURLJobSetup(job) {
		return nil
	}

	if err := o.validateURLJobTargets(ctx, job); err != nil {
		return err
	}

	if err := o.ensureURLJobPod(ctx, job); err != nil {
		return err
	}

	if err := o.ensureURLJobReady(ctx, job); err != nil {
		return err
	}

	o.persistURLJobProvenanceKey(ctx, job)

	if err := o.startScanning(ctx, job); err != nil {
		return fmt.Errorf("failed to start scanning: %w", err)
	}

	return nil
}

func (o *Orchestrator) shouldSkipURLJobSetup(job *models.Job) bool {
	switch job.State {
	case models.JobStateDone, models.JobStateFailed:
		slog.Debug("Job already in terminal state, ignoring URL setup", "job_id", job.ID, "state", job.State)
		return true
	case models.JobStateScanning, models.JobStateCompleting:
		slog.Debug("Job already in progress, skipping URL setup", "job_id", job.ID, "state", job.State)
		return true
	case models.JobStatePending, models.JobStateExtracting, models.JobStateReady:
		return false
	}

	return false
}

func (o *Orchestrator) validateURLJobTargets(ctx context.Context, job *models.Job) error {
	if o.podNetnsMode == podNetnsModeHost {
		return nil
	}

	if !containsLoopbackTargets(job.URLs) {
		return nil
	}

	msg := "loopback targets require POD_NETNS_MODE=host for job pods (local dev only)"
	slog.Warn(msg, "job_id", job.ID, "pod_netns_mode", o.podNetnsMode)
	o.failJobSafeWithDetails(ctx, job.ID, "setup", msg, fmt.Sprintf("pod_netns_mode=%s", o.podNetnsMode))

	return fmt.Errorf("%s", msg)
}

func (o *Orchestrator) ensureURLJobPod(ctx context.Context, job *models.Job) error {
	if job.PodID != "" {
		slog.Debug("Reusing existing pod for URL job", "pod_id", job.PodID, "job_id", job.ID)
		return nil
	}

	podReq := &podman.PodCreateRequest{
		Name: "job-" + job.ID,
		Labels: map[string]string{
			"managed_by": "orchestrator",
			"job_id":     job.ID,
		},
		Netns:   podman.PodNetns{Nsmode: o.podNetnsMode},
		HostAdd: o.podHostMappings,
	}
	if o.podNetnsMode == podNetnsModeBridge && o.podNetwork != "" {
		podReq.Networks = map[string]podman.PerNetworkOptions{
			o.podNetwork: {},
		}
	}

	podResp, err := o.podmanClient.CreatePod(ctx, podReq)
	if err != nil {
		return fmt.Errorf("failed to create pod: %w", err)
	}

	if podUpdateErr := o.database.UpdateJobPodID(ctx, job.ID, podResp.ID); podUpdateErr != nil {
		return fmt.Errorf("failed to update job pod ID: %w", podUpdateErr)
	}

	job.PodID = podResp.ID
	slog.Info("Created pod for URL job", "pod_id", podResp.ID, "job_id", job.ID)

	return nil
}

func (o *Orchestrator) ensureURLJobReady(ctx context.Context, job *models.Job) error {
	if job.State == models.JobStateReady {
		return nil
	}

	if !o.canTransition(job.State, models.JobStateReady) {
		msg := fmt.Sprintf("job %s not ready for READY transition from %s", job.ID, job.State)
		slog.Warn(msg, "job_id", job.ID, "from_state", job.State)
		o.failJobSafeWithDetails(ctx, job.ID, "setup", msg, stateTransitionDetails(job.State, models.JobStateReady))

		return fmt.Errorf("%s", msg)
	}

	if err := o.database.UpdateJobState(ctx, job.ID, models.JobStateReady); err != nil {
		return fmt.Errorf("failed to update job state: %w", err)
	}

	job.State = models.JobStateReady

	return nil
}

func (o *Orchestrator) persistURLJobProvenanceKey(ctx context.Context, job *models.Job) {
	// URL jobs generate provenance inside scanner containers. Use a stable MinIO key so
	// job.completed can always reference provenance.json for successful URL runs.
	provenanceKey := job.ID + "/provenance.json"
	if err := o.database.UpdateJobProvenanceKey(ctx, job.ID, provenanceKey); err != nil {
		slog.Warn("Failed to persist provenance key for URL job", "job_id", job.ID, "error", err)
		return
	}

	job.ProvenanceKey = provenanceKey
}

func containsLoopbackTargets(urls []string) bool {
	for _, raw := range urls {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}

		u, err := url.Parse(trimmed)
		if err != nil {
			continue
		}

		host := u.Host
		if host == "" {
			continue
		}

		if h, _, splitErr := net.SplitHostPort(host); splitErr == nil {
			host = h
		}

		host = strings.Trim(host, "[]")

		if strings.EqualFold(host, "localhost") {
			return true
		}

		if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
			return true
		}
	}

	return false
}
