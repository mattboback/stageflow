package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/mattboback/stageflow/packages/shared-go/logging"
	"github.com/mattboback/stageflow/packages/shared-go/models"
	"github.com/mattboback/stageflow/packages/shared-go/storage"
	podman "github.com/mattboback/stageflow/platform/orchestrator/internal/adapters/runtime"
)

func (o *Orchestrator) createJobPod(ctx context.Context, job *models.Job) (string, error) {
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
		return "", fmt.Errorf("failed to create pod: %w", err)
	}

	slog.Info("Created pod for job", "pod_id", podResp.ID, "job_id", job.ID)
	o.recordInternalEvent(ctx, job.ID, "orchestrator.pod.created", map[string]any{
		"pod_id": podResp.ID,
	})

	return podResp.ID, nil
}

// startExtractionWorkerWithTimeout starts the extraction worker and enforces a hard startup budget.
func (o *Orchestrator) startExtractionWorkerWithTimeout(ctx context.Context, job *models.Job, podID string) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, o.extractionTimeout)
	defer cancel()

	resultChan := make(chan error, 1)

	go func() {
		resultChan <- o.startExtractionWorker(timeoutCtx, job, podID)
	}()

	select {
	case err := <-resultChan:
		return err
	case <-timeoutCtx.Done():
		return fmt.Errorf("extraction worker start timed out after %v", o.extractionTimeout)
	}
}

func (o *Orchestrator) startExtractionWorker(ctx context.Context, job *models.Job, podID string) error {
	natsURL := "nats://" + o.natsHost + ":4222"
	minioEndpoint := o.minioHost + ":9000"

	if o.podNetnsMode == podNetnsModeHost {
		natsURL = hostNetnsNATSURL
		minioEndpoint = hostNetnsMinioEndpoint
	}

	env := map[string]string{
		"JOB_ID":                job.ID,
		"INPUT_PATH":            job.InputPath,
		"NATS_URL":              natsURL,
		"MINIO_ENDPOINT":        minioEndpoint,
		"MINIO_ACCESS_KEY":      o.minioAccessKey,
		"MINIO_SECRET_KEY":      o.minioSecretKey,
		"MINIO_USE_SSL":         strconv.FormatBool(o.minioUseSSL),
		"WORKSPACE":             "/workspace",
		"PORT":                  "8080",
		"MINIO_ARTIFACT_BUCKET": storage.BucketArtifacts,
	}

	if requestID := logging.RequestID(ctx); requestID != "" {
		env["REQUEST_ID"] = requestID
	}

	if runID := logging.RunID(ctx); runID != "" {
		env["RUN_ID"] = runID
	}

	// Workspace volume is shared across extraction and scanners for this job.
	volumeName := "workspace-" + job.ID
	if err := o.podmanClient.CreateVolume(ctx, volumeName); err != nil {
		return fmt.Errorf("failed to create workspace volume: %w", err)
	}

	workspaceVolume, err := o.podmanClient.InspectVolume(ctx, volumeName)
	if err != nil {
		return fmt.Errorf("failed to inspect workspace volume: %w", err)
	}

	containerReq := &podman.ContainerCreateRequest{
		Name:  "extraction-worker-" + job.ID,
		Image: o.extractionImage,
		Pod:   podID,
		Env:   env,
		Mounts: []podman.VolumeMount{
			{
				// Bind to the volume mountpoint so runc receives a valid host path.
				Type:        "bind",
				Source:      workspaceVolume.Mountpoint,
				Destination: "/workspace",
			},
		},
		Labels: map[string]string{
			"managed_by": "orchestrator",
			"job_id":     job.ID,
			"component":  "extraction-worker",
		},
	}

	containerResp, err := o.podmanClient.CreateContainer(ctx, containerReq)
	if err != nil {
		return fmt.Errorf("failed to create extraction worker container: %w", err)
	}

	slog.Info("Created extraction worker container", "container_id", containerResp.ID, "job_id", job.ID)
	o.recordInternalEvent(ctx, job.ID, "orchestrator.container.created", map[string]any{
		"component":    "extraction-worker",
		"container_id": containerResp.ID,
	})

	if startErr := o.podmanClient.StartContainer(ctx, containerResp.ID); startErr != nil {
		return fmt.Errorf("failed to start extraction worker container: %w", startErr)
	}

	slog.Info("Started extraction worker container", "container_id", containerResp.ID, "job_id", job.ID)
	o.recordInternalEvent(ctx, job.ID, "orchestrator.container.started", map[string]any{
		"component":    "extraction-worker",
		"container_id": containerResp.ID,
	})

	o.spawnMonitorContainer(backgroundWithCorrelation(ctx), containerResp.ID, job.ID, "extraction")

	return nil
}
