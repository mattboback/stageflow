package db

import (
	"context"
	"testing"
	"time"

	"github.com/mattboback/stageflow/packages/shared-go/models"
)

func TestCreateJob(t *testing.T) {
	db := setupTestDB(t)

	job := &models.Job{
		ID:        "job-123",
		State:     models.JobStatePending,
		InputType: "zip",
		InputPath: "staging/job-123/test.zip",
		Config: models.JobConfig{
			Modules:    []string{"axe", "keyboard"},
			Screenshot: true,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mustCreateJob(t, db, job)

	retrieved, err := db.GetJob(context.Background(), "job-123")
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	if retrieved.ID != job.ID {
		t.Errorf("Expected ID %s, got %s", job.ID, retrieved.ID)
	}
	if retrieved.State != job.State {
		t.Errorf("Expected state %s, got %s", job.State, retrieved.State)
	}
}

func TestCreateJobWithURLs(t *testing.T) {
	db := setupTestDB(t)

	urls := []string{"https://example.com", "https://example.org"}
	job := &models.Job{
		ID:        "job-urls",
		State:     models.JobStatePending,
		InputType: "urls",
		InputPath: "", // not used for urls jobs
		URLs:      urls,
		Config:    models.JobConfig{Modules: []string{"axe"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mustCreateJob(t, db, job)

	retrieved, err := db.GetJob(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	if got := len(retrieved.URLs); got != len(urls) {
		t.Fatalf("expected %d URLs, got %d", len(urls), got)
	}
	for i, u := range urls {
		if retrieved.URLs[i] != u {
			t.Errorf("URL %d mismatch: expected %s, got %s", i, u, retrieved.URLs[i])
		}
	}
}

func TestGetJobNotFound(t *testing.T) {
	db := setupTestDB(t)

	_, err := db.GetJob(context.Background(), "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent job")
	}
}
