// Response construction for job status: exercises job_status_response.go.

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/models"
	storagepkg "github.com/mattboback/stageflow/libs/go/storage"
	"github.com/mattboback/stageflow/services/platform-api/internal/jobstatus"
	"github.com/mattboback/stageflow/services/platform-api/internal/status"
)

func TestBuildJobStatusResponse_DoneWithArtifacts(t *testing.T) {
	server, _, _ := newTestServer(t)

	jobID := "job-response-done"
	applyTestSignal(t, server, jobstatus.Signal{
		Kind: jobstatus.SignalJobCreated,
		JobCreated: &events.JobCreatedPayload{
			JobID:     jobID,
			InputType: models.JobInputTypeURLs,
			URLs:      []string{"https://example.com"},
		},
	})
	applyTestSignal(t, server, jobstatus.Signal{
		Kind: jobstatus.SignalJobCompleted,
		JobCompleted: &events.JobCompletedPayload{
			JobID: jobID,
			Artifacts: events.ArtifactLocations{
				ReportJSON: jobID + "/report.json",
				ReportHTML: jobID + "/report.html",
			},
		},
	})

	rec, err := server.jobStatus.Current(context.Background(), jobID)
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

	if job.Artifacts.ReportHTML == "" {
		t.Fatal("expected report HTML URL")
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

	storage.uploads[fmt.Sprintf("%s::%s", storagepkg.BucketArtifacts, reportKey)] = reportBytes

	rec := &status.JobRecord{
		JobID:           jobID,
		State:           models.JobStateDone,
		TotalViolations: 6,
		ReportJSONKey:   reportKey,
		ReportKey:       jobID + "/report.html",
	}

	job, err := server.buildJobStatusResponse(context.Background(), rec)
	if err != nil {
		t.Fatalf("build response: %v", err)
	}

	if job.TotalViolations != 5 {
		t.Fatalf("expected normalized violations count 5, got %d", job.TotalViolations)
	}
}

func TestBuildJobStatusResponse_Pending(t *testing.T) {
	server, _, _ := newTestServer(t)

	jobID := "job-response-pending"
	applyTestSignal(t, server, jobstatus.Signal{
		Kind: jobstatus.SignalJobCreated,
		JobCreated: &events.JobCreatedPayload{
			JobID:     jobID,
			InputType: models.JobInputTypeURLs,
			URLs:      []string{"https://example.com"},
		},
	})

	rec, err := server.jobStatus.Current(context.Background(), jobID)
	if err != nil {
		t.Fatalf("get job: %v", err)
	}

	job, err := server.buildJobStatusResponse(context.Background(), rec)
	if err != nil {
		t.Fatalf("build response: %v", err)
	}

	if job.Artifacts != nil {
		t.Fatal("expected no artifacts for pending job")
	}
}

func TestBuildPerScannerArtifacts(t *testing.T) {
	server, _, _ := newTestServer(t)

	rec := &status.JobRecord{
		JobID: "test-per-scanner",
		State: models.JobStateDone,
		ScannerArtifacts: map[string]*status.ScannerArtifactRecord{
			"axe": {
				ScannerType: "axe",
				ResultsKey:  "test-per-scanner/axe/results.json",
				ReportKey:   "test-per-scanner/axe/report.html",
				StageLogKey: "test-per-scanner/axe/scan-stage-log.json",
				RecipeKey:   "test-per-scanner/axe/scan-recipe.json",
			},
		},
	}

	perScanner, ok := server.buildPerScannerArtifacts(context.Background(), rec)
	if !ok {
		t.Fatal("expected buildPerScannerArtifacts to return ok=true")
	}

	axeArtifacts, exists := perScanner["axe"]
	if !exists {
		t.Fatal("expected axe artifacts")
	}

	if axeArtifacts.ResultsJSON == "" {
		t.Fatal("expected results JSON URL for axe")
	}

	if axeArtifacts.ScannerType != "axe" {
		t.Fatalf("expected scanner_type=axe, got %q", axeArtifacts.ScannerType)
	}
}

// --- handleJobURLSubmit success + edge cases ---
