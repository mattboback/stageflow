package status

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/mattboback/stageflow/libs/go/models"
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

func TestGetJobDecodesPersistedJSONFields(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().UTC()
	completedAt := now.Add(time.Minute)

	_, err := store.db.ExecContext(ctx, `
		INSERT INTO job_status (
			job_id, state, input_type, created_at, updated_at, completed_at,
			error, total_pages, current_page, total_violations,
			report_json_key, report_key, scan_stage_log_key, scan_recipe_key,
			extraction_stage_log_key, extraction_recipe_key, provenance_key,
			last_stage, last_error_details, expected_scanners, completed_scanners, scanner_artifacts
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		"job-1",
		models.JobStateDone,
		models.JobInputTypeURLs,
		now,
		now,
		completedAt,
		"",
		2,
		2,
		7,
		"job-1/report.json",
		"job-1/report.html",
		"job-1/scan.log",
		"job-1/scan-recipe.json",
		"job-1/extract.log",
		"job-1/extract-recipe.json",
		"job-1/prov.json",
		"",
		"",
		`["axe","lighthouse"]`,
		`["axe"]`,
		`{"axe":{"scanner_type":"axe","results_key":"job-1/axe/results.json","report_key":"job-1/axe/report.html"}}`,
	)
	if err != nil {
		t.Fatalf("insert job row: %v", err)
	}

	rec := mustGet(ctx, t, store, "job-1")
	if rec.State != models.JobStateDone || rec.TotalViolations != 7 {
		t.Fatalf("unexpected base record: %+v", rec)
	}

	if len(rec.ExpectedScanners) != 2 || rec.ExpectedScanners[1] != "lighthouse" {
		t.Fatalf("expected expected scanners decoded, got %+v", rec.ExpectedScanners)
	}

	if len(rec.CompletedScanners) != 1 || rec.CompletedScanners[0] != "axe" {
		t.Fatalf("expected completed scanners decoded, got %+v", rec.CompletedScanners)
	}

	if rec.ScannerArtifacts["axe"] == nil || rec.ScannerArtifacts["axe"].ResultsKey != "job-1/axe/results.json" {
		t.Fatalf("expected scanner artifacts decoded, got %+v", rec.ScannerArtifacts)
	}

	if rec.CompletedAt == nil || !rec.CompletedAt.Equal(completedAt) {
		t.Fatalf("expected completed_at decoded, got %+v", rec.CompletedAt)
	}
}

func TestEnsureJobRowCreatesPendingRecord(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().UTC()

	if err := store.ensureJobRow(ctx, "job-ensure", now); err != nil {
		t.Fatalf("ensureJobRow: %v", err)
	}

	rec := mustGet(ctx, t, store, "job-ensure")
	if rec.State != models.JobStatePending {
		t.Fatalf("expected PENDING, got %s", rec.State)
	}

	if rec.CreatedAt.IsZero() || rec.UpdatedAt.IsZero() {
		t.Fatalf("expected timestamps to be set: %+v", rec)
	}
}

func TestAdvanceStateDoesNotOverrideTerminalStates(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().UTC()

	_, err := store.db.ExecContext(ctx, `
		INSERT INTO job_status (job_id, state, input_type, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, "job-terminal", models.JobStateFailed, models.JobInputTypeURLs, now, now)
	if err != nil {
		t.Fatalf("insert job row: %v", err)
	}

	if advanceErr := store.advanceState(
		ctx,
		"job-terminal",
		models.JobStateScanning,
		now.Add(time.Minute),
	); advanceErr != nil {
		t.Fatalf("advanceState: %v", advanceErr)
	}

	rec := mustGet(ctx, t, store, "job-terminal")
	if rec.State != models.JobStateFailed {
		t.Fatalf("expected terminal FAILED state to remain, got %s", rec.State)
	}
}

func TestSetFailureMarksJobFailed(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().UTC()

	if err := store.ensureJobRow(ctx, "job-failure", now); err != nil {
		t.Fatalf("ensureJobRow: %v", err)
	}

	if err := store.setFailure(ctx, "job-failure", "scanning", "boom", "stacktrace", now.Add(time.Minute)); err != nil {
		t.Fatalf("setFailure: %v", err)
	}

	rec := mustGet(ctx, t, store, "job-failure")
	if rec.State != models.JobStateFailed {
		t.Fatalf("expected FAILED, got %s", rec.State)
	}

	if rec.Error != "boom" || rec.LastStage != "scanning" || rec.LastErrorDetails != "stacktrace" {
		t.Fatalf("failure details not stored: %+v", rec)
	}

	if rec.CompletedAt == nil {
		t.Fatal("expected completed_at to be set")
	}
}
