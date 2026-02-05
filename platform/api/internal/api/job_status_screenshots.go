package api

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	report "github.com/mattboback/stageflow/packages/contracts/report/generated/go"
	"github.com/mattboback/stageflow/packages/shared-go/logging"
	"github.com/mattboback/stageflow/packages/shared-go/models"
	"github.com/mattboback/stageflow/packages/shared-go/storage"
	"github.com/mattboback/stageflow/platform/api/internal/status"
)

const (
	pageOverviewViolationID = "__page_overview__"
	artifactTypeScreenshot  = "screenshot"
	scannerTypeAxe          = "axe"
)

type screenshotRef struct {
	ScannerType string
	RuleID      string
	PageID      string
}

func collectScreenshotRefs(issues []report.IssueDetail, defaultScanner string) map[string][]screenshotRef {
	refs := make(map[string][]screenshotRef)

	for _, issue := range issues {
		scannerType := issue.Scanner
		if scannerType == "" {
			scannerType = defaultScanner
		}

		if scannerType == "" || issue.PageId == "" || issue.RuleId == "" {
			continue
		}

		for _, occ := range issue.Occurrences {
			for _, artID := range occ.ArtifactIds {
				if artID == "" {
					continue
				}

				refs[artID] = append(refs[artID], screenshotRef{
					ScannerType: scannerType,
					RuleID:      issue.RuleId,
					PageID:      issue.PageId,
				})
			}
		}
	}

	return refs
}

func (s *Server) collectJobScreenshots(ctx context.Context, rec *status.JobRecord) []models.ScreenshotArtifact {
	var allScreenshots []models.ScreenshotArtifact

	if rec.ReportJSONKey != "" {
		shots, err := s.extractReportScreenshots(ctx, rec.JobID, rec.ReportJSONKey)
		if err != nil {
			logging.Warn(ctx, "Failed to extract screenshots from aggregated report", "error", err)

			return nil
		}

		return shots
	}

	if len(rec.ScannerArtifacts) == 0 {
		return nil
	}

	for scannerType, sa := range rec.ScannerArtifacts {
		if sa.ResultsKey == "" {
			continue
		}

		shots, err := s.extractScannerScreenshots(ctx, rec.JobID, scannerType, sa.ResultsKey)
		if err != nil {
			logging.Warn(ctx, "Failed to extract scanner screenshots", "error", err, "scanner_type", scannerType)

			continue
		}

		allScreenshots = append(allScreenshots, shots...)
	}

	return allScreenshots
}

func (s *Server) extractScannerScreenshots(
	ctx context.Context,
	jobID, scannerType, resultsKey string,
) ([]models.ScreenshotArtifact, error) {
	reader, err := s.config.Storage.DownloadFile(ctx, storage.BucketArtifacts, resultsKey)
	if err != nil {
		return nil, fmt.Errorf("failed to download results.json: %w", err)
	}

	defer func() {
		_ = reader.Close()
	}()

	var results report.UnifiedReportV2
	if decodeErr := json.NewDecoder(reader).Decode(&results); decodeErr != nil {
		return nil, fmt.Errorf("failed to parse results.json: %w", decodeErr)
	}

	pageByID := make(map[string]report.PageSummary, len(results.Pages))
	for _, page := range results.Pages {
		pageByID[page.Id] = page
	}

	refsByArtifact := collectScreenshotRefs(results.Issues, scannerType)

	screenshots := s.collectArtifactScreenshots(ctx, jobID, pageByID, refsByArtifact, results.Artifacts)

	for _, page := range results.Pages {
		if overviewShot, ok := s.buildPageOverviewScreenshotV2(ctx, jobID, scannerType, page); ok {
			screenshots = append(screenshots, overviewShot)
		}
	}

	return screenshots, nil
}

func (s *Server) extractReportScreenshots(
	ctx context.Context,
	jobID, reportKey string,
) ([]models.ScreenshotArtifact, error) {
	reader, err := s.config.Storage.DownloadFile(ctx, storage.BucketArtifacts, reportKey)
	if err != nil {
		return nil, fmt.Errorf("failed to download report.json: %w", err)
	}

	defer func() {
		_ = reader.Close()
	}()

	var aggReport report.UnifiedReportV2
	if decodeErr := json.NewDecoder(reader).Decode(&aggReport); decodeErr != nil {
		return nil, fmt.Errorf("failed to parse report.json: %w", decodeErr)
	}

	pageByID := make(map[string]report.PageSummary, len(aggReport.Pages))
	for _, page := range aggReport.Pages {
		pageByID[page.Id] = page
	}

	refsByArtifact := collectScreenshotRefs(aggReport.Issues, "")

	screenshots := s.collectArtifactScreenshots(ctx, jobID, pageByID, refsByArtifact, aggReport.Artifacts)

	for _, page := range aggReport.Pages {
		overviewScanner := resolveOverviewScannerV2(aggReport.Issues, page.Id)
		if overviewShot, ok := s.buildPageOverviewScreenshotV2(ctx, jobID, overviewScanner, page); ok {
			screenshots = append(screenshots, overviewShot)
		}
	}

	return screenshots, nil
}

func (s *Server) collectArtifactScreenshots(
	ctx context.Context,
	jobID string,
	pageByID map[string]report.PageSummary,
	refsByArtifact map[string][]screenshotRef,
	artifacts []report.ReportArtifact,
) []models.ScreenshotArtifact {
	screenshots := make([]models.ScreenshotArtifact, 0)

	for _, artifact := range artifacts {
		if artifact.Type != artifactTypeScreenshot || artifact.Id == "" || artifact.Path == nil ||
			*artifact.Path == "" {
			continue
		}

		for _, ref := range refsByArtifact[artifact.Id] {
			page := pageByID[ref.PageID]
			screenshotKey, ok := jobScopedJoin(jobID, ref.ScannerType, ref.PageID, *artifact.Path)

			if !ok {
				logging.Warn(ctx, "Refusing to presign non-job-scoped screenshot key",
					"job_id", jobID,
					"scanner_type", ref.ScannerType,
					"page_id", ref.PageID,
					"artifact_path", *artifact.Path,
				)

				continue
			}

			screenshotURL, err := s.config.Storage.GetPresignedURL(
				ctx,
				storage.BucketArtifacts,
				screenshotKey,
				15*time.Minute,
			)
			if err != nil {
				logging.Warn(ctx, "Failed to generate presigned URL for screenshot", "error", err, "key", screenshotKey)

				continue
			}

			screenshots = append(screenshots, models.ScreenshotArtifact{
				ScannerType: ref.ScannerType,
				ViolationID: ref.RuleID,
				PageID:      ref.PageID,
				PageURL:     page.Url,
				URL:         screenshotURL,
			})
		}
	}

	return screenshots
}

func resolveOverviewScannerV2(issues []report.IssueDetail, pageID string) string {
	for _, issue := range issues {
		if issue.PageId == pageID && issue.Scanner == scannerTypeAxe {
			return scannerTypeAxe
		}
	}

	for _, issue := range issues {
		if issue.PageId == pageID && issue.Scanner != "" {
			return issue.Scanner
		}
	}

	return scannerTypeAxe
}

func (s *Server) buildPageOverviewScreenshotV2(
	ctx context.Context,
	jobID string,
	scannerType string,
	page report.PageSummary,
) (models.ScreenshotArtifact, bool) {
	if page.PageOverview == nil || page.PageOverview.ScreenshotFilename == "" {
		return models.ScreenshotArtifact{}, false
	}

	overviewKey, ok := jobScopedJoin(jobID, scannerType, page.Id, "screenshots", page.PageOverview.ScreenshotFilename)
	if !ok {
		logging.Warn(ctx, "Refusing to presign non-job-scoped page overview screenshot key",
			"job_id", jobID,
			"scanner_type", scannerType,
			"page_id", page.Id,
			"filename", page.PageOverview.ScreenshotFilename,
		)

		return models.ScreenshotArtifact{}, false
	}

	overviewURL, err := s.config.Storage.GetPresignedURL(ctx, storage.BucketArtifacts, overviewKey, 15*time.Minute)
	if err != nil {
		logging.Warn(
			ctx,
			"Failed to generate presigned URL for page overview screenshot",
			"error",
			err,
			"key",
			overviewKey,
		)

		return models.ScreenshotArtifact{}, false
	}

	return models.ScreenshotArtifact{
		ScannerType: scannerType,
		ViolationID: pageOverviewViolationID,
		PageID:      page.Id,
		PageURL:     page.Url,
		URL:         overviewURL,
	}, true
}
