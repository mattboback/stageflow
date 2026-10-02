package db

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/mattboback/stageflow/libs/go/models"
)

func TestDeletionCannotBeUndoneByConcurrentEventReplay(t *testing.T) {
	database := setupTestDB(t)
	ctx := context.Background()

	job := &models.Job{
		ID:        "job-delete-replay",
		State:     models.JobStatePending,
		InputType: models.JobInputTypeURLs,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := database.CreateJob(ctx, job); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() { defer wg.Done(); _, _ = database.CreateJobIfAbsent(ctx, job) }()
	}

	if err := database.DeleteJob(ctx, job.ID); err != nil {
		t.Fatal(err)
	}

	wg.Wait()

	created, err := database.CreateJobIfAbsent(ctx, job)
	if err != nil || created {
		t.Fatalf("deleted job recreated: %v %v", created, err)
	}

	exists, err := database.JobExists(ctx, job.ID)
	if err != nil || exists {
		t.Fatalf("deleted row still visible: %v %v", exists, err)
	}
}
