package test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	report "github.com/mattboback/stageflow/packages/contracts/report/generated/go"
	"github.com/mattboback/stageflow/packages/shared-go/models"
	"github.com/mattboback/stageflow/packages/shared-go/storage"
	"github.com/mattboback/stageflow/platform/orchestrator/internal/db"
	"github.com/mattboback/stageflow/platform/orchestrator/internal/orchestrator"
)

func seedScanResults(t *testing.T, store *memoryStorage, jobID, resultsPath string) {
	t.Helper()

	now := time.Now().UTC()
	results := report.UnifiedReportV2{
		Version: "2.0.0",
		Meta: report.ReportMeta{
			JobId:       jobID,
			ScannedAt:   &now,
			CompletedAt: &now,
		},
		Summary: report.ReportSummary{
			TotalIssues: 0,
			BySeverity: report.SeverityCounts{
				Critical: 0,
				Serious:  0,
				Moderate: 0,
				Minor:    0,
			},
			ByScanner:       map[string]int{},
			PagesScanned:    1,
			PagesWithIssues: 0,
		},
		Pages: []report.PageSummary{
			{
				Id:         "page-1",
				Url:        "https://example.com/" + jobID,
				Path:       strPtr("/"),
				IssueCount: 0,
				DurationMs: 0,
				StartedAt:  &now,
				FinishedAt: &now,
			},
		},
		Issues: nil,
	}

	data, err := json.Marshal(results)
	if err != nil {
		t.Fatalf("marshal scan results: %v", err)
	}

	if err := store.UploadFile(context.Background(), storage.BucketArtifacts, resultsPath, bytes.NewReader(data), int64(len(data))); err != nil {
		t.Fatalf("seed scan results: %v", err)
	}
}

func setupE2ETest(t *testing.T) (*orchestrator.Orchestrator, *db.Database, *mockPodmanClient, *mockPublisher, *memoryStorage) {
	t.Helper()
	database, err := db.NewDatabase(&db.Config{Path: ":memory:"})
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("Failed to close database: %v", err)
		}
	})

	podmanClient := newMockPodmanClient()
	publisher := newMockPublisher()
	mem := newMemoryStorage()

	orch := orchestrator.NewOrchestrator(&orchestrator.Config{
		PodmanClient:   podmanClient,
		Database:       database,
		Publisher:      publisher,
		Storage:        mem,
		StagingStorage: mem,
	})

	return orch, database, podmanClient, publisher, mem
}

func mustGetJob(t *testing.T, database *db.Database, jobID string) *models.Job {
	t.Helper()
	job, err := database.GetJob(context.Background(), jobID)
	if err != nil {
		t.Fatalf("Failed to get job %s: %v", jobID, err)
	}
	return job
}

func strPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
