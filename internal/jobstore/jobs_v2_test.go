package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mattboback/stageflow/libs/go/models"
)

func pendingJob(id string) *models.Job {
	return &models.Job{
		ID:               id,
		State:            models.JobStatePending,
		InputType:        models.JobInputTypeURLs,
		URLs:             []string{"https://example.com"},
		Config:           models.JobConfig{Modules: []string{"axe", "seo"}},
		ExpectedScanners: []string{"axe", "seo"},
	}
}
func TestConcurrentAdmissionAndClaimRespectBudgets(t *testing.T) {
	d := setupTestDB(t)
	ctx := context.Background()

	var (
		accepted atomic.Int32
		wg       sync.WaitGroup
	)

	errs := make(chan error, 30)

	for i := range 30 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			err := d.AdmitJob(ctx, pendingJob(fmt.Sprintf("queue-%02d", i)))
			if err == nil {
				accepted.Add(1)
			} else if !errors.Is(err, ErrQueueFull) {
				errs <- err
			}
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Error(err)
	}

	if accepted.Load() != QueueCapacity {
		t.Fatalf("admitted %d; cap %d", accepted.Load(), QueueCapacity)
	}

	var claimed atomic.Int32

	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			job, err := d.ClaimNext(ctx)
			if err != nil {
				t.Error(err)
			} else if job != nil {
				claimed.Add(1)

				if job.ClaimedAt == nil {
					t.Error("claim timestamp missing")
				}
			}
		}()
	}

	wg.Wait()

	if claimed.Load() != 1 {
		t.Fatalf("claimed %d concurrent jobs", claimed.Load())
	}

	if err := d.AdmitJob(ctx, pendingJob("freed-pending-slot")); err != nil {
		t.Fatalf("claim did not release pending queue slot: %v", err)
	}
}
func TestTerminalCleanupHoldsCapacityUntilDurablyCleaned(t *testing.T) {
	d := setupTestDB(t)

	ctx := context.Background()
	for _, id := range []string{"first", "next"} {
		if err := d.AdmitJob(ctx, pendingJob(id)); err != nil {
			t.Fatal(err)
		}
	}

	job, err := d.ClaimNext(ctx)
	if err != nil || job == nil {
		t.Fatalf("claim: %v", err)
	}

	if err = d.MarkTerminal(ctx, job.ID, models.JobStateFailed, "", 0, "failed", "", "setup"); err != nil {
		t.Fatal(err)
	}

	next, err := d.ClaimNext(ctx)
	if err != nil || next != nil {
		t.Fatalf("claimed before cleanup: %#v %v", next, err)
	}

	if err = d.MarkCleaned(ctx, job.ID); err != nil {
		t.Fatal(err)
	}

	next, err = d.ClaimNext(ctx)
	if err != nil || next == nil {
		t.Fatalf("capacity not released: %#v %v", next, err)
	}
}

//nolint:gocyclo // Sequential assertions cover persisted stage timestamps and invalid transitions.
func TestStageStartsAreAbsoluteAndCannotMoveBackward(t *testing.T) {
	d := setupTestDB(t)
	ctx := context.Background()
	job := pendingJob("absolute")

	job.InputType = models.JobInputTypeZip
	if err := d.AdmitJob(ctx, job); err != nil {
		t.Fatal(err)
	}

	claimed, err := d.ClaimNext(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if err = d.BeginStage(ctx, job.ID, models.JobStateExtracting); err != nil {
		t.Fatal(err)
	}

	initial, err := d.GetJob(ctx, job.ID)
	if err != nil {
		t.Fatal(err)
	}

	if err = d.UpdateScannerProgress(ctx, job.ID, "axe", 4, 10); err != nil {
		t.Fatal(err)
	}

	if err = d.BeginStage(ctx, job.ID, models.JobStateExtracting); err != nil {
		t.Fatal(err)
	}

	if err = d.BeginStage(ctx, job.ID, models.JobStateReady); err != nil {
		t.Fatal(err)
	}

	if err = d.BeginStage(ctx, job.ID, models.JobStateScanning); err != nil {
		t.Fatal(err)
	}

	scanning, err := d.GetJob(ctx, job.ID)
	if err != nil {
		t.Fatal(err)
	}

	if err = d.BeginStage(ctx, job.ID, models.JobStateReady); err == nil {
		t.Fatal("backward transition accepted")
	}

	if err = d.BeginStage(ctx, job.ID, models.JobStateScanning); err != nil {
		t.Fatal(err)
	}

	after, err := d.GetJob(ctx, job.ID)
	if err != nil {
		t.Fatal(err)
	}

	if !after.ClaimedAt.Equal(*claimed.ClaimedAt) || !after.ExtractionStartedAt.Equal(*initial.ExtractionStartedAt) ||
		!after.ScanStartedAt.Equal(*scanning.ScanStartedAt) {
		t.Fatal("progress/repeated tick reset an absolute deadline")
	}
}

//nolint:gocyclo // Exercises one terminal transaction and reload across a database restart.
func TestTerminalScrubAndTotalsSurviveRestart(t *testing.T) {
	d := setupTestDB(t)
	ctx := context.Background()
	job := pendingJob("private")

	job.Config.Auth = json.RawMessage(`{"mode":"form","steps":[{"type":"fill","value":"secret password"}]}`)
	if err := d.AdmitJob(ctx, job); err != nil {
		t.Fatal(err)
	}

	if _, err := d.ClaimNext(ctx); err != nil {
		t.Fatal(err)
	}

	if attempt, err := d.BeginScannerAttempt(ctx, job.ID, "axe"); err != nil || attempt != 1 {
		t.Fatalf("attempt: %d %v", attempt, err)
	}

	if err := d.SetScannerContainer(ctx, job.ID, "axe", "container-1"); err != nil {
		t.Fatal(err)
	}

	if err := d.RecordScannerResult(
		ctx,
		job.ID,
		"partial",
		&models.ScannerResult{
			ScannerType:  "axe",
			Error:        "secret+password failed",
			ResultsPath:  "secret%20password/results.json",
			PagesScanned: 3,
			TotalIssues:  7,
			TotalMs:      50,
		},
	); err != nil {
		t.Fatal(err)
	}

	if err := d.MarkTerminal(
		ctx,
		job.ID,
		models.JobStateFailed,
		"private/report.json",
		7,
		"secret password failed",
		"secret%20password details",
		"scan",
	); err != nil {
		t.Fatal(err)
	}

	if err := d.MarkTerminal(ctx, job.ID, models.JobStateFailed, "incorrect/report.json", 100, "", "", ""); err != nil {
		t.Fatal(err)
	}

	var schema string
	if err := d.db.QueryRowContext(ctx, `SELECT current_schema()`).Scan(&schema); err != nil {
		t.Fatal(err)
	}

	reopened, err := NewDatabase(&Config{URL: testDatabaseURL + "&search_path=" + schema, RequireSchema: true})
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()

	after, err := reopened.GetJob(ctx, job.ID)
	if err != nil {
		t.Fatal(err)
	}

	raw, err := json.Marshal(after)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(raw), "secret") || len(after.Config.Auth) != 0 {
		t.Fatalf("credentials survive terminal scrub: %s", raw)
	}

	if !after.CleanupPending || after.TotalViolations != 7 || after.ReportJSONKey != "private/report.json" ||
		after.ScannerStates["axe"] != "partial" ||
		!after.ScannerResults["axe"].Success ||
		after.ScannerAttempts["axe"] != 1 ||
		after.ScannerContainerIDs["axe"] != "container-1" {
		t.Fatalf("restart lost normalized state: %#v", after)
	}

	if err = d.RecordScannerResult(
		ctx,
		job.ID,
		"success",
		&models.ScannerResult{ScannerType: "seo", Error: "secret password"},
	); err == nil {
		t.Fatal("terminal job accepted a late scanner outcome")
	}

	if progressErr := d.UpdateScannerProgress(ctx, job.ID, "seo", 99, 99); progressErr == nil {
		t.Fatal("terminal job accepted late progress")
	}
}
func TestScannerAttemptsAndProgressAreAtomic(t *testing.T) {
	d := setupTestDB(t)
	ctx := context.Background()

	job := pendingJob("attempts")
	if err := d.AdmitJob(ctx, job); err != nil {
		t.Fatal(err)
	}

	var (
		accepted atomic.Int32
		wg       sync.WaitGroup
	)
	for range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			_, err := d.BeginScannerAttempt(ctx, job.ID, "axe")
			if err == nil {
				accepted.Add(1)
			} else if !errors.Is(err, ErrAttemptsExhausted) {
				t.Error(err)
			}
		}()
	}

	wg.Wait()

	if accepted.Load() != 2 {
		t.Fatalf("%d launch attempts", accepted.Load())
	}

	if err := d.UpdateScannerProgress(ctx, job.ID, "axe", 10, 20); err != nil {
		t.Fatal(err)
	}

	if err := d.UpdateScannerProgress(ctx, job.ID, "axe", 2, 5); err != nil {
		t.Fatal(err)
	}

	read, err := d.GetJob(ctx, job.ID)
	if err != nil {
		t.Fatal(err)
	}

	if read.ScannerProgress["axe"] != 10 || read.TotalPages != 20 {
		t.Fatalf("progress regressed: %#v", read)
	}
}
func TestRetentionDeletionCascadesAndSchemaHasNoLegacyTables(t *testing.T) {
	d := setupTestDB(t)
	ctx := context.Background()

	job := pendingJob("delete")
	if err := d.AdmitJob(ctx, job); err != nil {
		t.Fatal(err)
	}

	if err := d.MarkTerminal(ctx, job.ID, models.JobStateFailed, "", 0, "", "", ""); err != nil {
		t.Fatal(err)
	}

	if err := d.MarkCleaned(ctx, job.ID); err != nil {
		t.Fatal(err)
	}

	rows, err := d.TerminalJobsBefore(ctx, time.Now().Add(time.Minute))
	if err != nil || len(rows) != 1 {
		t.Fatalf("retention rows=%d err=%v", len(rows), err)
	}

	if err = d.DeleteJob(ctx, job.ID); err != nil {
		t.Fatal(err)
	}

	exists, err := d.JobExists(ctx, job.ID)
	if err != nil || exists {
		t.Fatal("job survived deletion")
	}

	var count int
	if err = d.db.QueryRowContext(ctx, `SELECT count(*) FROM job_scanners WHERE job_id=$1`, job.ID).
		Scan(&count); err != nil ||
		count != 0 {
		t.Fatalf("scanner rows survived: %d %v", count, err)
	}

	for _, table := range []string{"scanner_launches", "terminal_events", "maintenance_migrations", "deleted_jobs"} {
		var relation sql.NullString
		if err = d.db.QueryRowContext(ctx, `SELECT to_regclass($1)::text`, table).
			Scan(&relation); err != nil ||
			relation.Valid {
			t.Fatalf("legacy table survives: %s %v", table, err)
		}
	}
}

func TestDeletionKeepsCleanupOwnership(t *testing.T) {
	d := setupTestDB(t)

	ctx := context.Background()
	if err := d.AdmitJob(ctx, pendingJob("guard")); err != nil {
		t.Fatal(err)
	}

	if _, err := d.ClaimNext(ctx); err != nil {
		t.Fatal(err)
	}

	if err := d.DeleteJob(ctx, "guard"); !errors.Is(err, ErrJobBusy) {
		t.Fatalf("claimed deletion lost ownership: %v", err)
	}

	if err := d.MarkTerminal(ctx, "guard", models.JobStateFailed, "", 0, "", "", ""); err != nil {
		t.Fatal(err)
	}

	if err := d.DeleteJob(ctx, "guard"); !errors.Is(err, ErrCleanupPending) {
		t.Fatalf("terminal deletion lost cleanup ownership: %v", err)
	}

	stale, err := d.TerminalJobsBefore(ctx, time.Now().Add(time.Minute))
	if err != nil || len(stale) != 0 {
		t.Fatalf("retention selected uncleaned rows: %v %v", stale, err)
	}

	snapshot, err := (StatusReader{Database: d}).GetJob(ctx, "guard")
	if err != nil || !snapshot.CleanupPending || snapshot.ClaimedAt == nil {
		t.Fatalf("API cannot precheck cleanup ownership: %#v %v", snapshot, err)
	}

	if err = d.MarkCleaned(ctx, "guard"); err != nil {
		t.Fatal(err)
	}

	if err = d.DeleteJob(ctx, "guard"); err != nil {
		t.Fatal(err)
	}

	if err = d.DeleteJob(ctx, "guard"); err != nil {
		t.Fatalf("repeated deletion is not idempotent: %v", err)
	}
}
func TestRequireSchemaNeverCreatesMissingTables(t *testing.T) {
	d := setupTestDB(t)
	ctx := context.Background()

	var schema string
	if err := d.db.QueryRowContext(ctx, `SELECT current_schema()`).Scan(&schema); err != nil {
		t.Fatal(err)
	}

	if _, err := d.db.ExecContext(ctx, `DROP TABLE job_scanners; DROP TABLE jobs`); err != nil {
		t.Fatal(err)
	}

	opened, err := NewDatabase(&Config{URL: testDatabaseURL + "&search_path=" + schema, RequireSchema: true})
	if err == nil {
		opened.Close()
		t.Fatal("API silently initialized the missing schema")
	}

	var jobs, scanners sql.NullString
	if err = d.db.QueryRowContext(ctx, `SELECT to_regclass('jobs')::text,to_regclass('job_scanners')::text`).
		Scan(&jobs, &scanners); err != nil || jobs.Valid ||
		scanners.Valid {
		t.Fatalf("require-schema mutated tables: %v %v %v", jobs, scanners, err)
	}
}
func TestConcurrentTerminalScrubRejectsLateScannerSecrets(t *testing.T) {
	d := setupTestDB(t)
	ctx := context.Background()

	for i := range 20 {
		job := pendingJob(fmt.Sprintf("scrub-race-%d", i))
		job.State = models.JobStateScanning
		job.Config.Auth = json.RawMessage(`{"mode":"form","steps":[{"type":"fill","value":"race-secret"}]}`)
		mustCreateJob(t, d, job)

		start := make(chan struct{})
		errs := make(chan error, 2)

		var wg sync.WaitGroup

		wg.Add(2)
		go func() {
			defer wg.Done()

			<-start

			errs <- d.MarkTerminal(ctx, job.ID, models.JobStateFailed, "", 0, "race-secret failed", "", "")
		}()
		go func() {
			defer wg.Done()

			<-start

			errs <- d.RecordScannerResult(ctx, job.ID, "failed", &models.ScannerResult{ScannerType: "axe", Error: "race-secret scanner failed"})
		}()

		close(start)
		wg.Wait()
		close(errs)

		for err := range errs {
			if err != nil && !errors.Is(err, ErrJobNotFound) {
				t.Fatal(err)
			}
		}

		after, err := d.GetJob(ctx, job.ID)
		if err != nil {
			t.Fatal(err)
		}

		raw, err := json.Marshal(after)
		if err != nil {
			t.Fatal(err)
		}

		if strings.Contains(string(raw), "race-secret") {
			t.Fatalf("concurrent outcome escaped terminal scrub: %s", raw)
		}
	}
}
