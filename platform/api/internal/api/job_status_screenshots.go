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
	artifactTypeScreenshot = "screenshot"
	scannerTypeAxe         = "axe"
	occurrenceIDDelimiter  = "--occ-"
)

func buildDerivedIssueID(baseIssueID string, occurrenceIndex int) string {
	if occurrenceIndex <= 0 {
		return baseIssueID
	}

	return fmt.Sprintf("%s%s%d", baseIssueID, occurrenceIDDelimiter, occurrenceIndex+1)
}

func collectScreenshotArtifactPaths(artifacts []report.ReportArtifact) map[string]string {
	paths := make(map[string]string)

	for _, artifact := range artifacts {
		if artifact.Type != artifactTypeScreenshot || artifact.Id == "" || artifact.Path == nil ||
			*artifact.Path == "" {
			continue
		}

		paths[artifact.Id] = *artifact.Path
	}

	return paths
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

func (s *Server) extractScreenshotsFromReport(
	ctx context.Context,
	jobID string,
	results report.UnifiedReportV2,
	defaultScanner string,
) []models.ScreenshotArtifact {
	pageByID := make(map[string]report.PageSummary, len(results.Pages))
	for _, page := range results.Pages {
		pageByID[page.Id] = page
	}

	artifactPathByID := collectScreenshotArtifactPaths(results.Artifacts)
	screenshots := s.collectIssueScreenshots(
		ctx,
		jobID,
		results.Issues,
		pageByID,
		artifactPathByID,
		defaultScanner,
	)

	for _, page := range results.Pages {
		overviewScanner := defaultScanner
		if overviewScanner == "" {
			overviewScanner = resolveOverviewScannerV2(results.Issues, page.Id)
		}
		if overviewScanner == "" {
			overviewScanner = scannerTypeAxe
		}

		if overviewShot, ok := s.buildPageOverviewScreenshotV2(ctx, jobID, overviewScanner, page); ok {
			screenshots = append(screenshots, overviewShot)
		}
	}

	return screenshots
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

	return s.extractScreenshotsFromReport(ctx, jobID, results, scannerType), nil
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

	return s.extractScreenshotsFromReport(ctx, jobID, aggReport, ""), nil
}

func (s *Server) collectIssueScreenshots(
	ctx context.Context,
	jobID string,
	issues []report.IssueDetail,
	pageByID map[string]report.PageSummary,
	artifactPathByID map[string]string,
	defaultScanner string,
) []models.ScreenshotArtifact {
	screenshots := make([]models.ScreenshotArtifact, 0)
	seen := make(map[string]struct{})

	for _, issue := range issues {
		scannerID := issue.Scanner
		if scannerID == "" {
			scannerID = defaultScanner
		}

		if scannerID == "" || issue.PageId == "" || issue.Id == "" {
			continue
		}

		page := pageByID[issue.PageId]
		pageURL := page.Url
		if pageURL == "" {
			pageURL = issue.PageUrl
		}

		for occurrenceIndex, occurrence := range issue.Occurrences {
			derivedIssueID := buildDerivedIssueID(issue.Id, occurrenceIndex)

			for _, artifactID := range occurrence.ArtifactIds {
				if artifactID == "" {
					continue
				}

				artifactPath, ok := artifactPathByID[artifactID]
				if !ok {
					continue
				}

				dedupKey := fmt.Sprintf("%s|%d|%s", issue.Id, occurrenceIndex, artifactID)
				if _, exists := seen[dedupKey]; exists {
					continue
				}
				seen[dedupKey] = struct{}{}

				screenshotKey, ok := jobScopedJoin(jobID, scannerID, issue.PageId, artifactPath)

				if !ok {
					logging.Warn(ctx, "Refusing to presign non-job-scoped screenshot key",
						"job_id", jobID,
						"scanner_id", scannerID,
						"page_id", issue.PageId,
						"artifact_path", artifactPath,
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
					logging.Warn(
						ctx,
						"Failed to generate presigned URL for screenshot",
						"error",
						err,
						"key",
						screenshotKey,
					)

					continue
				}

				occurrenceIndexCopy := occurrenceIndex
				screenshot, valid := buildViolationScreenshotArtifact(
					derivedIssueID,
					occurrenceIndexCopy,
					artifactID,
					scannerID,
					issue.PageId,
					pageURL,
					screenshotURL,
				)
				if !valid {
					logging.Warn(
						ctx,
						"Dropping malformed violation screenshot artifact",
						"job_id",
						jobID,
						"scanner_id",
						scannerID,
						"page_id",
						issue.PageId,
						"issue_id",
						derivedIssueID,
						"artifact_id",
						artifactID,
					)

					continue
				}

				screenshots = append(screenshots, screenshot)
			}
		}
	}

	return screenshots
}

func buildViolationScreenshotArtifact(
	issueID string,
	occurrenceIndex int,
	artifactID string,
	scannerID string,
	pageID string,
	pageURL string,
	url string,
) (models.ScreenshotArtifact, bool) {
	if issueID == "" || artifactID == "" || scannerID == "" || pageID == "" || url == "" {
		return models.ScreenshotArtifact{}, false
	}

	occurrenceIndexCopy := occurrenceIndex

	return models.ScreenshotArtifact{
		Kind:            models.ScreenshotKindViolation,
		IssueID:         issueID,
		OccurrenceIndex: &occurrenceIndexCopy,
		ArtifactID:      artifactID,
		ScannerID:       scannerID,
		PageID:          pageID,
		PageURL:         pageURL,
		URL:             url,
	}, true
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

func buildPageOverviewArtifactID(scannerType, pageID string) string {
	return fmt.Sprintf("page-overview:%s:%s", scannerType, pageID)
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
		Kind:       models.ScreenshotKindPageOverview,
		ArtifactID: buildPageOverviewArtifactID(scannerType, page.Id),
		ScannerID:  scannerType,
		PageID:     page.Id,
		PageURL:    page.Url,
		URL:        overviewURL,
	}, true
}
