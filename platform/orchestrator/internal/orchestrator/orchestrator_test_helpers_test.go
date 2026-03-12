package orchestrator

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	report "github.com/mattboback/stageflow/packages/contracts/report/generated/go"
	"github.com/mattboback/stageflow/packages/shared-go/events"
	"github.com/mattboback/stageflow/packages/shared-go/models"
	"github.com/mattboback/stageflow/packages/shared-go/storage"
	db "github.com/mattboback/stageflow/platform/orchestrator/internal/adapters/repository"
	podman "github.com/mattboback/stageflow/platform/orchestrator/internal/adapters/runtime"
)

var orchestratorTestSchemaCounter uint64

// mockPodmanClient is a mock Podman client for testing.
type mockPodmanClient struct {
	createPodFunc       func(ctx context.Context, req *podman.PodCreateRequest) (*podman.PodCreateResponse, error)
	stopPodFunc         func(ctx context.Context, podID string) error
	removePodFunc       func(ctx context.Context, podID string, force bool) error
	createVolumeFunc    func(ctx context.Context, name string) error
	inspectVolumeFunc   func(ctx context.Context, name string) (*podman.VolumeInfo, error)
	removeVolumeFunc    func(ctx context.Context, name string, force bool) error
	createContainerFunc func(ctx context.Context, req *podman.ContainerCreateRequest) (*podman.ContainerCreateResponse, error)
	startContainerFunc  func(ctx context.Context, containerID string) error
	waitContainerFunc   func(ctx context.Context, containerID string) (*podman.ContainerWaitResponse, error)
	removeContainerFunc func(ctx context.Context, containerID string, force bool) error
	getLogsFunc         func(ctx context.Context, containerID string, stdout, stderr bool) (string, error)
}

func (m *mockPodmanClient) CreatePod(
	ctx context.Context,
	req *podman.PodCreateRequest,
) (*podman.PodCreateResponse, error) {
	if m.createPodFunc != nil {
		return m.createPodFunc(ctx, req)
	}

	return &podman.PodCreateResponse{ID: "pod-12345"}, nil
}

func (m *mockPodmanClient) StopPod(ctx context.Context, podID string) error {
	if m.stopPodFunc != nil {
		return m.stopPodFunc(ctx, podID)
	}

	return nil
}

func (m *mockPodmanClient) RemovePod(ctx context.Context, podID string, force bool) error {
	if m.removePodFunc != nil {
		return m.removePodFunc(ctx, podID, force)
	}

	return nil
}

func (m *mockPodmanClient) CreateVolume(ctx context.Context, name string) error {
	if m.createVolumeFunc != nil {
		return m.createVolumeFunc(ctx, name)
	}

	return nil
}

func (m *mockPodmanClient) InspectVolume(ctx context.Context, name string) (*podman.VolumeInfo, error) {
	if m.inspectVolumeFunc != nil {
		return m.inspectVolumeFunc(ctx, name)
	}

	return &podman.VolumeInfo{
		Name:       name,
		Mountpoint: fmt.Sprintf("/volumes/%s/_data", name),
	}, nil
}

func (m *mockPodmanClient) RemoveVolume(ctx context.Context, name string, force bool) error {
	if m.removeVolumeFunc != nil {
		return m.removeVolumeFunc(ctx, name, force)
	}

	return nil
}

func (m *mockPodmanClient) CreateContainer(
	ctx context.Context,
	req *podman.ContainerCreateRequest,
) (*podman.ContainerCreateResponse, error) {
	if m.createContainerFunc != nil {
		return m.createContainerFunc(ctx, req)
	}

	return &podman.ContainerCreateResponse{ID: "container-12345"}, nil
}

func (m *mockPodmanClient) StartContainer(ctx context.Context, containerID string) error {
	if m.startContainerFunc != nil {
		return m.startContainerFunc(ctx, containerID)
	}

	return nil
}

func (m *mockPodmanClient) WaitContainer(
	ctx context.Context,
	containerID string,
) (*podman.ContainerWaitResponse, error) {
	if m.waitContainerFunc != nil {
		return m.waitContainerFunc(ctx, containerID)
	}

	return &podman.ContainerWaitResponse{StatusCode: 0}, nil
}

func (m *mockPodmanClient) RemoveContainer(ctx context.Context, containerID string, force bool) error {
	if m.removeContainerFunc != nil {
		return m.removeContainerFunc(ctx, containerID, force)
	}

	return nil
}

func (m *mockPodmanClient) GetContainerLogs(
	ctx context.Context,
	containerID string,
	stdout, stderr bool,
) (string, error) {
	if m.getLogsFunc != nil {
		return m.getLogsFunc(ctx, containerID, stdout, stderr)
	}

	return "", nil
}

// mockPublisher is a mock event publisher for testing.
type mockPublisher struct {
	mu                    sync.Mutex
	publishedJobCompleted []*events.JobCompletedPayload
	publishedJobFailed    []*events.JobFailedPayload
}

func (m *mockPublisher) PublishJobCompleted(_ context.Context, payload *events.JobCompletedPayload) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.publishedJobCompleted = append(m.publishedJobCompleted, payload)

	return nil
}

func (m *mockPublisher) PublishJobFailed(_ context.Context, payload *events.JobFailedPayload) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.publishedJobFailed = append(m.publishedJobFailed, payload)

	return nil
}

func (m *mockPublisher) failedCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return len(m.publishedJobFailed)
}

func (m *mockPublisher) completedCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return len(m.publishedJobCompleted)
}

func (m *mockPublisher) firstCompleted() *events.JobCompletedPayload {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.publishedJobCompleted) == 0 {
		return nil
	}

	return m.publishedJobCompleted[0]
}

func newInMemoryDB(t *testing.T) *db.Database {
	t.Helper()

	admin, err := sql.Open("pgx", testDatabaseURL)
	if err != nil {
		t.Fatalf("Failed to connect admin database: %v", err)
	}

	schema := fmt.Sprintf(
		"t_%d_%d",
		time.Now().UnixNano(),
		atomic.AddUint64(&orchestratorTestSchemaCounter, 1),
	)

	createSchemaQuery := fmt.Sprintf("CREATE SCHEMA %s", quoteIdentifier(schema))
	if _, execErr := admin.ExecContext(context.Background(), createSchemaQuery); execErr != nil {
		t.Fatalf("Failed to create test schema: %v", execErr)
	}

	databaseURL := fmt.Sprintf("%s&search_path=%s", testDatabaseURL, schema)

	database, err := db.NewDatabase(&db.Config{URL: databaseURL})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	t.Cleanup(func() {
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

	return database
}

func quoteIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}

func setupTestOrchestrator(t *testing.T) (*Orchestrator, *db.Database, *mockPublisher, *memoryStorage) {
	t.Helper()
	return setupTestOrchestratorWithConfig(t, func(*Config) {})
}

func setupTestOrchestratorWithConfig(
	t *testing.T,
	configure func(*Config),
) (*Orchestrator, *db.Database, *mockPublisher, *memoryStorage) {
	t.Helper()
	database := newInMemoryDB(t)

	publisher := &mockPublisher{
		publishedJobCompleted: make([]*events.JobCompletedPayload, 0),
		publishedJobFailed:    make([]*events.JobFailedPayload, 0),
	}

	mem := newMemoryStorage()

	config := &Config{
		PodmanClient:   &mockPodmanClient{},
		Database:       database,
		Publisher:      publisher,
		Storage:        mem,
		StagingStorage: mem,
	}
	configure(config)

	orchestrator := NewOrchestrator(config)

	// Wait for monitor goroutines before schema teardown to prevent deadlock.
	// t.Cleanup runs LIFO, so this runs before the schema DROP in newInMemoryDB.
	t.Cleanup(orchestrator.WaitForMonitors)

	return orchestrator, database, publisher, mem
}

func insertJob(t *testing.T, database *db.Database, job *models.Job) {
	t.Helper()

	if err := database.CreateJob(context.Background(), job); err != nil {
		t.Fatalf("Failed to insert job: %v", err)
	}
}

func seedScanResults(t *testing.T, store *memoryStorage, jobID, resultsPath string) {
	t.Helper()

	now := time.Now().UTC()
	results := report.UnifiedReportV2{
		Version: testReportVersion,
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
				Path:       testStringPtr("/"),
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
