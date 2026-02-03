package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/mattboback/stageflow/packages/shared-go/logging"
	"github.com/mattboback/stageflow/packages/shared-go/models"
	"github.com/mattboback/stageflow/packages/shared-go/storage"
	"github.com/mattboback/stageflow/platform/orchestrator/internal/podman"
)

// startScanning starts the scanner containers in the pod (one per scanner type).
// Multiple scanners can run in parallel within the same pod.
func (o *Orchestrator) startScanning(ctx context.Context, job *models.Job) error {
	scannerTypes := o.getScannerTypes(job.Config.Modules)
	if len(scannerTypes) == 0 {
		return fmt.Errorf("no scanners resolved for job %s", job.ID)
	}

	if job.State != models.JobStateScanning {
		if !o.stateMachine.CanTransition(job.State, models.JobStateScanning) {
			msg := fmt.Sprintf("job %s cannot transition to SCANNING from %s", job.ID, job.State)
			slog.Warn(msg, "job_id", job.ID, "from_state", job.State)
			o.failJobSafeWithDetails(ctx, job.ID, "scanning", msg, stateTransitionDetails(job.State, models.JobStateScanning))

			return fmt.Errorf("%s", msg)
		}

		if err := o.database.UpdateJobState(ctx, job.ID, models.JobStateScanning); err != nil {
			return fmt.Errorf("failed to update job state: %w", err)
		}

		job.State = models.JobStateScanning

		// Record scan start time for metrics
		if err := o.database.RecordScanStart(ctx, job.ID); err != nil {
			slog.Warn("Failed to record scan start", "job_id", job.ID, "error", err)
		}
	}

	slog.Info("Starting scanners", "scanner_count", len(scannerTypes), "job_id", job.ID, "pod_id", job.PodID, "scanners", scannerTypes)

	// Set expected scanners in database for completion tracking
	if err := o.database.SetExpectedScanners(ctx, job.ID, scannerTypes); err != nil {
		return fmt.Errorf("failed to set expected scanners: %w", err)
	}

	job.ExpectedScanners = scannerTypes
	job.CompletedScanners = []string{}
	job.ScannerResults = make(map[string]*models.ScannerResult)

	if err := o.startScannersWithTimeout(ctx, job, scannerTypes); err != nil {
		return fmt.Errorf("failed to start scanners: %w", err)
	}

	// Watchdog: fail the job if scanning exceeds timeout.
	go o.watchDeadline(backgroundWithCorrelation(ctx), job.ID, models.JobStateScanning, o.scanTimeout, "scanning")

	return nil
}

// getScannerTypes returns the list of scanner types to run based on modules config.
// It uses the scanner registry to resolve module names/aliases to scanner IDs.
func (o *Orchestrator) getScannerTypes(modules []string) []string {
	return o.scannerRegistry.ResolveModules(modules)
}

func (o *Orchestrator) startScannersWithTimeout(ctx context.Context, job *models.Job, scannerTypes []string) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, o.scanTimeout)
	defer cancel()

	errChan := make(chan error, len(scannerTypes))
	for _, scannerType := range scannerTypes {
		go func(st string) {
			errChan <- o.startSingleScanner(timeoutCtx, job, st)
		}(scannerType)
	}

	var firstErr error

	for i := 0; i < len(scannerTypes); i++ {
		select {
		case err := <-errChan:
			if err != nil && firstErr == nil {
				firstErr = err
			}
		case <-timeoutCtx.Done():
			if firstErr == nil {
				firstErr = fmt.Errorf("scanner start timed out after %v", o.scanTimeout)
			}
		}
	}

	return firstErr
}

func (o *Orchestrator) startSingleScanner(ctx context.Context, job *models.Job, scannerType string) error {
	natsURL := "nats://" + o.natsHost + ":4222"
	minioEndpoint := o.minioHost + ":9000"

	resultsDir := "/results/" + scannerType

	provenancePath := "/workspace/provenance.json"
	if job.InputType == inputTypeURLs {
		// URL jobs need a writable provenance path; workspace is mounted read-only.
		provenancePath = resultsDir + "/provenance.json"
	}

	env := map[string]string{
		"JOB_ID":                job.ID,
		"SCANNER_TYPE":          scannerType,
		"NATS_URL":              natsURL,
		"MINIO_ENDPOINT":        minioEndpoint,
		"MINIO_ACCESS_KEY":      o.minioAccessKey,
		"MINIO_SECRET_KEY":      o.minioSecretKey,
		"MINIO_USE_SSL":         strconv.FormatBool(o.minioUseSSL),
		"MINIO_ARTIFACT_BUCKET": storage.BucketArtifacts,
		"PROVENANCE_PATH":       provenancePath,
		"RESULTS_DIR":           resultsDir,
		"PAGE_LOAD_TIMEOUT":     strconv.Itoa(o.pageLoadTimeout),
		"A11Y_SCROLL_TIMEOUT":   strconv.Itoa(o.scrollTimeout),
		"A11Y_SHOT_ENABLED":     strconv.FormatBool(job.Config.Screenshot),
	}

	highlightStyle := job.Config.HighlightStyle
	if highlightStyle == "" {
		highlightStyle = "dashed"
	}

	env["A11Y_HIGHLIGHT_STYLE"] = highlightStyle

	if requestID := logging.RequestID(ctx); requestID != "" {
		env["REQUEST_ID"] = requestID
	}

	if runID := logging.RunID(ctx); runID != "" {
		env["RUN_ID"] = runID
	}

	if scannerType == "ai-navigator" {
		if apiKey := os.Getenv("OPENROUTER_API_KEY"); apiKey != "" {
			env["OPENROUTER_API_KEY"] = apiKey
		}

		if appTitle := os.Getenv("OPENROUTER_APP_TITLE"); appTitle != "" {
			env["OPENROUTER_APP_TITLE"] = appTitle
		}

		if appReferer := os.Getenv("OPENROUTER_APP_REFERER"); appReferer != "" {
			env["OPENROUTER_APP_REFERER"] = appReferer
		}
	}

	// For URL jobs, pass URLs directly so scanner can create provenance file
	if job.InputType == inputTypeURLs && len(job.URLs) > 0 {
		urlsJSON, err := json.Marshal(job.URLs)
		if err != nil {
			return fmt.Errorf("failed to marshal URLs: %w", err)
		}

		env["SCAN_URLS"] = string(urlsJSON)
	}

	if len(job.Config.ScannerConfigs) > 0 {
		if scannerConfig, ok := job.Config.ScannerConfigs[scannerType]; ok {
			if len(scannerConfig) > 0 {
				configJSON, err := json.Marshal(scannerConfig)
				if err != nil {
					return fmt.Errorf("failed to marshal scanner config for %s: %w", scannerType, err)
				}

				env["SCANNER_OPTIONS"] = string(configJSON)
			}
		}
	}

	// Use same workspace volume as extraction worker
	workspaceVolume := "workspace-" + job.ID
	resultsVolume := "results-" + job.ID

	workspaceVolumeInfo, err := o.ensureVolume(ctx, workspaceVolume)
	if err != nil {
		return fmt.Errorf("failed to prepare workspace volume: %w", err)
	}

	resultsVolumeInfo, err := o.ensureVolume(ctx, resultsVolume)
	if err != nil {
		return fmt.Errorf("failed to prepare results volume: %w", err)
	}

	image := o.getScannerImage(scannerType)

	// Get resource limits from scanner definition
	var resourceLimits *podman.ResourceLimits
	if def, ok := o.scannerRegistry.Get(scannerType); ok && def.Requirements.MaxMemoryMB > 0 {
		resourceLimits = &podman.ResourceLimits{
			MemoryLimitMB: int64(def.Requirements.MaxMemoryMB),
			MemorySwapMB:  int64(def.Requirements.MaxMemoryMB), // Same as memory to disable swap
		}
	}

	// Apply default limits for scanners without explicit config
	if resourceLimits == nil {
		resourceLimits = &podman.ResourceLimits{
			MemoryLimitMB: 2048, // 2GB default
			MemorySwapMB:  2048,
		}
	}

	containerReq := &podman.ContainerCreateRequest{
		Name:  fmt.Sprintf("scanner-%s-%s", scannerType, job.ID),
		Image: image,
		Pod:   job.PodID,
		// Rootless Podman volumes are owned by the host UID; run as container root to write /results.
		User: "0",
		Env:  env,
		Mounts: []podman.VolumeMount{
			{
				Type:        "bind",
				Source:      workspaceVolumeInfo.Mountpoint,
				Destination: "/workspace",
				ReadOnly:    true, // Scanner only reads provenance
			},
			{
				Type:        "bind",
				Source:      resultsVolumeInfo.Mountpoint,
				Destination: "/results",
			},
		},
		Labels: map[string]string{
			"managed_by":   "orchestrator",
			"job_id":       job.ID,
			"component":    "scanner",
			"scanner_type": scannerType,
		},
		ResourceLimits: resourceLimits,
	}

	containerResp, err := o.podmanClient.CreateContainer(ctx, containerReq)
	if err != nil {
		return fmt.Errorf("failed to create %s scanner container: %w", scannerType, err)
	}

	slog.Info("Created scanner container", "scanner", scannerType, "container_id", containerResp.ID, "job_id", job.ID)
	o.recordInternalEvent(ctx, job.ID, "orchestrator.container.created", map[string]any{
		"component":    "scanner",
		"scanner_type": scannerType,
		"container_id": containerResp.ID,
	})

	if err := o.podmanClient.StartContainer(ctx, containerResp.ID); err != nil {
		return fmt.Errorf("failed to start %s scanner container: %w", scannerType, err)
	}

	slog.Info("Started scanner container", "scanner", scannerType, "container_id", containerResp.ID, "job_id", job.ID)
	o.recordInternalEvent(ctx, job.ID, "orchestrator.container.started", map[string]any{
		"component":    "scanner",
		"scanner_type": scannerType,
		"container_id": containerResp.ID,
	})

	go o.monitorContainer(backgroundWithCorrelation(ctx), containerResp.ID, job.ID, "scanner-"+scannerType)

	return nil
}

// getScannerImage returns the container image to use for a scanner type.
func (o *Orchestrator) getScannerImage(scannerType string) string {
	if o.scannerRegistry == nil {
		return o.scannerImage
	}

	return o.scannerRegistry.GetImage(scannerType)
}
