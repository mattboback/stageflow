package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mattboback/stageflow/internal/jobstore"
	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
	"github.com/mattboback/stageflow/libs/go/logging"
	"github.com/mattboback/stageflow/libs/go/models"
)

const (
	artifactTypePageOverview = "page-overview"
	scannerTypeAxe           = "axe"
)

func (s *Server) collectJobScreenshots(ctx context.Context, rec *jobstore.JobRecord) []models.ScreenshotArtifact {
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
	screenshots := make([]models.ScreenshotArtifact, 0, len(results.Pages))

	for _, page := range results.Pages {
		overviewScanner := defaultScanner
		overviewArtifactPath := ""

		if overviewScanner == "" {
			var found bool

			overviewScanner, overviewArtifactPath, found = resolveOverviewArtifactV2(
				results.Artifacts,
				results.Scanners,
				page.Id,
			)
			if !found {
				overviewScanner = resolveOverviewScannerV2(results.Issues, page.Id)
			}
		}

		if overviewScanner == "" {
			overviewScanner = scannerTypeAxe
		}

		if overviewShot, ok := s.buildPageOverviewScreenshotV2(
			ctx,
			jobID,
			overviewScanner,
			overviewArtifactPath,
			page,
		); ok {
			screenshots = append(screenshots, overviewShot)
		}
	}

	return screenshots
}

func (s *Server) extractScannerScreenshots(
	ctx context.Context,
	jobID, scannerType, resultsKey string,
) ([]models.ScreenshotArtifact, error) {
	reader, err := s.config.Files.Open(ctx, resultsKey)
	if err != nil {
		return nil, fmt.Errorf("failed to open results.json: %w", err)
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
	reader, err := s.config.Files.Open(ctx, reportKey)
	if err != nil {
		return nil, fmt.Errorf("failed to open report.json: %w", err)
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

func resolveOverviewArtifactV2(
	artifacts []report.ReportArtifact,
	scanners []report.ScannerSummary,
	pageID string,
) (scannerID, artifactPath string, ok bool) {
	if pageID == "" {
		return "", "", false
	}

	pathsByID := make(map[string]string, len(artifacts))

	for _, artifact := range artifacts {
		if artifact.Type != artifactTypePageOverview || artifact.Id == "" || artifact.Path == nil ||
			*artifact.Path == "" {
			continue
		}

		pathsByID[artifact.Id] = *artifact.Path
	}

	candidates := make([]string, 0, len(scanners)+1)
	candidates = append(candidates, scannerTypeAxe)

	for _, scanner := range scanners {
		if scanner.Id == "" || scanner.Id == scannerTypeAxe {
			continue
		}

		candidates = append(candidates, scanner.Id)
	}

	for _, candidate := range candidates {
		artifactID := fmt.Sprintf("page-overview-%s-%s", candidate, pageID)
		if path, exists := pathsByID[artifactID]; exists {
			return candidate, path, true
		}
	}

	return "", "", false
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
	artifactPath string,
	page report.PageSummary,
) (models.ScreenshotArtifact, bool) {
	if page.PageOverview == nil || page.PageOverview.ScreenshotFilename == "" {
		return models.ScreenshotArtifact{}, false
	}

	overviewPath := artifactPath
	if overviewPath == "" {
		overviewPath = "screenshots/" + page.PageOverview.ScreenshotFilename
	}

	// The scanner wrote this path, so it is served only if it is exactly a
	// screenshot inside this scanner's page directory.
	overviewKey := jobID + "/" + scannerType + "/" + page.Id + "/" + overviewPath

	overviewURL, err := publicArtifactURL(jobID, overviewKey)
	if err != nil {
		logging.Warn(ctx, "Leaving a page overview screenshot out of the job status", "key", overviewKey, "error", err)

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
