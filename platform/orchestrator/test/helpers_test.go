package test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	report "github.com/mattboback/stageflow/packages/contracts/report/generated/go"
	"github.com/mattboback/stageflow/packages/shared-go/models"
	"github.com/mattboback/stageflow/packages/shared-go/storage"
	db "github.com/mattboback/stageflow/platform/orchestrator/internal/adapters/repository"
	"github.com/mattboback/stageflow/platform/orchestrator/internal/orchestrator"
)

var e2eTestSchemaCounter uint64

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

	if uploadErr := store.UploadFile(
		context.Background(),
		storage.BucketArtifacts,
		resultsPath,
		bytes.NewReader(data),
		int64(len(data)),
	); uploadErr != nil {
		t.Fatalf("seed scan results: %v", uploadErr)
	}
}

func setupE2ETest(
	t *testing.T,
) (*orchestrator.Orchestrator, *db.Database, *mockPodmanClient, *mockPublisher, *memoryStorage) {
	t.Helper()

	var orch *orchestrator.Orchestrator

	admin, err := sql.Open("pgx", testDatabaseURL)
	if err != nil {
		t.Fatalf("Failed to connect admin database: %v", err)
	}

	schema := fmt.Sprintf("t_%d_%d", time.Now().UnixNano(), atomic.AddUint64(&e2eTestSchemaCounter, 1))

	createSchemaQuery := fmt.Sprintf("CREATE SCHEMA %s", quoteIdentifier(schema))
	if _, execErr := admin.ExecContext(context.Background(), createSchemaQuery); execErr != nil {
		t.Fatalf("Failed to create test schema: %v", execErr)
	}

	databaseURL := fmt.Sprintf("%s&search_path=%s", testDatabaseURL, schema)

	database, err := db.NewDatabase(&db.Config{URL: databaseURL})
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	podmanClient := newMockPodmanClient()
	publisher := newMockPublisher()
	mem := newMemoryStorage()

	orch = orchestrator.NewOrchestrator(&orchestrator.Config{
		PodmanClient:   podmanClient,
		Database:       database,
		Publisher:      publisher,
		Storage:        mem,
		StagingStorage: mem,
	})

	t.Cleanup(func() {
		if orch != nil {
			orch.WaitForMonitors()
		}

		if closeErr := database.Close(); closeErr != nil {
			t.Fatalf("Failed to close database: %v", closeErr)
		}

		dropSchemaQuery := fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", quoteIdentifier(schema))
		if _, dropErr := admin.ExecContext(
			context.Background(),
			dropSchemaQuery,
		); dropErr != nil {
			t.Fatalf("Failed to drop test schema: %v", dropErr)
		}

		if closeErr := admin.Close(); closeErr != nil {
			t.Fatalf("Failed to close admin database: %v", closeErr)
		}
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

func quoteIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}
