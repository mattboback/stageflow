package status

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/mattboback/stageflow/packages/shared-go/events"
	"github.com/mattboback/stageflow/packages/shared-go/models"
)

func newTestStore(t *testing.T) (store *Store, cleanup func()) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "status.db")

	var err error

	store, err = NewStore(&Config{Path: path})
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	cleanup = func() {
		_ = store.Close()
	}

	return store, cleanup
}

func mustGet(ctx context.Context, t *testing.T, store *Store, jobID string) *JobRecord {
	t.Helper()

	rec, err := store.GetJob(ctx, jobID)
	if err != nil {
		t.Fatalf("get job %s: %v", jobID, err)
	}

	return rec
}

func TestClampPercentage(t *testing.T) {
	cases := []struct {
		current  int
		total    int
		expected int
	}{
		{-1, 10, 0},
		{0, 0, 0},
		{5, 10, 50},
		{15, 10, 100},
	}

	for _, tc := range cases {
		if got := clampPercentage(tc.current, tc.total); got != tc.expected {
			t.Fatalf("clampPercentage(%d, %d) = %d, want %d", tc.current, tc.total, got, tc.expected)
		}
	}
}

func TestJobRecordToModel(t *testing.T) {
	now := time.Now().UTC()
	rec := &JobRecord{
		JobID:             "abc",
		State:             models.JobStateScanning,
		Error:             "oops",
		TotalPages:        5,
		CurrentPage:       2,
		ExpectedScanners:  []string{"axe", "lighthouse"},
		CompletedScanners: []string{"axe"},
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	model := rec.ToModel()
	if model.ID != rec.JobID || model.State != rec.State {
		t.Fatalf("basic fields not copied: %#v", model)
	}

	if model.Progress == nil || model.Progress.Percentage != 40 {
		t.Fatalf("expected progress at 40%%, got %#v", model.Progress)
	}

	if len(model.RemainingScanners) != 1 || model.RemainingScanners[0] != "lighthouse" {
		t.Fatalf("expected remaining scanner lighthouse, got %#v", model.RemainingScanners)
	}

	rec.TotalPages = 0

	model = rec.ToModel()
	if model.Progress != nil {
		t.Fatalf("expected nil progress when total pages is zero, got %#v", model.Progress)
	}
}

func TestNewStoreRequiresPath(t *testing.T) {
	if _, err := NewStore(nil); err == nil {
		t.Fatalf("expected error when config is nil")
	}

	if _, err := NewStore(&Config{}); err == nil {
		t.Fatalf("expected error when path is empty")
	}
}

func TestStoreLifecycleUpdates(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	ctx := context.Background()
	jobID := "job-42"

	assertZipJobCreated(ctx, t, store, jobID)
	assertExtractionReady(ctx, t, store, jobID)
	assertScanPageCompleted(ctx, t, store, jobID)
	assertScanCompleted(ctx, t, store, jobID)
	assertJobCompleted(ctx, t, store, jobID)
}

func assertZipJobCreated(ctx context.Context, t *testing.T, store *Store, jobID string) {
	t.Helper()

	if err := store.HandleJobCreated(ctx, &events.JobCreatedPayload{
		JobID:     jobID,
		InputType: models.JobInputTypeZip,
		InputPath: "/tmp/archive.zip",
		Config:    models.JobConfig{Modules: []string{"axe"}, Screenshot: true},
	}); err != nil {
		t.Fatalf("job created: %v", err)
	}

	rec := mustGet(ctx, t, store, jobID)
	if rec.State != models.JobStateExtracting {
		t.Fatalf("expected initial state EXTRACTING for zip jobs, got %s", rec.State)
	}

	if len(rec.ExpectedScanners) != 1 || rec.ExpectedScanners[0] != "axe" {
		t.Fatalf("expected requested scanners to be persisted, got %#v", rec.ExpectedScanners)
	}
}

func assertExtractionReady(ctx context.Context, t *testing.T, store *Store, jobID string) {
	t.Helper()

	if err := store.HandleExtractionReady(ctx, &events.ExtractionReadyPayload{
		JobID:                  jobID,
		TotalPages:             5,
		StageLogPath:           "extract.log",
		RecipePath:             "extract-recipe.json",
		ProvenanceArtifactPath: "prov.json",
	}); err != nil {
		t.Fatalf("extraction ready: %v", err)
	}

	rec := mustGet(ctx, t, store, jobID)
	if rec.State != models.JobStateReady {
		t.Fatalf("expected state READY_TO_SCAN, got %s", rec.State)
	}

	if rec.TotalPages != 5 || rec.ExtractionStageLogKey != "extract.log" || rec.ProvenanceKey != "prov.json" {
		t.Fatalf("extraction fields not persisted: %#v", rec)
	}
}

func assertScanPageCompleted(ctx context.Context, t *testing.T, store *Store, jobID string) {
	t.Helper()

	if err := store.HandleScanPageCompleted(ctx, &events.ScanPageCompletedPayload{
		JobID:      jobID,
		PageIndex:  3,
		TotalPages: 5,
	}); err != nil {
		t.Fatalf("scan page completed: %v", err)
	}

	rec := mustGet(ctx, t, store, jobID)
	if rec.State != models.JobStateScanning || rec.CurrentPage != 3 {
		t.Fatalf("scan progress not updated: %#v", rec)
	}
}

func assertScanCompleted(ctx context.Context, t *testing.T, store *Store, jobID string) {
	t.Helper()

	if err := store.HandleScanCompleted(ctx, &events.ScanCompletedPayload{
		JobID:             jobID,
		ScannerType:       "axe",
		ResultsPath:       "results.json",
		ReportPath:        "report.html",
		StageLogPath:      "scan.log",
		RecipePath:        "scan-recipe.json",
		TotalPagesScanned: 5,
		Summary:           events.ScanSummary{TotalViolations: 7},
	}); err != nil {
		t.Fatalf("scan completed: %v", err)
	}

	rec := mustGet(ctx, t, store, jobID)
	if rec.State != models.JobStateScanning || rec.TotalViolations != 7 || rec.CurrentPage != 5 {
		t.Fatalf("scan completion not applied: %#v", rec)
	}

	if len(rec.CompletedScanners) != 1 || rec.CompletedScanners[0] != "axe" {
		t.Fatalf("expected completed scanner tracking, got %#v", rec.CompletedScanners)
	}
}

func assertJobCompleted(ctx context.Context, t *testing.T, store *Store, jobID string) {
	t.Helper()

	if err := store.HandleJobCompleted(ctx, &events.JobCompletedPayload{
		JobID:  jobID,
		Status: "success",
		Artifacts: events.ArtifactLocations{
			ReportJSON:     "report.json",
			ReportHTML:     "report.html",
			ScanStageLog:   "scan.log",
			ScanRecipe:     "scan-recipe.json",
			ProvenanceJSON: "prov.json",
		},
	}); err != nil {
		t.Fatalf("job completed: %v", err)
	}

	rec := mustGet(ctx, t, store, jobID)
	if rec.State != models.JobStateDone || rec.CompletedAt == nil {
		t.Fatalf("job not marked done: %#v", rec)
	}

	if rec.ReportJSONKey == "" || rec.ReportKey == "" || rec.ScanStageLogKey == "" || rec.ProvenanceKey == "" {
		t.Fatalf("artifact keys missing: %#v", rec)
	}

	if rec.Error != "" || rec.LastStage != "" || rec.LastErrorDetails != "" {
		t.Fatalf("error fields should be cleared after success: %#v", rec)
	}
}

func TestHandleJobCreatedURLInitialState(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	ctx := context.Background()
	jobID := "job-urls"

	if err := store.HandleJobCreated(ctx, &events.JobCreatedPayload{
		JobID:     jobID,
		InputType: models.JobInputTypeURLs,
		URLs:      []string{"https://example.com", "https://example.com/about"},
		Config:    models.JobConfig{Modules: []string{"axe"}, Screenshot: true},
	}); err != nil {
		t.Fatalf("job created: %v", err)
	}

	rec := mustGet(ctx, t, store, jobID)
	if rec.State != models.JobStateScanning {
		t.Fatalf("expected initial state SCANNING for URL jobs, got %s", rec.State)
	}

	if rec.TotalPages != 2 || rec.CurrentPage != 0 {
		t.Fatalf("expected initial progress (0/2), got current=%d total=%d", rec.CurrentPage, rec.TotalPages)
	}

	if len(rec.ExpectedScanners) != 1 || rec.ExpectedScanners[0] != "axe" {
		t.Fatalf("expected requested scanners to be persisted, got %#v", rec.ExpectedScanners)
	}
}

func TestHandleJobFailedCreatesProjection(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	ctx := context.Background()

	payload := &events.JobFailedPayload{
		JobID:        "missing",
		Stage:        "scanning",
		Error:        "boom",
		ErrorDetails: "stacktrace",
	}
	if err := store.HandleJobFailed(ctx, payload); err != nil {
		t.Fatalf("job failed: %v", err)
	}

	rec := mustGet(ctx, t, store, payload.JobID)
	if rec.State != models.JobStateFailed {
		t.Fatalf("expected FAILED state, got %s", rec.State)
	}

	if rec.Error != payload.Error || rec.LastStage != payload.Stage || rec.LastErrorDetails != payload.ErrorDetails {
		t.Fatalf("failure details not stored: %#v", rec)
	}

	if rec.CompletedAt == nil {
		t.Fatalf("expected completed_at to be set on failure")
	}
}
