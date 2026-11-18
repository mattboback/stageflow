package api

import (
	"context"

	"github.com/mattboback/stageflow/internal/jobstore"
	"github.com/mattboback/stageflow/libs/go/logging"
	"github.com/mattboback/stageflow/libs/go/models"
)

func (s *Server) buildJobStatusResponse(ctx context.Context, rec *jobstore.JobRecord) (*models.JobStatus, error) {
	job := rec.ToModel()

	if rec.State != models.JobStateDone {
		return job, nil
	}

	if rec.ReportJSONKey != "" {
		report, err := s.downloadReport(ctx, rec.JobID, rec.ReportJSONKey)
		if err != nil {
			logging.Warn(ctx, "Failed to normalize completed job issue count from aggregated report", "error", err)
		} else {
			job.TotalViolations = report.Summary.TotalIssues
		}
	}

	links, err := s.jobFileLinks(ctx, rec)
	if err != nil {
		return nil, err
	}

	if links.ReportJSON != "" || links.ProvenanceJSON != "" || len(links.ScannerArtifacts) > 0 ||
		len(links.Screenshots) > 0 {
		job.Artifacts = links
	}

	return job, nil
}

// jobFileLinks lists the API routes that serve a finished job's public files.
func (s *Server) jobFileLinks(ctx context.Context, rec *jobstore.JobRecord) (*models.ArtifactLocation, error) {
	links := &models.ArtifactLocation{}

	if _, inJob := jobRelative(rec.JobID, rec.ReportJSONKey); inJob {
		url, err := publicArtifactURL(rec.JobID, rec.ReportJSONKey)
		if err != nil {
			return nil, err
		}

		links.ReportJSON = url
	} else if rec.ReportJSONKey != "" {
		logging.Warn(ctx, "Report JSON key is outside the job", "key", rec.ReportJSONKey)
	}

	links.ProvenanceJSON = optionalFileLink(ctx, rec.JobID, rec.ProvenanceKey)

	if len(rec.ScannerArtifacts) > 0 {
		links.ScannerArtifacts = scannerFileLinks(ctx, rec)
	}

	if screenshots := s.collectJobScreenshots(ctx, rec); len(screenshots) > 0 {
		links.Screenshots = screenshots
	}

	return links, nil
}

func scannerFileLinks(ctx context.Context, rec *jobstore.JobRecord) map[string]*models.ScannerArtifacts {
	perScanner := make(map[string]*models.ScannerArtifacts, len(rec.ScannerArtifacts))

	for scannerType, files := range rec.ScannerArtifacts {
		perScanner[scannerType] = &models.ScannerArtifacts{
			ScannerType: files.ScannerType,
			ResultsJSON: optionalFileLink(ctx, rec.JobID, files.ResultsKey),
		}
	}

	return perScanner
}

// optionalFileLink returns the API route for key, or "" when the job has no
// such public file.
func optionalFileLink(ctx context.Context, jobID, key string) string {
	if key == "" {
		return ""
	}

	url, err := publicArtifactURL(jobID, key)
	if err != nil {
		logging.Warn(ctx, "Leaving a file out of the job status", "key", key, "error", err)

		return ""
	}

	return url
}
