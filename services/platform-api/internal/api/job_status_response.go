package api

import (
	"context"
	"time"

	"github.com/mattboback/stageflow/libs/go/logging"
	"github.com/mattboback/stageflow/libs/go/models"
	"github.com/mattboback/stageflow/libs/go/storage"
	"github.com/mattboback/stageflow/services/platform-api/internal/status"
)

func (s *Server) buildJobStatusResponse(ctx context.Context, rec *status.JobRecord) (*models.JobStatus, error) {
	job := rec.ToModel()

	if rec.State != models.JobStateDone {
		return job, nil
	}

	artifacts, artifactSet, err := s.buildArtifactsForDoneJob(ctx, rec)
	if err != nil {
		return nil, err
	}

	screenshots := s.collectJobScreenshots(ctx, rec)
	if len(screenshots) > 0 {
		artifacts.Screenshots = screenshots
		artifactSet = true
	}

	if artifactSet {
		job.Artifacts = artifacts
	}

	return job, nil
}

func (s *Server) buildArtifactsForDoneJob(
	ctx context.Context,
	rec *status.JobRecord,
) (*models.ArtifactLocation, bool, error) {
	artifacts := &models.ArtifactLocation{}
	artifactSet := false

	set, err := s.setRequiredArtifactURL(
		ctx,
		rec.JobID,
		&artifacts.ReportJSON,
		rec.ReportJSONKey,
		"Refusing to presign non-job-scoped report JSON key",
	)
	if err != nil {
		return nil, false, err
	}

	artifactSet = set || artifactSet

	set, err = s.setRequiredArtifactURL(
		ctx,
		rec.JobID,
		&artifacts.ReportHTML,
		rec.ReportKey,
		"Refusing to presign non-job-scoped report HTML key",
	)
	if err != nil {
		return nil, false, err
	}

	artifactSet = set || artifactSet

	artifactSet = s.setOptionalArtifactURL(
		ctx,
		rec.JobID,
		&artifacts.ScanStageLog,
		rec.ScanStageLogKey,
		"Failed to generate presigned URL for scan stage log",
		"key",
		rec.ScanStageLogKey,
	) ||
		artifactSet
	artifactSet = s.setOptionalArtifactURL(
		ctx,
		rec.JobID,
		&artifacts.ScanRecipe,
		rec.ScanRecipeKey,
		"Failed to generate presigned URL for scan recipe",
		"key",
		rec.ScanRecipeKey,
	) ||
		artifactSet
	artifactSet = s.setOptionalArtifactURL(
		ctx,
		rec.JobID,
		&artifacts.ExtractionStageLog,
		rec.ExtractionStageLogKey,
		"Failed to generate presigned URL for extraction stage log",
		"key",
		rec.ExtractionStageLogKey,
	) ||
		artifactSet
	artifactSet = s.setOptionalArtifactURL(
		ctx,
		rec.JobID,
		&artifacts.ExtractionRecipe,
		rec.ExtractionRecipeKey,
		"Failed to generate presigned URL for extraction recipe",
		"key",
		rec.ExtractionRecipeKey,
	) ||
		artifactSet
	artifactSet = s.setOptionalArtifactURL(
		ctx,
		rec.JobID,
		&artifacts.ProvenanceJSON,
		rec.ProvenanceKey,
		"Failed to generate presigned URL for provenance",
		"key",
		rec.ProvenanceKey,
	) ||
		artifactSet

	if len(rec.ScannerArtifacts) > 0 {
		perScanner, ok := s.buildPerScannerArtifacts(ctx, rec)
		if ok {
			artifacts.ScannerArtifacts = perScanner
			artifactSet = true
		}
	}

	return artifacts, artifactSet, nil
}

func (s *Server) setOptionalArtifactURL(
	ctx context.Context,
	jobID string,
	dest *string,
	key, warnMsg string,
	args ...any,
) bool {
	if key == "" {
		return false
	}

	scopedKey, ok := jobScopedKey(jobID, key)
	if !ok {
		logging.Warn(ctx, "Refusing to presign non-job-scoped artifact key", "key", key)

		return false
	}

	url, err := s.config.Storage.GetPresignedURL(ctx, storage.BucketArtifacts, scopedKey, 15*time.Minute)
	if err != nil {
		logging.Warn(ctx, warnMsg, append([]any{"error", err}, args...)...)

		return false
	}

	*dest = url

	return true
}

func (s *Server) setRequiredArtifactURL(
	ctx context.Context,
	jobID string,
	dest *string,
	key, refuseMsg string,
) (bool, error) {
	if key == "" {
		return false, nil
	}

	scopedKey, ok := jobScopedKey(jobID, key)
	if !ok {
		logging.Warn(ctx, refuseMsg, "key", key)

		return false, nil
	}

	url, err := s.config.Storage.GetPresignedURL(ctx, storage.BucketArtifacts, scopedKey, 15*time.Minute)
	if err != nil {
		return false, err
	}

	*dest = url

	return true, nil
}

func (s *Server) buildPerScannerArtifacts(
	ctx context.Context,
	rec *status.JobRecord,
) (map[string]*models.ScannerArtifacts, bool) {
	perScanner := make(map[string]*models.ScannerArtifacts, len(rec.ScannerArtifacts))

	for scannerType, sa := range rec.ScannerArtifacts {
		scannerArt := &models.ScannerArtifacts{
			ScannerType: sa.ScannerType,
		}

		_ = s.setOptionalArtifactURL(
			ctx,
			rec.JobID,
			&scannerArt.ResultsJSON,
			sa.ResultsKey,
			"Failed to generate presigned URL for scanner results",
			"scanner_type",
			scannerType,
		)
		_ = s.setOptionalArtifactURL(
			ctx,
			rec.JobID,
			&scannerArt.ReportHTML,
			sa.ReportKey,
			"Failed to generate presigned URL for scanner report",
			"scanner_type",
			scannerType,
		)
		_ = s.setOptionalArtifactURL(
			ctx,
			rec.JobID,
			&scannerArt.ScanStageLog,
			sa.StageLogKey,
			"Failed to generate presigned URL for scanner stage log",
			"scanner_type",
			scannerType,
		)
		_ = s.setOptionalArtifactURL(
			ctx,
			rec.JobID,
			&scannerArt.ScanRecipe,
			sa.RecipeKey,
			"Failed to generate presigned URL for scanner recipe",
			"scanner_type",
			scannerType,
		)

		perScanner[scannerType] = scannerArt
	}

	return perScanner, len(perScanner) > 0
}
