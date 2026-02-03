package db

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mattboback/stageflow/packages/shared-go/models"
)

func TestDatabaseOperations_RespectCanceledContext(t *testing.T) {
	db := setupTestDB(t)

	job := &models.Job{
		ID:        "job-context-cancel",
		State:     models.JobStatePending,
		InputType: "zip",
		InputPath: "test.zip",
		Config:    models.JobConfig{Modules: []string{"axe"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mustCreateJob(t, db, job)

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := db.GetJob(canceledCtx, job.ID); !errors.Is(err, context.Canceled) {
		t.Fatalf("GetJob should respect canceled context, got: %v", err)
	}

	if _, err := db.ListJobs(canceledCtx, ListJobsOptions{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("ListJobs should respect canceled context, got: %v", err)
	}

	if err := db.UpdateJobState(canceledCtx, job.ID, models.JobStateScanning); !errors.Is(err, context.Canceled) {
		t.Fatalf("UpdateJobState should respect canceled context, got: %v", err)
	}

	currentJob, err := db.GetJob(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("failed to fetch job after canceled update: %v", err)
	}
	if currentJob.State != models.JobStatePending {
		t.Fatalf("expected job state to remain %s, got %s", models.JobStatePending, currentJob.State)
	}
}

func TestDatabaseOperations_RespectExpiredDeadline(t *testing.T) {
	db := setupTestDB(t)

	expiredCtx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	job := &models.Job{
		ID:        "job-context-deadline",
		State:     models.JobStatePending,
		InputType: "zip",
		InputPath: "test.zip",
		Config:    models.JobConfig{Modules: []string{"axe"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := db.CreateJob(expiredCtx, job); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("CreateJob should respect expired deadline, got: %v", err)
	}

	if err := db.InsertJobEvent(expiredCtx, &JobEventInsert{
		JobID:     "job-context-deadline",
		Event:     "job.created",
		Timestamp: time.Now().UTC(),
		Payload:   "{}",
	}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("InsertJobEvent should respect expired deadline, got: %v", err)
	}
}
