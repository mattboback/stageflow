package db

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/mattboback/stageflow/libs/go/models"
)

func TestRecordScannerCompletionConcurrency(t *testing.T) {
	db := setupTestDBFile(t)

	// Run multiple iterations to increase the chance of hitting the race window.
	const iterations = 10

	for i := range iterations {
		jobID := fmt.Sprintf("job-concurrent-%d", i)

		job := &models.Job{
			ID:        jobID,
			State:     models.JobStateScanning,
			InputType: "zip",
			InputPath: "test.zip",
			Config:    models.JobConfig{Modules: []string{"axe", "lighthouse"}},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mustCreateJob(t, db, job)

		if err := db.SetExpectedScanners(context.Background(), jobID, []string{"axe", "lighthouse"}); err != nil {
			t.Fatalf("iter %d: failed to set expected scanners: %v", i, err)
		}

		start := make(chan struct{})

		type result struct {
			allComplete bool
			err         error
		}

		results := make(chan result, 2)

		run := func(scanner string) {
			<-start

			allComplete, err := db.RecordScannerCompletion(context.Background(), jobID, &models.ScannerResult{
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
			t.Fatalf("iter %d: scanner1 error: %v", i, res1.err)
		}

		if res2.err != nil {
			t.Fatalf("iter %d: scanner2 error: %v", i, res2.err)
		}

		// Exactly one of them should report completion (the second finisher).
		if (res1.allComplete && res2.allComplete) || (!res1.allComplete && !res2.allComplete) {
			t.Fatalf(
				"iter %d: expected exactly one completion signal, got res1=%v res2=%v",
				i, res1.allComplete, res2.allComplete,
			)
		}

		stored, err := db.GetJob(context.Background(), jobID)
		if err != nil {
			t.Fatalf("iter %d: failed to get job: %v", i, err)
		}

		if len(stored.CompletedScanners) != 2 {
			t.Fatalf("iter %d: expected 2 completed scanners, got %d", i, len(stored.CompletedScanners))
		}

		if len(stored.ScannerResults) != 2 {
			t.Fatalf("iter %d: expected 2 scanner results, got %d", i, len(stored.ScannerResults))
		}
	}
}
