package db

import (
	"context"
	"testing"
	"time"

	"github.com/mattboback/stageflow/packages/shared-go/models"
)

func TestUpdateJobState(t *testing.T) {
	db := setupTestDB(t)

	job := &models.Job{
		ID:        "job-123",
		State:     models.JobStatePending,
		InputType: "zip",
		InputPath: "test.zip",
		Config:    models.JobConfig{Modules: []string{"axe"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mustCreateJob(t, db, job)

	err := db.UpdateJobState(context.Background(), "job-123", models.JobStateExtracting)
	if err != nil {
		t.Fatalf("Failed to update job state: %v", err)
	}

	retrieved, err := db.GetJob(context.Background(), "job-123")
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	if retrieved.State != models.JobStateExtracting {
		t.Errorf("Expected state EXTRACTING, got %s", retrieved.State)
	}
}

func TestUpdateJobPodID(t *testing.T) {
	db := setupTestDB(t)

	job := &models.Job{
		ID:        "job-123",
		State:     models.JobStatePending,
		InputType: "zip",
		InputPath: "test.zip",
		Config:    models.JobConfig{Modules: []string{"axe"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mustCreateJob(t, db, job)

	err := db.UpdateJobPodID(context.Background(), "job-123", "pod-456")
	if err != nil {
		t.Fatalf("Failed to update job pod ID: %v", err)
	}

	retrieved, err := db.GetJob(context.Background(), "job-123")
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	if retrieved.PodID != "pod-456" {
		t.Errorf("Expected pod ID pod-456, got %s", retrieved.PodID)
	}
}

func TestCompleteJob(t *testing.T) {
	db := setupTestDB(t)

	job := &models.Job{
		ID:        "job-123",
		State:     models.JobStateScanning,
		InputType: "zip",
		InputPath: "test.zip",
		Config:    models.JobConfig{Modules: []string{"axe"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mustCreateJob(t, db, job)

	err := db.CompleteJob(context.Background(), "job-123")
	if err != nil {
		t.Fatalf("Failed to complete job: %v", err)
	}

	retrieved, err := db.GetJob(context.Background(), "job-123")
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	if retrieved.State != models.JobStateDone {
		t.Errorf("Expected state DONE, got %s", retrieved.State)
	}

	if retrieved.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set")
	}
}

func TestFailJob(t *testing.T) {
	db := setupTestDB(t)

	job := &models.Job{
		ID:        "job-123",
		State:     models.JobStateScanning,
		InputType: "zip",
		InputPath: "test.zip",
		Config:    models.JobConfig{Modules: []string{"axe"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mustCreateJob(t, db, job)

	err := db.FailJob(
		context.Background(),
		"job-123",
		"scanning",
		"Test error message",
		"Scanner crashed",
	)
	if err != nil {
		t.Fatalf("Failed to fail job: %v", err)
	}

	retrieved, err := db.GetJob(context.Background(), "job-123")
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	if retrieved.State != models.JobStateFailed {
		t.Errorf("Expected state FAILED, got %s", retrieved.State)
	}

	if retrieved.Error != "Test error message" {
		t.Errorf("Expected error message, got %s", retrieved.Error)
	}

	if retrieved.LastStage != "scanning" {
		t.Errorf("Expected stage scanning, got %s", retrieved.LastStage)
	}

	if retrieved.ErrorDetails != "Scanner crashed" {
		t.Errorf("Expected error details, got %s", retrieved.ErrorDetails)
	}

	if retrieved.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set")
	}
}

func TestListJobsByState(t *testing.T) {
	db := setupTestDB(t)

	jobs := []*models.Job{
		{
			ID:        "job-1",
			State:     models.JobStatePending,
			InputType: "zip",
			InputPath: "test1.zip",
			Config:    models.JobConfig{Modules: []string{"axe"}},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "job-2",
			State:     models.JobStatePending,
			InputType: "zip",
			InputPath: "test2.zip",
			Config:    models.JobConfig{Modules: []string{"axe"}},
			CreatedAt: time.Now().Add(1 * time.Second),
			UpdatedAt: time.Now().Add(1 * time.Second),
		},
		{
			ID:        "job-3",
			State:     models.JobStateScanning,
			InputType: "zip",
			InputPath: "test3.zip",
			Config:    models.JobConfig{Modules: []string{"axe"}},
			CreatedAt: time.Now().Add(2 * time.Second),
			UpdatedAt: time.Now().Add(2 * time.Second),
		},
	}

	for _, job := range jobs {
		mustCreateJob(t, db, job)
	}

	pendingJobs, err := db.ListJobsByState(context.Background(), models.JobStatePending)
	if err != nil {
		t.Fatalf("Failed to list jobs: %v", err)
	}

	if len(pendingJobs) != 2 {
		t.Errorf("Expected 2 pending jobs, got %d", len(pendingJobs))
	}

	scanningJobs, err := db.ListJobsByState(context.Background(), models.JobStateScanning)
	if err != nil {
		t.Fatalf("Failed to list scanning jobs: %v", err)
	}

	if len(scanningJobs) != 1 {
		t.Errorf("Expected 1 scanning job, got %d", len(scanningJobs))
	}
}
