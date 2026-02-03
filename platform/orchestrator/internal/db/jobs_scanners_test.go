package db

import (
	"context"
	"testing"
	"time"

	"github.com/mattboback/stageflow/packages/shared-go/models"
)

func TestRecordScannerCompletionConcurrency(t *testing.T) {
	db := setupTestDBFile(t)

	job := &models.Job{
		ID:        "job-concurrent",
		State:     models.JobStateScanning,
		InputType: "zip",
		InputPath: "test.zip",
		Config:    models.JobConfig{Modules: []string{"axe", "lighthouse"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mustCreateJob(t, db, job)

	if err := db.SetExpectedScanners(context.Background(), job.ID, []string{"axe", "lighthouse"}); err != nil {
		t.Fatalf("failed to set expected scanners: %v", err)
	}

	start := make(chan struct{})
	type result struct {
		allComplete bool
		err         error
	}
	results := make(chan result, 2)

	run := func(scanner string) {
		<-start
		allComplete, err := db.RecordScannerCompletion(context.Background(), job.ID, &models.ScannerResult{
			ScannerType: scanner,
			ResultsPath: "path/" + scanner + ".json",
			Success:     true,
		})
		results <- result{allComplete: allComplete, err: err}
	}

	go run("axe")
	go run("lighthouse")

	close(start)

	res1 := <-results
	res2 := <-results

	if res1.err != nil {
		t.Fatalf("scanner1 error: %v", res1.err)
	}
	if res2.err != nil {
		t.Fatalf("scanner2 error: %v", res2.err)
	}

	// Exactly one of them should report completion (the second finisher).
	if (res1.allComplete && res2.allComplete) || (!res1.allComplete && !res2.allComplete) {
		t.Fatalf(
			"expected exactly one completion signal, got res1=%v res2=%v",
			res1.allComplete,
			res2.allComplete,
		)
	}

	stored, err := db.GetJob(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("failed to get job: %v", err)
	}

	if len(stored.CompletedScanners) != 2 {
		t.Fatalf("expected 2 completed scanners, got %d", len(stored.CompletedScanners))
	}

	if len(stored.ScannerResults) != 2 {
		t.Fatalf("expected 2 scanner results, got %d", len(stored.ScannerResults))
	}
}
