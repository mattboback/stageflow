package db

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/mattboback/stageflow/libs/go/models"
)

func TestPrepareScannerLaunchesIsIdempotentAndPreservesResults(t *testing.T) {
	database := setupTestDB(t)
	job := &models.Job{
		ID:        "job-launch-prepare",
		State:     models.JobStateScanning,
		InputType: models.JobInputTypeURLs,
		URLs:      []string{"https://example.com"},
		Config:    models.JobConfig{Modules: []string{"axe", "keyboard"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mustCreateJob(t, database, job)

	if err := database.PrepareScannerLaunches(t.Context(), job.ID, []string{"axe", "keyboard"}); err != nil {
		t.Fatalf("PrepareScannerLaunches() first call error = %v", err)
	}

	if _, err := database.RecordScannerCompletion(t.Context(), job.ID, &models.ScannerResult{
		ScannerType: "axe",
		ResultsPath: "job-launch-prepare/axe/results.json",
		Success:     true,
	}); err != nil {
		t.Fatalf("RecordScannerCompletion() error = %v", err)
	}

	if err := database.PrepareScannerLaunches(t.Context(), job.ID, []string{"axe", "keyboard"}); err != nil {
		t.Fatalf("PrepareScannerLaunches() duplicate call error = %v", err)
	}

	stored, err := database.GetJob(t.Context(), job.ID)
	if err != nil {
		t.Fatalf("GetJob() error = %v", err)
	}

	if len(stored.CompletedScanners) != 1 || stored.CompletedScanners[0] != "axe" {
		t.Fatalf("completed scanners = %v, want [axe]", stored.CompletedScanners)
	}

	if stored.ScannerResults["axe"] == nil {
		t.Fatal("duplicate preparation erased the persisted axe result")
	}
}

func TestClaimScannerLaunchIsAtomic(t *testing.T) {
	database := setupTestDB(t)
	job := &models.Job{
		ID:        "job-launch-claim",
		State:     models.JobStateScanning,
		InputType: models.JobInputTypeURLs,
		Config:    models.JobConfig{Modules: []string{"axe"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mustCreateJob(t, database, job)

	if err := database.PrepareScannerLaunches(t.Context(), job.ID, []string{"axe"}); err != nil {
		t.Fatalf("PrepareScannerLaunches() error = %v", err)
	}

	const contenders = 12

	start := make(chan struct{})
	results := make(chan bool, contenders)
	errs := make(chan error, contenders)

	for range contenders {
		go func() {
			<-start

			claimed, err := database.ClaimScannerLaunch(context.Background(), job.ID, "axe")
			results <- claimed

			errs <- err
		}()
	}

	close(start)

	claims := 0

	for range contenders {
		if err := <-errs; err != nil {
			t.Fatalf("ClaimScannerLaunch() error = %v", err)
		}

		if <-results {
			claims++
		}
	}

	if claims != 1 {
		t.Fatalf("successful claims = %d, want 1", claims)
	}
}

func TestScannerLaunchRecoveryReclaimsLaunchedRecord(t *testing.T) {
	database := setupTestDB(t)
	job := &models.Job{
		ID:        "job-launch-recovery",
		State:     models.JobStateScanning,
		InputType: models.JobInputTypeURLs,
		Config:    models.JobConfig{Modules: []string{"axe"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mustCreateJob(t, database, job)

	if err := database.PrepareScannerLaunches(t.Context(), job.ID, []string{"axe"}); err != nil {
		t.Fatalf("PrepareScannerLaunches() error = %v", err)
	}

	claimed, claimErr := database.ClaimScannerLaunch(t.Context(), job.ID, "axe")
	if claimErr != nil || !claimed {
		t.Fatalf("ClaimScannerLaunch() = (%v, %v), want (true, nil)", claimed, claimErr)
	}

	markErr := database.MarkScannerLaunched(t.Context(), job.ID, "axe", "container-axe")
	if markErr != nil {
		t.Fatalf("MarkScannerLaunched() error = %v", markErr)
	}

	reclaimed, recoveryErr := database.ClaimScannerLaunchRecovery(t.Context(), job.ID, "axe")
	if recoveryErr != nil || !reclaimed {
		t.Fatalf("ClaimScannerLaunchRecovery() = (%v, %v), want (true, nil)", reclaimed, recoveryErr)
	}

	markErr = database.MarkScannerLaunched(t.Context(), job.ID, "axe", "container-axe")
	if markErr != nil {
		t.Fatalf("MarkScannerLaunched() after recovery error = %v", markErr)
	}

	launch, getErr := database.GetScannerLaunch(t.Context(), job.ID, "axe")
	if getErr != nil {
		t.Fatalf("GetScannerLaunch() error = %v", getErr)
	}

	if launch.State != ScannerLaunchLaunched || launch.ContainerID != "container-axe" {
		t.Fatalf("launch = %#v, want launched container-axe", launch)
	}

	if launch.AttemptCount != 2 {
		t.Fatalf("attempt count = %d, want 2", launch.AttemptCount)
	}
}

func TestPrepareScannerLaunchesRejectsChangedScannerSet(t *testing.T) {
	database := setupTestDB(t)
	jobID := fmt.Sprintf("job-launch-mismatch-%d", time.Now().UnixNano())
	mustCreateJob(t, database, &models.Job{
		ID:        jobID,
		State:     models.JobStateScanning,
		InputType: models.JobInputTypeURLs,
		Config:    models.JobConfig{Modules: []string{"axe"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	if err := database.PrepareScannerLaunches(t.Context(), jobID, []string{"axe"}); err != nil {
		t.Fatalf("PrepareScannerLaunches() error = %v", err)
	}

	if err := database.PrepareScannerLaunches(t.Context(), jobID, []string{"keyboard"}); err == nil {
		t.Fatal("PrepareScannerLaunches() accepted a changed scanner set")
	}
}
