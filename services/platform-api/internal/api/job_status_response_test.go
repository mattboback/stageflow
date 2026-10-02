// Response construction for job status: exercises job_status_response.go.

package api

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/mattboback/stageflow/internal/jobstore"
	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
	"github.com/mattboback/stageflow/libs/go/models"
)

func TestBuildJobStatusResponse_DoneWithArtifacts(t *testing.T) {
	server, _, _ := newTestServer(t)

	jobID := "job-response-done"
	testJobs(server).put(scanningURLJob(jobID))
	testJobs(server).finish(t, jobID, jobID+"/report.json")

	rec, err := server.statusReader.GetJob(context.Background(), jobID)
	if err != nil {
		t.Fatalf("get job: %v", err)
	}

	job, err := server.buildJobStatusResponse(context.Background(), rec)
	if err != nil {
		t.Fatalf("build response: %v", err)
	}

	if job.Artifacts == nil {
		t.Fatal("expected artifacts to be set")
	}

	if job.Artifacts.ReportJSON == "" {
		t.Fatal("expected report JSON URL")
	}
}

func TestBuildJobStatusResponse_DoneUsesAggregatedReportIssueCount(t *testing.T) {
	server, storage, _ := newTestServer(t)

	jobID := "job-response-report-count"
	reportKey := jobID + "/report.json"
	now := time.Now().UTC()
	infoCount := 2
	axeCount := 1
	seoCount := 4
	reportDoc := report.UnifiedReportV2{
		Version: "2.0.0",
		Meta: report.ReportMeta{
			JobId:       jobID,
			ScannedAt:   &now,
			CompletedAt: &now,
		},
		Summary: report.ReportSummary{
			TotalIssues: 5,
			BySeverity: report.SeverityCounts{
				Critical: 0,
				Serious:  1,
				Moderate: 2,
				Minor:    0,
				Info:     &infoCount,
			},
			ByScanner:       map[string]int{"axe": 1, "seo": 4},
			PagesScanned:    1,
			PagesWithIssues: 1,
		},
		Scanners: []report.ScannerSummary{
			{Id: "axe", Status: report.ScannerStatusSuccess, IssueCount: &axeCount},
			{Id: "seo", Status: report.ScannerStatusSuccess, IssueCount: &seoCount},
		},
		Pages: []report.PageSummary{{
			Id:         "page-1",
			Url:        "https://example.com",
			IssueCount: 5,
			StartedAt:  &now,
			FinishedAt: &now,
		}},
		Issues:    []report.IssueDetail{},
		Artifacts: []report.ReportArtifact{},
		Errors:    []report.ReportError{},
	}

	reportBytes, err := json.Marshal(reportDoc)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}

	storage.uploads[reportKey] = reportBytes

	rec := &jobstore.JobRecord{
		JobID:           jobID,
		State:           models.JobStateDone,
		TotalViolations: 6,
		ReportJSONKey:   reportKey,
	}

	job, err := server.buildJobStatusResponse(context.Background(), rec)
	if err != nil {
		t.Fatalf("build response: %v", err)
	}

	if job.TotalViolations != 5 {
		t.Fatalf("expected normalized violations count 5, got %d", job.TotalViolations)
	}
}

func TestBuildJobStatusResponse_UnfinishedJobHasNoArtifacts(t *testing.T) {
	server, _, _ := newTestServer(t)

	jobID := "job-response-unfinished"
	testJobs(server).put(scanningURLJob(jobID))

	rec, err := server.statusReader.GetJob(context.Background(), jobID)
	if err != nil {
		t.Fatalf("get job: %v", err)
	}

	job, err := server.buildJobStatusResponse(context.Background(), rec)
	if err != nil {
		t.Fatalf("build response: %v", err)
	}

	if job.Artifacts != nil {
		t.Fatal("expected no artifacts for an unfinished job")
	}
}

func TestBuildJobStatusResponse_ReportKeyMustBeThisJobsPublicFile(t *testing.T) {
	server, _, _ := newTestServer(t)
	ctx := context.Background()

	outside := &jobstore.JobRecord{JobID: "job-a", State: models.JobStateDone, ReportJSONKey: "job-b/report.json"}

	job, err := server.buildJobStatusResponse(ctx, outside)
	if err != nil {
		t.Fatalf("report key of another job: %v", err)
	}

	if job.Artifacts != nil {
		t.Fatalf("linked another job's report: %+v", job.Artifacts)
	}

	private := &jobstore.JobRecord{JobID: "job-a", State: models.JobStateDone, ReportJSONKey: "job-a/auth.json"}
	if _, err = server.buildJobStatusResponse(ctx, private); err == nil {
		t.Fatal("a private file was accepted as the report")
	}
}

func TestScannerFileLinks(t *testing.T) {
	rec := &jobstore.JobRecord{
		JobID: "test-per-scanner",
		State: models.JobStateDone,
		ScannerArtifacts: map[string]*jobstore.ScannerArtifactRecord{
			"axe": {
				ScannerType: "axe",
				ResultsKey:  "test-per-scanner/axe/results.json",
			},
			// A scanner that failed before writing results has no file to link.
			"seo": {ScannerType: "seo"},
		},
	}

	perScanner := scannerFileLinks(context.Background(), rec)

	axeArtifacts, exists := perScanner["axe"]
	if !exists {
		t.Fatal("expected axe artifacts")
	}

	if axeArtifacts.ResultsJSON != "/api/v1/jobs/test-per-scanner/files/axe/results.json" {
		t.Fatalf("axe results link = %q", axeArtifacts.ResultsJSON)
	}

	if seo := perScanner["seo"]; seo == nil || seo.ResultsJSON != "" {
		t.Fatalf("seo without results = %+v", seo)
	}

	if axeArtifacts.ScannerType != "axe" {
		t.Fatalf("expected scanner_type=axe, got %q", axeArtifacts.ScannerType)
	}
}

// --- handleJobURLSubmit success + edge cases ---
