package orchestrator

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	storage "github.com/mattboback/stageflow/internal/artifacts"

	"github.com/mattboback/stageflow/libs/go/events"
)

// HandleJobCreated handles job.created events.
func (o *Orchestrator) HandleJobCreated(ctx context.Context, payload *events.JobCreatedPayload) error {
	return o.newService().HandleJobCreated(ctx, payload)
}

// HandleExtractionReady handles extraction.ready events.
func (o *Orchestrator) HandleExtractionReady(ctx context.Context, payload *events.ExtractionReadyPayload) error {
	if payload == nil {
		return errors.New("missing extraction readiness")
	}
	// Older event producers already published public provenance. New disk-backed
	// extractors publish in their scoped directory, which the orchestrator copies.
	if payload.ProvenancePath != "/provenance/provenance.json" {
		return o.newService().HandleExtractionReady(ctx, payload)
	}

	file, err := os.Open(filepath.Join(o.dataDir, "jobs", payload.JobID, ".provenance", "provenance.json"))
	if err != nil {
		return fmt.Errorf("read extracted provenance: %w", err)
	}

	data, err := io.ReadAll(io.LimitReader(file, storage.MaxArtifactBytes+1))
	file.Close()

	if err != nil {
		return err
	}

	if int64(len(data)) > storage.MaxArtifactBytes {
		return errors.New("provenance too large")
	}

	key := payload.JobID + "/provenance.json"
	if uploadErr := o.storage.UploadFile(
		ctx,
		storage.BucketArtifacts,
		key,
		bytes.NewReader(data),
		int64(len(data)),
	); uploadErr != nil {
		return uploadErr
	}

	payload.ProvenancePath = "/job/provenance.json"
	payload.ProvenanceArtifactPath = key

	return o.newService().HandleExtractionReady(ctx, payload)
}

// HandleExtractionFailed handles extraction.failed events.
func (o *Orchestrator) HandleExtractionFailed(ctx context.Context, payload *events.ExtractionFailedPayload) error {
	return o.newService().HandleExtractionFailed(ctx, payload)
}

// HandleScanPageCompleted handles scanner progress updates for job status reads.
func (o *Orchestrator) HandleScanPageCompleted(ctx context.Context, payload *events.ScanPageCompletedPayload) error {
	return o.newService().HandleScanPageCompleted(ctx, payload)
}

// HandleScanCompleted handles scan.completed events.
func (o *Orchestrator) HandleScanCompleted(ctx context.Context, payload *events.ScanCompletedPayload) error {
	return o.newService().HandleScanCompleted(ctx, payload)
}

// HandleScanFailed handles scan.failed events.
func (o *Orchestrator) HandleScanFailed(ctx context.Context, payload *events.ScanFailedPayload) error {
	return o.newService().HandleScanFailed(ctx, payload)
}
